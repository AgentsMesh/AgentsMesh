use crate::cli::run_runner_cli;
use crate::error::{LocalRunnerError, Result};
use crate::paths::InstallPaths;
use sha2::{Digest, Sha256};
use std::io::{Cursor, Read};
use std::path::Path;
use tokio::task;

pub async fn is_installed(paths: &InstallPaths) -> bool {
    tokio::fs::metadata(paths.binary_path()).await.is_ok()
}

pub async fn installed_version(paths: &InstallPaths) -> Option<String> {
    let output = run_runner_cli(paths, &["version"]).await.ok()?;
    if output.status != 0 {
        return None;
    }
    parse_version(&output.stdout)
}

fn parse_version(stdout: &str) -> Option<String> {
    let trimmed = stdout.trim();
    let prefix = "AgentsMesh Runner ";
    let rest = trimmed.strip_prefix(prefix)?;
    let version = rest.split_whitespace().next()?;
    Some(version.to_string())
}

pub async fn install_binary(
    paths: &InstallPaths,
    release_url: &str,
    expected_sha256: Option<&str>,
) -> Result<()> {
    if release_url.is_empty() {
        return Err(LocalRunnerError::InvalidArgument(
            "release_url must not be empty".into(),
        ));
    }

    let body = download_with_optional_verify(release_url, expected_sha256).await?;

    let install_dir = paths.install_dir().to_path_buf();
    let binary_path = paths.binary_path();
    let url_owned = release_url.to_string();

    task::spawn_blocking(move || -> Result<()> {
        std::fs::create_dir_all(&install_dir)?;
        let bytes = extract_runner_bytes(&url_owned, &body)?;
        atomic_write_executable(&binary_path, &bytes)
    })
    .await
    .map_err(|e| LocalRunnerError::Io(std::io::Error::other(e.to_string())))?
}

async fn download_with_optional_verify(
    url: &str,
    expected_sha256: Option<&str>,
) -> Result<Vec<u8>> {
    let mut resp = reqwest::get(url)
        .await
        .map_err(|e| LocalRunnerError::Download(e.to_string()))?;
    if !resp.status().is_success() {
        return Err(LocalRunnerError::Download(format!(
            "GET {url} returned {}",
            resp.status()
        )));
    }

    let cap = resp.content_length().unwrap_or(0) as usize;
    let mut hasher = Sha256::new();
    let mut buf = Vec::with_capacity(cap);
    while let Some(chunk) = resp
        .chunk()
        .await
        .map_err(|e| LocalRunnerError::Download(e.to_string()))?
    {
        hasher.update(&chunk);
        buf.extend_from_slice(&chunk);
    }

    if let Some(expected) = expected_sha256 {
        verify_hash(hasher, expected)?;
    }

    Ok(buf)
}

fn verify_hash(hasher: Sha256, expected_hex: &str) -> Result<()> {
    let actual = hex::encode(hasher.finalize());
    let expected = expected_hex.trim().to_ascii_lowercase();
    if actual != expected {
        return Err(LocalRunnerError::ChecksumMismatch { expected, actual });
    }
    Ok(())
}

fn extract_runner_bytes(url: &str, archive: &[u8]) -> Result<Vec<u8>> {
    if url.ends_with(".zip") {
        extract_from_zip(archive)
    } else if url.ends_with(".tar.gz") || url.ends_with(".tgz") {
        extract_from_tar_gz(archive)
    } else {
        Err(LocalRunnerError::Extract(format!(
            "unsupported archive extension in {url}"
        )))
    }
}

fn extract_from_tar_gz(archive: &[u8]) -> Result<Vec<u8>> {
    let dec = flate2::read::GzDecoder::new(Cursor::new(archive));
    let mut tar = tar::Archive::new(dec);
    for entry in tar
        .entries()
        .map_err(|e| LocalRunnerError::Extract(e.to_string()))?
    {
        let mut entry = entry.map_err(|e| LocalRunnerError::Extract(e.to_string()))?;
        let path = entry
            .path()
            .map_err(|e| LocalRunnerError::Extract(e.to_string()))?
            .into_owned();
        if is_runner_entry(&path) {
            let mut buf = Vec::new();
            entry
                .read_to_end(&mut buf)
                .map_err(|e| LocalRunnerError::Extract(e.to_string()))?;
            return Ok(buf);
        }
    }
    Err(LocalRunnerError::BinaryMissingFromArchive)
}

fn extract_from_zip(archive: &[u8]) -> Result<Vec<u8>> {
    let mut zip = zip::ZipArchive::new(Cursor::new(archive))
        .map_err(|e| LocalRunnerError::Extract(e.to_string()))?;
    for i in 0..zip.len() {
        let mut entry = zip
            .by_index(i)
            .map_err(|e| LocalRunnerError::Extract(e.to_string()))?;
        let name = entry.name().to_string();
        if is_runner_entry(Path::new(&name)) {
            let mut buf = Vec::new();
            entry
                .read_to_end(&mut buf)
                .map_err(|e| LocalRunnerError::Extract(e.to_string()))?;
            return Ok(buf);
        }
    }
    Err(LocalRunnerError::BinaryMissingFromArchive)
}

