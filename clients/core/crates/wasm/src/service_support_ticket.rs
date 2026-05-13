use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::SupportTicketService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmSupportTicketService(pub(crate) SupportTicketService);

#[wasm_bindgen]
impl WasmSupportTicketService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(SupportTicketService::new(client))
    }

    // Multipart REST endpoints (no Connect multipart wire).

    pub async fn create_ticket(
        &self, title: &str, category: &str, content: &str,
        priority: Option<String>, file_data: Vec<js_sys::Uint8Array>, file_names: Vec<String>,
    ) -> Result<String, String> {
        let bytes: Vec<Vec<u8>> = file_data.iter().map(|d| d.to_vec()).collect();
        self.0.create_ticket(title, category, content, priority, bytes, file_names).await
    }

    pub async fn add_message(
        &self, ticket_id: i64, content: &str,
        file_data: Vec<js_sys::Uint8Array>, file_names: Vec<String>,
    ) -> Result<String, String> {
        let bytes: Vec<Vec<u8>> = file_data.iter().map(|d| d.to_vec()).collect();
        self.0.add_message(ticket_id, content, bytes, file_names).await
    }

    // -------- Connect-RPC (binary wire) --------

    #[wasm_bindgen(js_name = listSupportTicketsConnect)]
    pub async fn list_support_tickets_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_support_tickets_connect(request).await
    }

    #[wasm_bindgen(js_name = getSupportTicketConnect)]
    pub async fn get_support_ticket_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_support_ticket_connect(request).await
    }

    #[wasm_bindgen(js_name = getAttachmentUrlConnect)]
    pub async fn get_attachment_url_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_attachment_url_connect(request).await
    }
}
