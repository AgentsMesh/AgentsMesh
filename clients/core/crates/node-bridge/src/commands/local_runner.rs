use crate::{AppState, err};
use napi_derive::napi;

#[napi]
impl AppState {
    #[napi]
    pub async fn local_runner_binary_path(&self) -> String {
        self.local_runner.binary_path().display().to_string()
    }

    #[napi]
    pub async fn local_runner_host_target(&self) -> Option<String> {
        self.local_runner.host_target()
    }

    #[napi]
    pub async fn local_runner_fallback_version(&self) -> String {
        agentsmesh_local_runner::FALLBACK_RUNNER_VERSION.to_string()
    }

    #[napi]
    pub async fn local_runner_is_installed(&self) -> bool {
        self.local_runner.is_installed().await
    }

    #[napi]
    pub async fn local_runner_installed_version(&self) -> Option<String> {
        self.local_runner.installed_version().await
    }

    #[napi]
    pub async fn local_runner_install_binary(
        &self,
        release_url: String,
        expected_sha256: Option<String>,
    ) -> napi::Result<()> {
        self.local_runner
            .install_binary(&release_url, expected_sha256.as_deref())
            .await
            .map_err(err)
    }

    #[napi]
    pub async fn local_runner_is_registered(&self) -> bool {
        self.local_runner.is_registered().await
    }

    #[napi]
    pub async fn local_runner_local_node_id(&self) -> Option<String> {
        self.local_runner.local_node_id().await
    }

    #[napi]
    pub async fn local_runner_register(&self, token: String) -> napi::Result<()> {
        self.local_runner.register(&token).await.map_err(err)
    }

    #[napi]
    pub async fn local_runner_service_install(&self) -> napi::Result<()> {
        self.local_runner.service_install().await.map_err(err)
    }

    #[napi]
    pub async fn local_runner_service_uninstall(&self) -> napi::Result<()> {
        self.local_runner.service_uninstall().await.map_err(err)
    }

    #[napi]
    pub async fn local_runner_service_start(&self) -> napi::Result<()> {
        self.local_runner.service_start().await.map_err(err)
    }

    #[napi]
    pub async fn local_runner_service_stop(&self) -> napi::Result<()> {
        self.local_runner.service_stop().await.map_err(err)
    }

    /// Stringly-typed status keeps the IPC contract stable across napi
    /// enum-codegen drift; renderer maps back to a typed enum on the TS side.
    #[napi]
    pub async fn local_runner_service_status(&self) -> napi::Result<String> {
        let status = self.local_runner.service_status().await.map_err(err)?;
        Ok(match status {
            agentsmesh_local_runner::ServiceStatus::Running => "running",
            agentsmesh_local_runner::ServiceStatus::Stopped => "stopped",
            agentsmesh_local_runner::ServiceStatus::Unknown => "unknown",
            agentsmesh_local_runner::ServiceStatus::NotInstalled => "not_installed",
            agentsmesh_local_runner::ServiceStatus::Stale => "stale",
        }
        .to_string())
    }
}
