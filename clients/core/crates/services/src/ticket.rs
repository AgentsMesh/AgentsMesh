use std::sync::Arc;
use std::sync::RwLock;

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::ticket_state::TicketState;
use agentsmesh_types::{
    Ticket, TicketStatus, TicketPriority, Label, BoardColumn,
    CreateTicketRequest, UpdateTicketRequest, UpdateTicketStatusRequest,
    CreateLabelRequest,
};

use crate::parse_status;

pub struct TicketService {
    client: Arc<ApiClient>,
    state: RwLock<TicketState>,
}

impl TicketService {
    pub fn new(client: Arc<ApiClient>, state: TicketState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub fn tickets_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_tickets()).unwrap_or_default()
    }

    pub fn get_ticket_by_slug_json(&self, slug: &str) -> Option<String> {
        self.state.read().unwrap().get_ticket_by_slug(slug)
            .map(|t| serde_json::to_string(t).unwrap_or_default())
    }

    pub fn current_ticket_json(&self) -> Option<String> {
        self.state.read().unwrap().get_current_ticket()
            .map(|t| serde_json::to_string(t).unwrap_or_default())
    }

    pub fn board_columns_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_board_columns()).unwrap_or_default()
    }

    pub fn labels_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_labels()).unwrap_or_default()
    }

    pub fn filter_tickets_json(
        &self, search: &str, statuses_json: &str,
        priorities_json: &str, repository_ids_json: &str,
    ) -> String {
        let statuses: Vec<TicketStatus> = serde_json::from_str(statuses_json).unwrap_or_default();
        let priorities: Vec<TicketPriority> = serde_json::from_str(priorities_json).unwrap_or_default();
        let repo_ids: Vec<i64> = serde_json::from_str(repository_ids_json).unwrap_or_default();
        let s = if search.is_empty() { None } else { Some(search) };
        let binding = self.state.read().unwrap();
        let filtered = binding.filter_tickets(s, &statuses, &priorities, &repo_ids);
        serde_json::to_string(&filtered).unwrap_or_default()
    }

    pub fn set_tickets(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<Ticket>>(json) {
            self.state.write().unwrap().set_tickets(v);
        }
    }

    pub fn add_ticket(&self, json: &str) {
        if let Ok(t) = serde_json::from_str::<Ticket>(json) {
            self.state.write().unwrap().add_ticket(t);
        }
    }

    pub fn update_ticket_local(&self, slug: &str, json: &str) {
        if let Ok(t) = serde_json::from_str::<Ticket>(json) {
            self.state.write().unwrap().update_ticket(slug, t);
        }
    }

    pub fn update_ticket_status_local(&self, slug: &str, status: &str) {
        let parsed = parse_status::<TicketStatus>(status);
        self.state.write().unwrap().update_ticket_status(slug, parsed);
    }

    pub fn remove_ticket(&self, slug: &str) {
        self.state.write().unwrap().remove_ticket(slug);
    }

    pub fn set_current_ticket(&self, json: &str) {
        let t = if json.is_empty() { None } else { serde_json::from_str::<Ticket>(json).ok() };
        self.state.write().unwrap().set_current_ticket(t);
    }

    pub fn set_board_columns(&self, json: &str) {
        if let Ok(cols) = serde_json::from_str::<Vec<BoardColumn>>(json) {
            self.state.write().unwrap().set_board_columns(cols);
        }
    }

    pub fn append_column_tickets(&self, status: &str, json: &str) {
        let parsed = parse_status::<TicketStatus>(status);
        if let Ok(tickets) = serde_json::from_str::<Vec<Ticket>>(json) {
            self.state.write().unwrap().append_column_tickets(parsed, tickets);
        }
    }

    pub fn set_labels(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<Label>>(json) {
            self.state.write().unwrap().set_labels(v);
        }
    }

    pub fn add_label(&self, json: &str) {
        if let Ok(l) = serde_json::from_str::<Label>(json) {
            self.state.write().unwrap().add_label(l);
        }
    }

    pub fn remove_label(&self, id: f64) {
        self.state.write().unwrap().remove_label(id as i64);
    }

    pub async fn fetch_tickets(
        &self, status: Option<String>, limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_tickets(status.as_deref(), limit, offset)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_tickets(resp.tickets.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn fetch_board(&self, repository_id: Option<i64>) -> Result<String, String> {
        let resp = self.client
            .get_ticket_board(repository_id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_board_columns(resp.columns.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn load_more_column(
        &self, status: &str, offset: u32, limit: u32,
    ) -> Result<String, String> {
        let resp = self.client
            .list_tickets(Some(status), Some(limit), Some(offset))
            .await.map_err(crate::wire)?;
        let parsed = parse_status::<TicketStatus>(status);
        self.state.write().unwrap().append_column_tickets(parsed, resp.tickets.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn fetch_ticket(&self, slug: &str) -> Result<String, String> {
        let ticket: Ticket = self.client
            .get_ticket(slug)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_current_ticket(Some(ticket.clone()));
        serde_json::to_string(&ticket).map_err(crate::wire)
    }

    pub async fn create_ticket(&self, request_json: &str) -> Result<String, String> {
        let req: CreateTicketRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let ticket: Ticket = self.client
            .create_ticket(&req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().add_ticket(ticket.clone());
        serde_json::to_string(&ticket).map_err(crate::wire)
    }

    pub async fn update_ticket(&self, slug: &str, request_json: &str) -> Result<String, String> {
        let req: UpdateTicketRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let ticket: Ticket = self.client
            .update_ticket(slug, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().update_ticket(slug, ticket.clone());
        serde_json::to_string(&ticket).map_err(crate::wire)
    }

    pub async fn delete_ticket(&self, slug: &str) -> Result<(), String> {
        self.client.delete_ticket(slug).await.map_err(crate::wire)?;
        self.state.write().unwrap().remove_ticket(slug);
        Ok(())
    }

    pub async fn update_ticket_status(&self, slug: &str, status: &str) -> Result<String, String> {
        let parsed = parse_status::<TicketStatus>(status);
        let req = UpdateTicketStatusRequest { status: parsed };
        let ticket: Ticket = self.client
            .update_ticket_status(slug, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().update_ticket(slug, ticket.clone());
        serde_json::to_string(&ticket).map_err(crate::wire)
    }

    pub async fn fetch_labels(&self, repository_id: Option<i64>) -> Result<String, String> {
        let resp = self.client
            .list_labels(repository_id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_labels(resp.labels.clone());
        serde_json::to_string(&resp.labels).map_err(crate::wire)
    }

    pub async fn create_label(&self, name: &str, color: &str, repository_id: Option<i64>) -> Result<String, String> {
        let req = CreateLabelRequest { name: name.to_string(), color: color.to_string() };
        let _ = repository_id;
        let label: Label = self.client
            .create_label(&req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().add_label(label.clone());
        serde_json::to_string(&label).map_err(crate::wire)
    }

    pub async fn delete_label(&self, id: f64) -> Result<(), String> {
        self.client.delete_label(id as i64).await.map_err(crate::wire)?;
        self.state.write().unwrap().remove_label(id as i64);
        Ok(())
    }

    pub async fn get_ticket_pods(
        &self, slug: &str, active_only: Option<bool>,
    ) -> Result<String, String> {
        let resp = self.client
            .get_ticket_pods(slug, active_only)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_ticket_pods(slug, resp.pods.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub fn ticket_pods_json(&self, slug: &str) -> String {
        let pods = self.state.read().unwrap().get_ticket_pods(slug);
        serde_json::to_string(&pods).unwrap_or_else(|_| "[]".into())
    }

    pub async fn get_sub_tickets(&self, slug: &str) -> Result<String, String> {
        let resp = self.client
            .get_sub_tickets(slug)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
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

use agentsmesh_types::proto_ticket_v1 as ticket_proto;
use prost::Message;

impl TicketService {
    pub async fn list_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::ListTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_tickets request: {e}"))?;
        let resp = self.client.list_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_ticket request: {e}"))?;
        let resp = self.client.get_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::CreateTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_ticket request: {e}"))?;
        let resp = self.client.create_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_ticket request: {e}"))?;
        let resp = self.client.update_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::DeleteTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_ticket request: {e}"))?;
        let resp = self.client.delete_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_ticket_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateTicketStatusRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_ticket_status request: {e}"))?;
        let resp = self.client.update_ticket_status_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_active_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetActiveTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_active_tickets request: {e}"))?;
        let resp = self.client.get_active_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_board_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetBoardRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_board request: {e}"))?;
        let resp = self.client.get_board_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_sub_tickets_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::GetSubTicketsRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_sub_tickets request: {e}"))?;
        let resp = self.client.get_sub_tickets_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn add_assignee_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::AddAssigneeRequest::decode(request_bytes)
            .map_err(|e| format!("decode add_assignee request: {e}"))?;
        let resp = self.client.add_assignee_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn remove_assignee_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::RemoveAssigneeRequest::decode(request_bytes)
            .map_err(|e| format!("decode remove_assignee request: {e}"))?;
        let resp = self.client.remove_assignee_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_labels_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::ListLabelsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_labels request: {e}"))?;
        let resp = self.client.list_labels_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::CreateLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_label request: {e}"))?;
        let resp = self.client.create_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::UpdateLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_label request: {e}"))?;
        let resp = self.client.update_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::DeleteLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_label request: {e}"))?;
        let resp = self.client.delete_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn add_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::AddLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode add_label request: {e}"))?;
        let resp = self.client.add_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn remove_label_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ticket_proto::RemoveLabelRequest::decode(request_bytes)
            .map_err(|e| format!("decode remove_label request: {e}"))?;
        let resp = self.client.remove_label_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
