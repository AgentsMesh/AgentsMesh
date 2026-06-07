use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_ticket_relations_v1 as tr_proto;
use prost::Message;

pub struct TicketRelationsService {
    client: Arc<ApiClient>,
}

// Connect-RPC binary wire. See proto-naming-conventions.md §2.5.
// Each `*_connect` method takes prost-encoded bytes and returns prost-encoded
// bytes — matching the wasm bridge's `Result<Vec<u8>, String>` surface. Caller
// (TS) encodes via @bufbuild/protobuf `.toBinary()` and decodes via
// `.fromBinary()`.
impl TicketRelationsService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list_relations_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::ListRelationsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_relations request: {e}"))?;
        tracing::debug!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "list relations");
        let resp = self.client.list_relations_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_relation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::CreateRelationRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_relation request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, target_slug = %req.target_slug, relation_type = %req.relation_type, "create relation");
        let resp = self.client.create_relation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_relation_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::DeleteRelationRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_relation request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, relation_id = req.relation_id, "delete relation");
        let resp = self.client.delete_relation_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_merge_requests_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::ListMergeRequestsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_merge_requests request: {e}"))?;
        tracing::debug!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "list merge requests");
        let resp = self.client.list_ticket_merge_requests_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_commits_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::ListCommitsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_commits request: {e}"))?;
        tracing::debug!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "list commits");
        let resp = self.client.list_ticket_commits_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn link_commit_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::LinkCommitRequest::decode(request_bytes)
            .map_err(|e| format!("decode link_commit request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "link commit");
        let resp = self.client.link_commit_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn unlink_commit_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::UnlinkCommitRequest::decode(request_bytes)
            .map_err(|e| format!("decode unlink_commit request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, commit_id = req.commit_id, "unlink commit");
        let resp = self.client.unlink_commit_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_comments_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::ListCommentsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_comments request: {e}"))?;
        tracing::debug!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "list comments");
        let resp = self.client.list_comments_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_comment_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::CreateCommentRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_comment request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, "create comment");
        let resp = self.client.create_comment_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_comment_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::UpdateCommentRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_comment request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, comment_id = req.comment_id, "update comment");
        let resp = self.client.update_comment_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_comment_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = tr_proto::DeleteCommentRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_comment request: {e}"))?;
        tracing::info!(target: "ticket_relations", org_slug = %req.org_slug, ticket_slug = %req.ticket_slug, comment_id = req.comment_id, "delete comment");
        let resp = self.client.delete_comment_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
