use crate::core::AgentsMeshCore;
use crate::dto::{
    update_pod_alias_req, CreatePodRequestDto, CreatePodResponseDto, PodConnectionInfoDto, PodDto,
    PodListResponseDto,
};
use crate::error::CoreError;

/// Pod lifecycle — list/create/terminate/alias, plus relay connection info
/// for terminal attach. Input/resize/signals go via the relay WebSocket
/// (Swift side uses `relay_encode_*` + SwiftTerm delegate, not through FFI).
#[uniffi::export(async_runtime = "tokio")]
impl AgentsMeshCore {
    pub async fn list_pods(
        &self,
        status: Option<String>,
        runner_id: Option<i64>,
        created_by_id: Option<i64>,
        limit: Option<i64>,
        offset: Option<i64>,
    ) -> Result<PodListResponseDto, CoreError> {
        let resp = self
            .api
            .list_pods(status.as_deref(), runner_id, created_by_id, limit, offset)
            .await?;
        Ok(resp.into())
    }

    pub async fn get_pod(&self, pod_key: String) -> Result<PodDto, CoreError> {
        let pod = self.api.get_pod(&pod_key).await?;
        Ok(pod.into())
    }

    pub async fn create_pod(&self, req: CreatePodRequestDto) -> Result<CreatePodResponseDto, CoreError> {
        let resp = self.api.create_pod(&req.into()).await?;
        Ok(resp.into())
    }

    pub async fn terminate_pod(&self, pod_key: String) -> Result<(), CoreError> {
        self.api.terminate_pod(&pod_key).await?;
        Ok(())
    }

    pub async fn update_pod_alias(
        &self,
        pod_key: String,
        alias: Option<String>,
    ) -> Result<PodDto, CoreError> {
        let req = update_pod_alias_req(alias);
        let pod = self.api.update_pod_alias(&pod_key, &req).await?;
        Ok(pod.into())
    }

    /// Relay WS connection info (direct, no relay hop). Used for local
    /// runners where the browser/device can reach the runner directly.
    pub async fn get_pod_connection_info(
        &self,
        pod_key: String,
    ) -> Result<PodConnectionInfoDto, CoreError> {
        let info = self.api.get_pod_connection_info(&pod_key).await?;
        Ok(info.into())
    }

    /// Relay WS connection info through the relay server. Used for remote
    /// runners behind NAT where relay terminates both ends.
    pub async fn get_pod_relay_connection(
        &self,
        pod_key: String,
    ) -> Result<PodConnectionInfoDto, CoreError> {
        let info = self.api.get_pod_relay_connection(&pod_key).await?;
        Ok(info.into())
    }
}
