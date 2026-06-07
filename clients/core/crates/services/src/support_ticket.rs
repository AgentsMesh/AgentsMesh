use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_support_ticket_v1 as st_proto;
use prost::Message;

pub struct SupportTicketService {
    client: Arc<ApiClient>,
}

impl SupportTicketService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // All operations including create / message-reply / attachment flow
    // go through Connect-binary wire (conventions §2.5). Multipart REST is
    // gone — attachments use a 3-step presigned-URL handshake driven by
    // the TS / Swift caller (presign → S3 PUT → associate).

    pub async fn list_support_tickets_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::ListSupportTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_support_tickets request: {e}"))?;
        tracing::debug!(target: "support_ticket", "list support tickets");
        let resp = self.client.list_support_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_support_ticket_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::GetSupportTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_support_ticket request: {e}"))?;
        tracing::debug!(target: "support_ticket", ticket_id = req.id, "get support ticket");
        let resp = self.client.get_support_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_attachment_url_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::GetAttachmentUrlRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_attachment_url request: {e}"))?;
        tracing::debug!(target: "support_ticket", attachment_id = req.attachment_id, "get attachment url");
        let resp = self.client.get_support_ticket_attachment_url_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_support_ticket_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::CreateSupportTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_support_ticket request: {e}"))?;
        tracing::info!(target: "support_ticket", category = %req.category, "create support ticket");
        let resp = self.client.create_support_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn add_support_ticket_message_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::AddSupportTicketMessageRequest::decode(request_bytes)
            .map_err(|e| format!("decode add_support_ticket_message request: {e}"))?;
        tracing::info!(target: "support_ticket", ticket_id = req.ticket_id, "add support ticket message");
        let resp = self.client.add_support_ticket_message_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn presign_attachment_upload_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::PresignAttachmentUploadRequest::decode(request_bytes)
            .map_err(|e| format!("decode presign_attachment_upload request: {e}"))?;
        tracing::info!(target: "support_ticket", ticket_id = req.ticket_id, size = req.size, "presign attachment upload");
        let resp = self.client.presign_support_ticket_attachment_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn associate_attachments_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::AssociateAttachmentsRequest::decode(request_bytes)
            .map_err(|e| format!("decode associate_attachments request: {e}"))?;
        tracing::info!(target: "support_ticket", ticket_id = req.ticket_id, "associate attachments");
        let resp = self.client.associate_support_ticket_attachments_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
