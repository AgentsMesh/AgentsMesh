mod cli;
mod error;
mod install;
mod paths;
mod register;
mod service;

pub use error::{LocalRunnerError, Result};
pub use paths::InstallPaths;
pub use service::ServiceStatus;

/// Bundled runner version used when the backend's `latest-release` endpoint
/// is unreachable. Keep in sync with
/// `backend/internal/api/rest/v1/runners_release.go` — bumped per desktop
/// release. Source: latest tag at github.com/AgentsMesh/AgentsMesh.
pub const FALLBACK_RUNNER_VERSION: &str = "0.29.0";

use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct LocalRunnerManager {
    paths: InstallPaths,
    server_url: String,
}

impl LocalRunnerManager {
    pub fn new(home: PathBuf, server_url: String) -> Self {
        Self {
            paths: InstallPaths::new(home),
            server_url,
        }
    }

    pub fn from_default_home(server_url: String) -> Self {
        let home = dirs::home_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::new(home, server_url)
    }

    pub fn binary_path(&self) -> PathBuf {
        self.paths.binary_path()
    }

    pub fn host_target(&self) -> Option<String> {
        let os = match std::env::consts::OS {
            "macos" => "darwin",
            "linux" => "linux",
            "windows" => "windows",
            _ => return None,
        };
        let arch = match std::env::consts::ARCH {
            "x86_64" => "amd64",
            "aarch64" => "arm64",
            _ => return None,
        };
        Some(format!("{os}_{arch}"))
    }

    pub async fn is_installed(&self) -> bool {
        install::is_installed(&self.paths).await
    }

    pub async fn installed_version(&self) -> Option<String> {
        install::installed_version(&self.paths).await
    }

    pub async fn install_binary(
        &self,
        release_url: &str,
        expected_sha256: Option<&str>,
    ) -> Result<()> {
        install::install_binary(&self.paths, release_url, expected_sha256).await
    }

    pub async fn is_registered(&self) -> bool {
        register::is_registered(&self.paths).await
    }

    pub async fn local_node_id(&self) -> Option<String> {
        register::local_node_id(&self.paths).await
    }

    pub async fn register(&self, token: &str) -> Result<()> {
        register::register(&self.paths, token, &self.server_url).await
    }

    pub async fn service_install(&self) -> Result<()> {
        service::install(&self.paths).await
    }

    pub async fn service_uninstall(&self) -> Result<()> {
        service::uninstall(&self.paths).await
    }

    pub async fn service_start(&self) -> Result<()> {
        service::start(&self.paths).await
    }

    pub async fn service_stop(&self) -> Result<()> {
        service::stop(&self.paths).await
    }

    pub async fn service_status(&self) -> Result<ServiceStatus> {
        service::status(&self.paths).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn binary_path_under_dot_agentsmesh() {
        let mgr = LocalRunnerManager::new(
            PathBuf::from("/tmp/fake-home"),
            "https://example.com".into(),
        );
        let path = mgr.binary_path();
        assert!(path.starts_with("/tmp/fake-home/.agentsmesh/bin"));
    }

    #[tokio::test]
    async fn is_installed_reports_false_for_missing_binary() {
        let mgr = LocalRunnerManager::new(
            PathBuf::from("/nonexistent-home-for-tests"),
            String::new(),
        );
        assert!(!mgr.is_installed().await);
    }

    #[tokio::test]
    async fn host_target_matches_release_naming() {
        let mgr = LocalRunnerManager::new(PathBuf::from("/tmp"), String::new());
        let target = mgr.host_target().expect("supported host");
        assert!(
            matches!(
                target.as_str(),
                "darwin_amd64" | "darwin_arm64" | "linux_amd64" | "linux_arm64" | "windows_amd64" | "windows_arm64"
            ),
            "unexpected host target: {target}"
        );
    }

    #[tokio::test]
    async fn register_rejects_empty_token() {
        let mgr = LocalRunnerManager::new(PathBuf::from("/tmp"), String::new());
        let err = mgr.register("").await.unwrap_err();
        assert!(matches!(err, LocalRunnerError::InvalidArgument(_)));
    }
}
