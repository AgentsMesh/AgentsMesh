use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_org_v1 as org_proto;
use prost::Message;

pub struct OrgApiService {
    client: Arc<ApiClient>,
}

impl OrgApiService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes via @bufbuild/protobuf .toBinary() → wasm bridge → here →
    // ApiClient.*_connect (binary in / binary out, conventions §2.5). No
    // JSON path.
    //
    // org_slug is sourced from the caller-supplied request, not from
    // AuthManager — keeps these methods unit-testable without an org context
    // in the token store. The TS adapter populates org_slug before encoding.

    pub async fn list_my_orgs_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::ListMyOrgsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_my_orgs request: {e}"))?;
        tracing::debug!(target: "org", "list my orgs");
        let resp = self.client.list_my_orgs_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_org_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::CreateOrgRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_org request: {e}"))?;
        tracing::info!(target: "org", slug = %req.slug, "create org");
        let resp = self.client.create_org_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_personal_org_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::CreatePersonalOrgRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_personal_org request: {e}"))?;
        tracing::info!(target: "org", "create personal org");
        let resp = self.client.create_personal_org_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_org_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::GetOrgRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_org request: {e}"))?;
        tracing::debug!(target: "org", org_slug = %req.org_slug, "get org");
        let resp = self.client.get_org_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_org_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::UpdateOrgRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_org request: {e}"))?;
        tracing::info!(target: "org", org_slug = %req.org_slug, "update org");
        let resp = self.client.update_org_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_org_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::DeleteOrgRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_org request: {e}"))?;
        tracing::info!(target: "org", org_slug = %req.org_slug, "delete org");
        let resp = self.client.delete_org_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_members_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::ListMembersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_members request: {e}"))?;
        tracing::debug!(target: "org", org_slug = %req.org_slug, "list members");
        let resp = self.client.list_members_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn invite_member_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::InviteMemberRequest::decode(request_bytes)
            .map_err(|e| format!("decode invite_member request: {e}"))?;
        tracing::info!(target: "org", org_slug = %req.org_slug, user_id = req.user_id, "invite member");
        let resp = self.client.invite_member_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn remove_member_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::RemoveMemberRequest::decode(request_bytes)
            .map_err(|e| format!("decode remove_member request: {e}"))?;
        tracing::info!(target: "org", org_slug = %req.org_slug, user_id = req.user_id, "remove member");
        let resp = self.client.remove_member_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_member_role_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = org_proto::UpdateMemberRoleRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_member_role request: {e}"))?;
        tracing::info!(target: "org", org_slug = %req.org_slug, user_id = req.user_id, "update member role");
        let resp = self.client.update_member_role_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
