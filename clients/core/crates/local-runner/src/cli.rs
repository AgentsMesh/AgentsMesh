use crate::error::{LocalRunnerError, Result};
use crate::paths::InstallPaths;
use std::path::PathBuf;
use std::process::Command;
use tokio::task;

// std::process on a blocking task instead of tokio's `process` feature —
// workspace excludes the latter for iOS compat.
pub(crate) async fn run_cli(binary: PathBuf, args: Vec<String>) -> Result<CliOutput> {
    if !binary.exists() {
        return Err(LocalRunnerError::BinaryNotFound {
            path: binary.display().to_string(),
        });
    }
    task::spawn_blocking(move || {
        let output = Command::new(&binary).args(&args).output()?;
        Ok(CliOutput {
            status: output.status.code().unwrap_or(-1),
            stdout: String::from_utf8_lossy(&output.stdout).into_owned(),
            stderr: String::from_utf8_lossy(&output.stderr).into_owned(),
        })
    })
    .await
    .map_err(|e| LocalRunnerError::Io(std::io::Error::other(e.to_string())))?
}

#[derive(Debug)]
pub(crate) struct CliOutput {
    pub status: i32,
    pub stdout: String,
    pub stderr: String,
}

impl CliOutput {
    pub fn ok(self) -> Result<String> {
        if self.status == 0 {
            Ok(self.stdout)
        } else {
            Err(LocalRunnerError::CliFailed {
                status: self.status,
                stderr: self.stderr,
            })
        }
    }
}

pub(crate) async fn run_runner_cli(paths: &InstallPaths, args: &[&str]) -> Result<CliOutput> {
    let owned: Vec<String> = args.iter().map(|s| s.to_string()).collect();
    run_cli(paths.binary_path(), owned).await
}
