use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_ticket_v1 as ticket_proto;
use prost::Message;

// Networking-only service for the ticket domain. The ticket cache (tickets,
// board, labels, ticket→pods) lives in the shared `AppState.tickets` (reached
// via `WasmTicketState`) — this service speaks only the Connect-RPC wire.
pub struct TicketService {
    client: Arc<ApiClient>,
}

impl TicketService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn get_ticket_pods(
        &self, slug: &str, active_only: Option<bool>,
    ) -> Result<String, String> {
        // proto.mesh.v1 owns ticket→pod lookup (MeshService domain). Project
        // each MeshPodNode into proto.pod.v1.Pod for the shared PodState
        // cache (only the channel-surface subset of fields is filled).
        use agentsmesh_types::proto_mesh_v1 as mp;
        use agentsmesh_types::proto_pod_v1::Pod;
        let req = mp::GetTicketPodsRequest {
            org_slug: self.client.current_org_slug(),
            ticket_slug: slug.to_string(),
            active_only,
        };
        tracing::debug!(target: "ticket", ticket_slug = %req.ticket_slug, "get ticket pods");
        let proto_resp = self.client
            .get_ticket_pods_connect(&req)
            .await.map_err(crate::wire)?;
        let pods: Vec<Pod> = proto_resp.pods.iter().map(|n| Pod {
            pod_key: n.pod_key.clone(),
            status: n.status.clone(),
            agent_status: n.agent_status.clone(),
            alias: n.alias.clone(),
            title: n.title.clone(),
            agent_slug: n.agent_slug.clone(),
            runner_id: if n.runner_id == 0 { None } else { Some(n.runner_id) },
            started_at: n.started_at.clone(),
            ..Default::default()
        }).collect();
        let envelope = serde_json::json!({
            "pods": pods,
            "total": serde_json::Value::Null,
            "limit": serde_json::Value::Null,
            "offset": serde_json::Value::Null,
        });
        serde_json::to_string(&envelope).map_err(crate::wire)
    }
}

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// Each `*_connect` method takes prost-encoded bytes and returns
// prost-encoded bytes — matching the wasm bridge's `Result<Vec<u8>, String>`
// surface. Caller (TS) encodes via @bufbuild/protobuf .toBinary() and
// decodes via .fromBinary().

impl TicketService {
    pub async fn list_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::ListTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_tickets request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, "list tickets");
        let resp = self.client.list_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_ticket request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "get ticket");
        let resp = self.client.get_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::CreateTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_ticket request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, repository_id = ?req.repository_id, "create ticket");
        let resp = self.client.create_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_ticket request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "update ticket");
        let resp = self.client.update_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::DeleteTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_ticket request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "delete ticket");
        let resp = self.client.delete_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_ticket_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateTicketStatusRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_ticket_status request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, status = %req.status, "update ticket status");
        let resp = self.client.update_ticket_status_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_active_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetActiveTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_active_tickets request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, "get active tickets");
        let resp = self.client.get_active_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_board_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetBoardRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_board request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, "get board");
        let resp = self.client.get_board_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_sub_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetSubTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_sub_tickets request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "get sub tickets");
        let resp = self.client.get_sub_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn add_assignee_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::AddAssigneeRequest::decode(request_bytes)
            .map_err(|e| format!("decode add_assignee request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, user_id = req.user_id, "add assignee");
        let resp = self.client.add_assignee_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn remove_assignee_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::RemoveAssigneeRequest::decode(request_bytes)
            .map_err(|e| format!("decode remove_assignee request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, user_id = req.user_id, "remove assignee");
        let resp = self.client.remove_assignee_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_labels_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::ListLabelsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_labels request: {e}"))?;
        tracing::debug!(target: "ticket", org_slug = %req.org_slug, "list labels");
        let resp = self.client.list_labels_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::CreateLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_label request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, "create label");
        let resp = self.client.create_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_label request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, label_id = req.id, "update label");
        let resp = self.client.update_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::DeleteLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_label request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, label_id = req.id, "delete label");
        let resp = self.client.delete_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn add_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::AddLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode add_label request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, label_id = req.label_id, "add label");
        let resp = self.client.add_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn remove_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::RemoveLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode remove_label request: {e}"))?;
        tracing::info!(target: "ticket", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, label_id = req.label_id, "remove label");
        let resp = self.client.remove_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
