use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_state::autopilot_state::{AutopilotController, AutopilotIteration};
use agentsmesh_types::proto_autopilot_v1::{
    AutopilotController as WireController, GetIterationsResponse, ListAutopilotControllersResponse,
};
use agentsmesh_types::proto_autopilot_state_v1::{
    AppendIterationRequest, AutopilotControllerSnapshot, AutopilotIterationSnapshot,
    InsertControllerRequest, PatchControllerRequest, RemoveControllerRequest,
    ReplaceCachedControllersRequest, ReplaceCachedIterationsRequest,
    SetCurrentControllerRequest, UpdateThinkingRequest,
};
use parking_lot::RwLock;
use prost::Message;
use serde_json::Value;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmAutopilotState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

fn from_snapshot(s: AutopilotControllerSnapshot) -> AutopilotController {
    AutopilotController {
        autopilot_controller_key: s.autopilot_controller_key,
        pod_key: s.pod_key,
        status: s.status,
        phase: s.phase,
        prompt: s.prompt,
        max_iterations: s.max_iterations,
        iteration_timeout_sec: s.iteration_timeout_sec,
        no_progress_threshold: s.no_progress_threshold,
        same_error_threshold: s.same_error_threshold,
        approval_timeout_min: s.approval_timeout_min,
        current_iteration: s.current_iteration,
        control_agent_slug: s.control_agent_slug,
        circuit_breaker_state: s.circuit_breaker_state,
        circuit_breaker_reason: s.circuit_breaker_reason,
        created_at: s.created_at,
        updated_at: s.updated_at,
    }
}

fn from_iteration_snapshot(s: AutopilotIterationSnapshot) -> AutopilotIteration {
    AutopilotIteration {
        id: s.id,
        controller_key: s.controller_key,
        iteration_number: s.iteration_number,
        status: s.status,
        result: s.result,
        started_at: s.started_at,
        completed_at: s.completed_at,
    }
}

// Inverse of from_snapshot — state → proto Snapshot for the zero-JSON read side
// (controllers_bytes/etc.), so web/desktop decode via the same
// snapshotToController projection as the mutators (nested circuit_breaker).
fn to_snapshot(c: &AutopilotController) -> AutopilotControllerSnapshot {
    AutopilotControllerSnapshot {
        autopilot_controller_key: c.autopilot_controller_key.clone(),
        pod_key: c.pod_key.clone(),
        status: c.status.clone(),
        phase: c.phase.clone(),
        prompt: c.prompt.clone(),
        max_iterations: c.max_iterations,
        iteration_timeout_sec: c.iteration_timeout_sec,
        no_progress_threshold: c.no_progress_threshold,
        same_error_threshold: c.same_error_threshold,
        approval_timeout_min: c.approval_timeout_min,
        current_iteration: c.current_iteration,
        control_agent_slug: c.control_agent_slug.clone(),
        circuit_breaker_state: c.circuit_breaker_state.clone(),
        circuit_breaker_reason: c.circuit_breaker_reason.clone(),
        created_at: c.created_at.clone(),
        updated_at: c.updated_at.clone(),
    }
}

fn to_iteration_snapshot(i: &AutopilotIteration) -> AutopilotIterationSnapshot {
    AutopilotIterationSnapshot {
        id: i.id,
        controller_key: i.controller_key.clone(),
        iteration_number: i.iteration_number,
        status: i.status.clone(),
        result: i.result.clone(),
        started_at: i.started_at.clone(),
        completed_at: i.completed_at.clone(),
    }
}

