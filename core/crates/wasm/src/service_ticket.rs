use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::TicketService;
use agentsmesh_state::ticket_state::TicketState;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmTicketService(pub(crate) TicketService);

#[wasm_bindgen]
impl WasmTicketService {
    pub(crate) fn new(client: Arc<ApiClient>, state: TicketState) -> Self {
        Self(TicketService::new(client, state))
    }

    pub fn tickets_json(&self) -> String { self.0.tickets_json() }

    pub fn get_ticket_by_slug_json(&self, slug: &str) -> JsValue {
        match self.0.get_ticket_by_slug_json(slug) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn current_ticket_json(&self) -> JsValue {
        match self.0.current_ticket_json() {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn board_columns_json(&self) -> String { self.0.board_columns_json() }
    pub fn labels_json(&self) -> String { self.0.labels_json() }

    pub fn filter_tickets_json(
        &self, search: &str, statuses_json: &str,
        priorities_json: &str, repository_ids_json: &str,
    ) -> String {
        self.0.filter_tickets_json(search, statuses_json, priorities_json, repository_ids_json)
    }

    pub fn set_tickets(&self, json: &str) { self.0.set_tickets(json); }
    pub fn add_ticket(&self, json: &str) { self.0.add_ticket(json); }

    pub fn update_ticket_local(&self, slug: &str, json: &str) {
        self.0.update_ticket_local(slug, json);
    }

    pub fn update_ticket_status_local(&self, slug: &str, status: &str) {
        self.0.update_ticket_status_local(slug, status);
    }

    pub fn remove_ticket(&self, slug: &str) { self.0.remove_ticket(slug); }
    pub fn set_current_ticket(&self, json: &str) { self.0.set_current_ticket(json); }
    pub fn set_board_columns(&self, json: &str) { self.0.set_board_columns(json); }

    pub fn append_column_tickets(&self, status: &str, json: &str) {
        self.0.append_column_tickets(status, json);
    }

    pub fn set_labels(&self, json: &str) { self.0.set_labels(json); }
    pub fn add_label(&self, json: &str) { self.0.add_label(json); }
    pub fn remove_label(&self, id: f64) { self.0.remove_label(id); }

    pub async fn fetch_tickets(
        &self, status: Option<String>, limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        self.0.fetch_tickets(status, limit, offset).await
    }

    pub async fn fetch_board(&self, repository_id: Option<i64>) -> Result<String, String> {
        self.0.fetch_board(repository_id).await
    }

    pub async fn load_more_column(
        &self, status: &str, offset: u32, limit: u32,
    ) -> Result<String, String> {
        self.0.load_more_column(status, offset, limit).await
    }

    pub async fn fetch_ticket(&self, slug: &str) -> Result<String, String> {
        self.0.fetch_ticket(slug).await
    }

    pub async fn create_ticket(&self, request_json: &str) -> Result<String, String> {
        self.0.create_ticket(request_json).await
    }

    pub async fn update_ticket(&self, slug: &str, request_json: &str) -> Result<String, String> {
        self.0.update_ticket(slug, request_json).await
    }

    pub async fn delete_ticket(&self, slug: &str) -> Result<(), String> {
        self.0.delete_ticket(slug).await
    }

    pub async fn update_ticket_status(&self, slug: &str, status: &str) -> Result<String, String> {
        self.0.update_ticket_status(slug, status).await
    }

    pub async fn fetch_labels(&self, repository_id: Option<i64>) -> Result<String, String> {
        self.0.fetch_labels(repository_id).await
    }

    pub async fn create_label(&self, name: &str, color: &str, repository_id: Option<i64>) -> Result<String, String> {
        self.0.create_label(name, color, repository_id).await
    }

    pub async fn delete_label(&self, id: f64) -> Result<(), String> {
        self.0.delete_label(id).await
    }

    pub async fn get_ticket_pods(
        &self, slug: &str, active_only: Option<bool>,
    ) -> Result<String, String> {
        self.0.get_ticket_pods(slug, active_only).await
    }

    pub fn ticket_pods_json(&self, slug: &str) -> String {
        self.0.ticket_pods_json(slug)
    }

    pub async fn get_sub_tickets(&self, slug: &str) -> Result<String, String> {
        self.0.get_sub_tickets(slug).await
    }
}
