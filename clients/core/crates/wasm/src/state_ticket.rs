use agentsmesh_state::ticket_state::TicketState;
use agentsmesh_types::{BoardColumn, Label, Ticket, TicketPriority, TicketStatus};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmTicketState {
    inner: TicketState,
}

#[wasm_bindgen]
impl WasmTicketState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self { inner: TicketState::with_storage(crate::new_memory_backend()) }
    }

    pub fn tickets_json(&self) -> String {
        serde_json::to_string(self.inner.get_tickets()).unwrap_or_default()
    }

    pub fn get_ticket_by_slug_json(&self, slug: &str) -> JsValue {
        match self.inner.get_ticket_by_slug(slug) {
            Some(t) => JsValue::from_str(&serde_json::to_string(t).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn set_tickets(&mut self, tickets_json: &str) {
        if let Ok(tickets) = serde_json::from_str::<Vec<Ticket>>(tickets_json) {
            self.inner.set_tickets(tickets);
        }
    }

    pub fn add_ticket(&mut self, ticket_json: &str) {
        if let Ok(ticket) = serde_json::from_str::<Ticket>(ticket_json) {
            self.inner.add_ticket(ticket);
        }
    }

    pub fn update_ticket(&mut self, slug: &str, ticket_json: &str) {
        if let Ok(ticket) = serde_json::from_str::<Ticket>(ticket_json) {
            self.inner.update_ticket(slug, ticket);
        }
    }

    pub fn update_ticket_status(&mut self, slug: &str, status: &str) {
        if let Ok(s) = serde_json::from_str::<TicketStatus>(&format!("\"{status}\"")) {
            self.inner.update_ticket_status(slug, s);
        }
    }

    pub fn remove_ticket(&mut self, slug: &str) {
        self.inner.remove_ticket(slug);
    }

    pub fn filter_tickets_json(
        &self,
        search: &str,
        statuses_json: &str,
        priorities_json: &str,
        repository_ids_json: &str,
    ) -> String {
        let search_opt = if search.is_empty() { None } else { Some(search) };
        let statuses: Vec<TicketStatus> = serde_json::from_str(statuses_json).unwrap_or_default();
        let priorities: Vec<TicketPriority> = serde_json::from_str(priorities_json).unwrap_or_default();
        let repo_ids: Vec<i64> = serde_json::from_str(repository_ids_json).unwrap_or_default();
        let filtered = self.inner.filter_tickets(search_opt, &statuses, &priorities, &repo_ids);
        serde_json::to_string(&filtered).unwrap_or_default()
    }

    pub fn board_columns_json(&self) -> String {
        serde_json::to_string(self.inner.get_board_columns()).unwrap_or_default()
    }

    pub fn set_board_columns(&mut self, columns_json: &str) {
        if let Ok(cols) = serde_json::from_str::<Vec<BoardColumn>>(columns_json) {
            self.inner.set_board_columns(cols);
        }
    }

    pub fn append_column_tickets(&mut self, status: &str, tickets_json: &str) {
        if let (Ok(s), Ok(tickets)) = (
            serde_json::from_str::<TicketStatus>(&format!("\"{status}\"")),
            serde_json::from_str::<Vec<Ticket>>(tickets_json),
        ) {
            self.inner.append_column_tickets(s, tickets);
        }
    }

    pub fn labels_json(&self) -> String {
        serde_json::to_string(self.inner.get_labels()).unwrap_or_default()
    }

    pub fn set_labels(&mut self, labels_json: &str) {
        if let Ok(labels) = serde_json::from_str::<Vec<Label>>(labels_json) {
            self.inner.set_labels(labels);
        }
    }

    pub fn add_label(&mut self, label_json: &str) {
        if let Ok(label) = serde_json::from_str::<Label>(label_json) {
            self.inner.add_label(label);
        }
    }

    pub fn remove_label(&mut self, id: f64) {
        self.inner.remove_label(id as i64);
    }

    pub fn current_ticket_json(&self) -> JsValue {
        match self.inner.get_current_ticket() {
            Some(t) => JsValue::from_str(&serde_json::to_string(t).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn set_current_ticket(&mut self, ticket_json: &str) {
        if ticket_json.is_empty() || ticket_json == "null" {
            self.inner.set_current_ticket(None);
        } else if let Ok(t) = serde_json::from_str::<Ticket>(ticket_json) {
            self.inner.set_current_ticket(Some(t));
        }
    }
}
