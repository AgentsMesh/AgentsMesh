use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_invitation_v1 as inv_proto;
use prost::Message;

pub struct InvitationService {
    client: Arc<ApiClient>,
}

impl InvitationService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes via @bufbuild/protobuf .toBinary() → wasm bridge → here →
    // ApiClient.*_connect (binary in / binary out, conventions §2.5). No
    // JSON path on the client.
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes — matching the wasm bridge's `Result<Vec<u8>, String>`
    // surface (conventions §2.5). The TS adapter populates org_slug / token
    // before encoding.

    pub async fn list_invitations_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::ListInvitationsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_invitations request: {e}"))?;
        tracing::debug!(target: "invitation", org_slug = %req.org_slug, "list invitations");
        let resp = self.client.list_invitations_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_invitation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::CreateInvitationRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_invitation request: {e}"))?;
        tracing::info!(target: "invitation", org_slug = %req.org_slug, role = %req.role, "create invitation");
        let resp = self.client.create_invitation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn revoke_invitation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::RevokeInvitationRequest::decode(request_bytes)
            .map_err(|e| format!("decode revoke_invitation request: {e}"))?;
        tracing::info!(target: "invitation", org_slug = %req.org_slug, invitation_id = req.id, "revoke invitation");
        let resp = self.client.revoke_invitation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn resend_invitation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::ResendInvitationRequest::decode(request_bytes)
            .map_err(|e| format!("decode resend_invitation request: {e}"))?;
        tracing::info!(target: "invitation", org_slug = %req.org_slug, invitation_id = req.id, "resend invitation");
        let resp = self.client.resend_invitation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn accept_invitation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::AcceptInvitationRequest::decode(request_bytes)
            .map_err(|e| format!("decode accept_invitation request: {e}"))?;
        tracing::info!(target: "invitation", "accept invitation");
        let resp = self.client.accept_invitation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_pending_invitations_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::ListPendingInvitationsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_pending_invitations request: {e}"))?;
        tracing::debug!(target: "invitation", "list pending invitations");
        let resp = self.client.list_pending_invitations_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_invitation_by_token_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = inv_proto::GetInvitationByTokenRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_invitation_by_token request: {e}"))?;
        tracing::debug!(target: "invitation", "get invitation by token");
        let resp = self.client.get_invitation_by_token_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