impl WasmAutopilotState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmAutopilotState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::new())),
        }
    }

    pub fn controllers_json(&self) -> String {
        serde_json::to_string(self.state.read().autopilot.controllers()).unwrap_or_default()
    }

    pub fn current_controller_json(&self) -> JsValue {
        match self.state.read().autopilot.current_controller() {
            Some(c) => JsValue::from_str(&serde_json::to_string(c).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    // Read side, zero-JSON: prost-encode state controllers into the list wrapper.
    // Web/desktop decode via fromBinary + controllerSnapshotToCache (re-folds
    // circuit_breaker), replacing controllers_json (flat state struct serde).
    pub fn controllers_bytes(&self) -> Vec<u8> {
        let controllers = self.state.read().autopilot.controllers().iter().map(to_snapshot).collect();
        ReplaceCachedControllersRequest { controllers }.encode_to_vec()
    }

    // Empty bytes when current is None → renderer treats as null (mirrors the
    // current_controller_json NULL sentinel).
    pub fn current_controller_bytes(&self) -> Vec<u8> {
        match self.state.read().autopilot.current_controller() {
            Some(c) => SetCurrentControllerRequest { controller: Some(to_snapshot(c)) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn controller_by_pod_key_bytes(&self, pod_key: &str) -> Vec<u8> {
        match self.state.read().autopilot.get_controller_by_pod_key(pod_key) {
            Some(c) => SetCurrentControllerRequest { controller: Some(to_snapshot(c)) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn iterations_bytes(&self, key: &str) -> Vec<u8> {
        match self.state.read().autopilot.get_iterations(key) {
            Some(iters) => ReplaceCachedIterationsRequest {
                autopilot_controller_key: key.to_string(),
                iterations: iters.iter().map(to_iteration_snapshot).collect(),
            }
            .encode_to_vec(),
            None => Vec::new(),
        }
    }

    // Fetch→state: decode wire ListAutopilotControllersResponse + fold into
    // state. Replaces TS fromProtoController + controllerToProto +
    // replace_cached_controllers — web fetch hands raw wire bytes to Rust.
    pub fn apply_fetched_controllers(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListAutopilotControllersResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().autopilot.apply_fetched_controllers(resp.items);
        Ok(())
    }

    // Fetch→state: decode wire GetIterationsResponse + fold into state under key.
    pub fn apply_fetched_iterations(&self, key: &str, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = GetIterationsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().autopilot.apply_fetched_iterations(key.to_string(), resp.items);
        Ok(())
    }

    // Single-object fetch (B): decode the wire GetAutopilotController response +
    // upsert/set-current via the shared wire→state converter — no TS
    // controllerSnapshotToCache + insert/set_current dispatch round-trip.
    pub fn apply_fetched_current_controller(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let wire = WireController::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().autopilot.apply_fetched_current_controller(wire);
        Ok(())
    }

    pub fn set_current_controller_proto(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = SetCurrentControllerRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().autopilot.set_current_controller(req.controller.map(from_snapshot));
        Ok(())
    }

    pub fn insert_controller(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertControllerRequest::decode(req_bytes).map_err(decode_err)?;
        if let Some(c) = req.controller {
            self.state.write().autopilot.add_controller(from_snapshot(c));
        }
        Ok(())
    }

    pub fn patch_controller(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchControllerRequest::decode(req_bytes).map_err(decode_err)?;
        if let Some(c) = req.controller {
            self.state.write().autopilot.update_controller(&req.autopilot_controller_key, from_snapshot(c));
        }
        Ok(())
    }

    pub fn remove_controller(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = RemoveControllerRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().autopilot.remove_controller(&req.autopilot_controller_key);
        Ok(())
    }

    // `_proto` aliases match the combined-service method names the shared store
    // calls (getAutopilotService surface), so the store can consolidate onto
    // getAutopilotState (this struct) without renaming its call sites.
    pub fn remove_controller_proto(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        self.remove_controller(req_bytes)
    }

    pub fn get_iterations_json(&self, key: &str) -> JsValue {
        match self.state.read().autopilot.get_iterations(key) {
            Some(iters) => JsValue::from_str(&serde_json::to_string(iters).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn append_iteration(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = AppendIterationRequest::decode(req_bytes).map_err(decode_err)?;
        if let Some(iter) = req.iteration {
            self.state.write().autopilot.add_iteration(req.autopilot_controller_key, from_iteration_snapshot(iter));
        }
        Ok(())
    }

    pub fn get_thinking_json(&self, key: &str) -> JsValue {
        match self.state.read().autopilot.get_thinking(key) {
            Some(t) => JsValue::from_str(&serde_json::to_string(t).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn get_thinking_history_json(&self, key: &str) -> JsValue {
        match self.state.read().autopilot.get_thinking_history(key) {
            Some(h) => JsValue::from_str(&serde_json::to_string(h).unwrap_or_default()),
            None => JsValue::from_str("[]"),
        }
    }

    pub fn update_thinking(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = UpdateThinkingRequest::decode(req_bytes).map_err(decode_err)?;
        if let Ok(thinking) = serde_json::from_str::<Value>(&req.thinking_json) {
            self.state.write().autopilot.update_thinking(req.autopilot_controller_key, thinking);
        }
        Ok(())
    }

    pub fn update_thinking_proto(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        self.update_thinking(req_bytes)
    }

    pub fn get_controller_by_pod_key_json(&self, pod_key: &str) -> JsValue {
        match self.state.read().autopilot.get_controller_by_pod_key(pod_key) {
            Some(c) => JsValue::from_str(&serde_json::to_string(c).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }
}