fn is_runner_entry(path: &Path) -> bool {
    let name = match path.file_name().and_then(|s| s.to_str()) {
        Some(n) => n,
        None => return false,
    };
    name == "agentsmesh-runner" || name == "agentsmesh-runner.exe"
}

fn atomic_write_executable(target: &Path, bytes: &[u8]) -> Result<()> {
    let parent = target
        .parent()
        .ok_or_else(|| LocalRunnerError::InvalidArgument("binary path has no parent".into()))?;
    let staging = parent.join(format!(
        ".{}.tmp",
        target
            .file_name()
            .and_then(|s| s.to_str())
            .unwrap_or("agentsmesh-runner")
    ));
    std::fs::write(&staging, bytes)?;
    set_executable(&staging)?;
    if target.exists() {
        std::fs::remove_file(target)?;
    }
    std::fs::rename(&staging, target)?;
    Ok(())
}

#[cfg(unix)]
fn set_executable(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    let mut perm = std::fs::metadata(path)?.permissions();
    perm.set_mode(0o755);
    std::fs::set_permissions(path, perm)?;
    Ok(())
}

#[cfg(not(unix))]
fn set_executable(_path: &Path) -> Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use flate2::Compression;
    use flate2::write::GzEncoder;

    #[test]
    fn parses_version_output() {
        let stdout = "AgentsMesh Runner 0.4.7 (built 2026-04-15T10:00:00Z)\n";
        assert_eq!(parse_version(stdout).as_deref(), Some("0.4.7"));
    }

    #[test]
    fn rejects_unrecognised_version_output() {
        assert!(parse_version("unrelated output").is_none());
        assert!(parse_version("").is_none());
    }

    #[test]
    fn checksum_mismatch_is_detected() {
        let mut h = Sha256::new();
        h.update(b"hello");
        let err = verify_hash(h, "deadbeef").unwrap_err();
        assert!(matches!(err, LocalRunnerError::ChecksumMismatch { .. }));
    }

    #[test]
    fn checksum_matches_when_correct() {
        let mut h = Sha256::new();
        h.update(b"hello");
        let mut hasher_for_hex = Sha256::new();
        hasher_for_hex.update(b"hello");
        let hex = hex::encode(hasher_for_hex.finalize());
        verify_hash(h, &hex).unwrap();
    }

    #[test]
    fn extracts_runner_from_targz() {
        let archive = make_tar_gz(&[("agentsmesh-runner", b"runner-bytes" as &[u8])]);
        let bytes = extract_runner_bytes("foo.tar.gz", &archive).unwrap();
        assert_eq!(bytes, b"runner-bytes");
    }

    #[test]
    fn ignores_unrelated_entries_in_targz() {
        let archive = make_tar_gz(&[
            ("README.md", b"docs" as &[u8]),
            ("agentsmesh-runner", b"runner-bytes"),
            ("LICENSE", b"text"),
        ]);
        let bytes = extract_runner_bytes("foo.tar.gz", &archive).unwrap();
        assert_eq!(bytes, b"runner-bytes");
    }

    #[test]
    fn rejects_archive_without_runner() {
        let archive = make_tar_gz(&[("README.md", b"docs" as &[u8])]);
        let err = extract_runner_bytes("foo.tar.gz", &archive).unwrap_err();
        assert!(matches!(err, LocalRunnerError::BinaryMissingFromArchive));
    }

    #[test]
    fn rejects_unknown_archive_extension() {
        let err = extract_runner_bytes("foo.bin", b"x").unwrap_err();
        assert!(matches!(err, LocalRunnerError::Extract(_)));
    }

    #[test]
    fn install_binary_rejects_empty_url() {
        let rt = tokio::runtime::Runtime::new().unwrap();
        let paths = InstallPaths::new("/tmp/local-runner-test-empty");
        let err = rt.block_on(install_binary(&paths, "", None)).unwrap_err();
        assert!(matches!(err, LocalRunnerError::InvalidArgument(_)));
    }

    fn make_tar_gz(entries: &[(&str, &[u8])]) -> Vec<u8> {
        let buf = Vec::new();
        let enc = GzEncoder::new(buf, Compression::default());
        let mut builder = tar::Builder::new(enc);
        for (name, data) in entries {
            let mut header = tar::Header::new_gnu();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            builder.append_data(&mut header, name, *data).unwrap();
        }
        builder.into_inner().unwrap().finish().unwrap()
    }
}
