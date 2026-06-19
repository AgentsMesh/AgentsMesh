use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_pod_state_v1::ReplaceCachedPodsRequest;
use agentsmesh_types::proto_ticket_v1::{Board, ListTicketsResponse, Ticket};
use agentsmesh_types::proto_ticket_state_v1::{
    ApplyTicketDeletedEventRequest, ApplyTicketStatusEventRequest,
    AppendBoardColumnTicketsRequest, FilterTicketsRequest, FilterTicketsResponse,
    InsertCreatedLabelRequest, InsertCreatedTicketRequest,
    PatchCachedTicketRequest, RemoveCachedLabelRequest,
    ReplaceBoardColumnsRequest, ReplaceCachedLabelsRequest,
    ReplaceCachedTicketsRequest, SetCurrentTicketRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmTicketState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmTicketState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmTicketState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::with_storage(crate::new_memory_backend()))),
        }
    }

    pub fn tickets_json(&self) -> String {
        serde_json::to_string(self.state.read().tickets.get_tickets()).unwrap_or_default()
    }

    // Read side (B, zero-JSON): prost-encode state tickets into the same wrapper
    // replace_cached_tickets decodes, so the shared selector decodes uniformly.
    pub fn tickets_bytes(&self) -> Vec<u8> {
        let tickets = self.state.read().tickets.get_tickets().to_vec();
        ReplaceCachedTicketsRequest { tickets }.encode_to_vec()
    }

    // Fetch→state (B): wire Ticket == cache Ticket, so decode ListTicketsResponse
    // and fold into state directly — no TS ticketsToProto/protoTicketToTicket.
    pub fn apply_fetched_tickets(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListTicketsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_tickets(resp.items);
        Ok(())
    }

    // ticket→pods cache moved off the orphan TicketService onto runtime.state
    // (the dispatch-hook SSOT). `useTicketPods` fetches via the service then
    // mirrors the result here for synchronous React reads.
    pub fn ticket_pods_json(&self, slug: &str) -> String {
        serde_json::to_string(&self.state.read().tickets.get_ticket_pods(slug))
            .unwrap_or_else(|_| "[]".to_string())
    }

    // Read side (B, zero-JSON): prost-encode ticket→pods into the pod_state
    // ReplaceCachedPodsRequest wrapper so the hook decodes via fromBinary + the
    // shared podToCache projection, not serde JSON.
    pub fn ticket_pods_bytes(&self, slug: &str) -> Vec<u8> {
        let pods = self.state.read().tickets.get_ticket_pods(slug);
        ReplaceCachedPodsRequest { pods }.encode_to_vec()
    }

    pub fn set_ticket_pods(&self, slug: &str, pods_json: &str) -> Result<(), JsValue> {
        let pods: Vec<agentsmesh_types::proto_pod_v1::Pod> =
            serde_json::from_str(pods_json).map_err(decode_err)?;
        self.state.write().tickets.set_ticket_pods(slug, pods);
        Ok(())
    }

    pub fn board_columns_json(&self) -> String {
        serde_json::to_string(self.state.read().tickets.get_board_columns()).unwrap_or_default()
    }

    // Read side (B, zero-JSON): prost-encode board columns into the same wrapper
    // replace_board_columns decodes, so the selector decodes via fromBinary.
    pub fn board_columns_bytes(&self) -> Vec<u8> {
        let columns = self.state.read().tickets.get_board_columns().to_vec();
        ReplaceBoardColumnsRequest { columns }.encode_to_vec()
    }

    pub fn labels_json(&self) -> String {
        serde_json::to_string(self.state.read().tickets.get_labels()).unwrap_or_default()
    }

    pub fn labels_bytes(&self) -> Vec<u8> {
        let labels = self.state.read().tickets.get_labels().to_vec();
        ReplaceCachedLabelsRequest { labels }.encode_to_vec()
    }

    pub fn current_ticket_json(&self) -> JsValue {
        match self.state.read().tickets.get_current_ticket() {
            Some(t) => JsValue::from_str(&serde_json::to_string(t).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    // Empty bytes when current is None → renderer treats as null (mirrors the
    // current_ticket_json NULL sentinel).
    pub fn current_ticket_bytes(&self) -> Vec<u8> {
        match self.state.read().tickets.get_current_ticket() {
            Some(t) => SetCurrentTicketRequest { ticket: Some(t.clone()) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    // Fetch→state (B): wire Ticket == cache Ticket, so decode the wire response
    // and set current directly — no TS ticketToProto/SetCurrentTicketRequest.
    pub fn apply_fetched_current_ticket(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let ticket = Ticket::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_current_ticket(Some(ticket));
        Ok(())
    }

    // Fetch→state (B): wire Board.columns == state BoardColumn, fold directly.
    pub fn apply_fetched_board_columns(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let board = Board::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_board_columns(board.columns);
        Ok(())
    }

    // Fetch→state (B): wire ListTicketsResponse.items appended to a column.
    pub fn apply_appended_board_column_tickets(
        &self, status: &str, resp_bytes: &[u8],
    ) -> Result<(), JsValue> {
        let resp = ListTicketsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().tickets.append_column_tickets(status, resp.items);
        Ok(())
    }

    // Fetch→state (B): wire ListLabelsResponse.items == state Label.
    pub fn apply_fetched_labels(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = agentsmesh_types::proto_ticket_v1::ListLabelsResponse::decode(resp_bytes)
            .map_err(decode_err)?;
        self.state.write().tickets.set_labels(resp.items);
        Ok(())
    }

    pub fn insert_created_ticket(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertCreatedTicketRequest::decode(req_bytes).map_err(decode_err)?;
        let ticket = req.ticket.ok_or_else(|| JsValue::from_str("missing ticket"))?;
        self.state.write().tickets.add_ticket(ticket);
        Ok(())
    }

    pub fn patch_cached_ticket(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchCachedTicketRequest::decode(req_bytes).map_err(decode_err)?;
        let ticket = req.ticket.ok_or_else(|| JsValue::from_str("missing ticket"))?;
        self.state.write().tickets.update_ticket(&req.slug, ticket);
        Ok(())
    }

    pub fn apply_ticket_status_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyTicketStatusEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.update_ticket_status(&req.slug, &req.status);
        Ok(())
    }

    pub fn apply_ticket_deleted_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyTicketDeletedEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.remove_ticket(&req.slug);
        Ok(())
    }

    pub fn replace_board_columns(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceBoardColumnsRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_board_columns(req.columns);
        Ok(())
    }

    pub fn append_board_column_tickets(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = AppendBoardColumnTicketsRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.append_column_tickets(&req.status, req.tickets);
        Ok(())
    }

    pub fn set_current_ticket(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = SetCurrentTicketRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_current_ticket(req.ticket);
        Ok(())
    }

    pub fn replace_cached_labels(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceCachedLabelsRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.set_labels(req.labels);
        Ok(())
    }

    pub fn insert_created_label(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertCreatedLabelRequest::decode(req_bytes).map_err(decode_err)?;
        let label = req.label.ok_or_else(|| JsValue::from_str("missing label"))?;
        self.state.write().tickets.add_label(label);
        Ok(())
    }

    pub fn remove_cached_label(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = RemoveCachedLabelRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().tickets.remove_label(req.id);
        Ok(())
    }

    pub fn filter_tickets(&self, req_bytes: &[u8]) -> Result<Vec<u8>, JsValue> {
        let req = FilterTicketsRequest::decode(req_bytes).map_err(decode_err)?;
        let search = if req.search.is_empty() { None } else { Some(req.search.as_str()) };
        let guard = self.state.read();
        let tickets: Vec<_> = guard.tickets
            .filter_tickets(search, &req.statuses, &req.priorities, &req.repository_ids)
            .into_iter().cloned().collect();
        let resp = FilterTicketsResponse { tickets };
        Ok(resp.encode_to_vec())
    }
}
