use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;
use agentsmesh_types::proto_pod_v1 as pod_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in backend/internal/api/connect/pod/.
// Procedure paths derive from `proto.pod.v1.PodService.<Method>` (§12).

impl ApiClient {
    pub async fn list_pods_connect(
        &self,
        req: &pod_proto::ListPodsRequest,
    ) -> Result<pod_proto::ListPodsResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/ListPods", req).await
    }

    pub async fn get_pod_connect(
        &self,
        req: &pod_proto::GetPodRequest,
    ) -> Result<pod_proto::Pod, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/GetPod", req).await
    }

    pub async fn create_pod_connect(
        &self,
        req: &pod_proto::CreatePodRequest,
    ) -> Result<pod_proto::CreatePodResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/CreatePod", req).await
    }

    pub async fn terminate_pod_connect(
        &self,
        req: &pod_proto::TerminatePodRequest,
    ) -> Result<pod_proto::TerminatePodResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/TerminatePod", req).await
    }

    pub async fn update_pod_alias_connect(
        &self,
        req: &pod_proto::UpdatePodAliasRequest,
    ) -> Result<pod_proto::UpdatePodAliasResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/UpdatePodAlias", req).await
    }

    pub async fn update_pod_perpetual_connect(
        &self,
        req: &pod_proto::UpdatePodPerpetualRequest,
    ) -> Result<pod_proto::UpdatePodPerpetualResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/UpdatePodPerpetual", req).await
    }

    pub async fn get_pod_connection_connect(
        &self,
        req: &pod_proto::GetPodConnectionRequest,
    ) -> Result<pod_proto::PodConnectionInfo, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/GetPodConnection", req).await
    }

    pub async fn send_pod_prompt_connect(
        &self,
        req: &pod_proto::SendPodPromptRequest,
    ) -> Result<pod_proto::SendPodPromptResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/SendPodPrompt", req).await
    }

    pub async fn list_pods_by_ticket_connect(
        &self,
        req: &pod_proto::ListPodsByTicketRequest,
    ) -> Result<pod_proto::ListPodsByTicketResponse, ApiError> {
        connect_call(self, "/proto.pod.v1.PodService/ListPodsByTicket", req).await
    }
}

// =============================================================================
// Legacy REST methods — preserved for dual-track migration.
// =============================================================================

impl ApiClient {
    pub async fn list_pods(
        &self,
        status: Option<&str>,
        runner_id: Option<i64>,
        created_by_id: Option<i64>,
        limit: Option<i64>,
        offset: Option<i64>,
    ) -> Result<PodListResponse, ApiError> {
        let mut path = self.org_path("/pods");
        let mut params = Vec::new();
        if let Some(v) = status {
            params.push(format!("status={v}"));
        }
        if let Some(v) = runner_id {
            params.push(format!("runner_id={v}"));
        }
        if let Some(v) = created_by_id {
            params.push(format!("created_by_id={v}"));
        }
        if let Some(v) = limit {
            params.push(format!("limit={v}"));
        }
        if let Some(v) = offset {
            params.push(format!("offset={v}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn get_pod(&self, key: &str) -> Result<Pod, ApiError> {
        self.get_resource(&self.org_path(&format!("/pods/{key}")), "pod").await
    }

    pub async fn create_pod(&self, data: &CreatePodRequest) -> Result<CreatePodResponse, ApiError> {
        self.post(&self.org_path("/pods"), data).await
    }

    pub async fn terminate_pod(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/pods/{key}/terminate")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn get_pod_connection_info(
        &self,
        key: &str,
    ) -> Result<PodConnectionInfo, ApiError> {
        self.get(&self.org_path(&format!("/pods/{key}/connect")))
            .await
    }

    pub async fn get_pod_relay_connection(
        &self,
        key: &str,
    ) -> Result<PodConnectionInfo, ApiError> {
        self.get(&self.org_path(&format!("/pods/{key}/relay/connect")))
            .await
    }

    pub async fn update_pod_alias(
        &self,
        key: &str,
        data: &UpdatePodAliasRequest,
    ) -> Result<Pod, ApiError> {
        self.patch_resource(&self.org_path(&format!("/pods/{key}/alias")), data, "pod").await
    }
}
