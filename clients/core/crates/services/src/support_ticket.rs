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

    // Multipart endpoints stay on REST: Connect-RPC has no multipart wire,
    // and ticket creation / message replies carry optional file uploads.

    pub async fn create_ticket(
        &self, title: &str, category: &str, content: &str,
        priority: Option<String>, file_data: Vec<Vec<u8>>, file_names: Vec<String>,
    ) -> Result<String, String> {
        let mut form = reqwest::multipart::Form::new()
            .text("title", title.to_string())
            .text("category", category.to_string())
            .text("content", content.to_string());
        if let Some(p) = priority { form = form.text("priority", p); }
        for (i, (data, name)) in file_data.into_iter().zip(file_names.iter()).enumerate() {
            let part = reqwest::multipart::Part::bytes(data).file_name(name.clone());
            form = form.part(format!("files[{i}]"), part);
        }
        let resp = self.client
            .post_multipart::<serde_json::Value>("/api/v1/support-tickets", form)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn add_message(
        &self, ticket_id: i64, content: &str,
        file_data: Vec<Vec<u8>>, file_names: Vec<String>,
    ) -> Result<String, String> {
        let mut form = reqwest::multipart::Form::new()
            .text("content", content.to_string());
        for (i, (data, name)) in file_data.into_iter().zip(file_names.iter()).enumerate() {
            let part = reqwest::multipart::Part::bytes(data).file_name(name.clone());
            form = form.part(format!("files[{i}]"), part);
        }
        let resp = self.client
            .post_multipart::<serde_json::Value>(
                &format!("/api/v1/support-tickets/{ticket_id}/messages"), form,
            )
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes via @bufbuild/protobuf .toBinary() → wasm bridge → here →
    // ApiClient.*_connect (binary in / binary out, conventions §2.5). No
    // JSON path on the client.

    pub async fn list_support_tickets_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::ListSupportTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_support_tickets request: {e}"))?;
        let resp = self.client.list_support_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_support_ticket_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::GetSupportTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_support_ticket request: {e}"))?;
        let resp = self.client.get_support_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_attachment_url_connect(
        &self, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = st_proto::GetAttachmentUrlRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_attachment_url request: {e}"))?;
        let resp = self.client.get_support_ticket_attachment_url_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
