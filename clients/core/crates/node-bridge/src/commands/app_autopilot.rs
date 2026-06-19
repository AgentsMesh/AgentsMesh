use napi_derive::napi;

use agentsmesh_state::autopilot_state::{AutopilotController, AutopilotIteration};
use agentsmesh_types::proto_autopilot_v1::{
    AutopilotController as WireController, GetIterationsResponse, ListAutopilotControllersResponse,
};
use agentsmesh_types::proto_autopilot_state_v1::{
    AppendIterationRequest, AutopilotControllerSnapshot, AutopilotIterationSnapshot,
    InsertControllerRequest, PatchControllerRequest, RemoveControllerRequest,
    ReplaceCachedControllersRequest, ReplaceCachedIterationsRequest, SetCurrentControllerRequest,
    UpdateThinkingRequest,
};
use prost::Message as _;

use crate::AppState;

// Autopilot state surface over the shared `runtime.state` (dispatch-hook SSOT),
// mirroring app_channel.rs / app_pod.rs / app_runner.rs. Keeps
// `runtime.state.autopilot` fed by fetch baseline so the post-dispatch snapshot
// (main/realtime.ts) carries the full controller + iterations + thinking the
// renderer mirror needs.
fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
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

// Inverse of from_snapshot — state → proto Snapshot for the realtime mirror,
// encoded into ReplaceCached*Request so the renderer decodes via the same
// snapshotToController/Iteration projection as the mutators (shape parity).
fn to_snapshot(c: AutopilotController) -> AutopilotControllerSnapshot {
    AutopilotControllerSnapshot {
        autopilot_controller_key: c.autopilot_controller_key,
        pod_key: c.pod_key,
        status: c.status,
        phase: c.phase,
        prompt: c.prompt,
        max_iterations: c.max_iterations,
        iteration_timeout_sec: c.iteration_timeout_sec,
        no_progress_threshold: c.no_progress_threshold,
        same_error_threshold: c.same_error_threshold,
        approval_timeout_min: c.approval_timeout_min,
        current_iteration: c.current_iteration,
        control_agent_slug: c.control_agent_slug,
        circuit_breaker_state: c.circuit_breaker_state,
        circuit_breaker_reason: c.circuit_breaker_reason,
        created_at: c.created_at,
        updated_at: c.updated_at,
    }
}

fn to_iteration_snapshot(i: AutopilotIteration) -> AutopilotIterationSnapshot {
    AutopilotIterationSnapshot {
        id: i.id,
        controller_key: i.controller_key,
        iteration_number: i.iteration_number,
        status: i.status,
        result: i.result,
        started_at: i.started_at,
        completed_at: i.completed_at,
    }
}

#[napi]
impl AppState {
    // ── Snapshot reads ──

    #[napi]
    pub fn app_autopilot_controllers_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().autopilot.controllers()).unwrap_or_default()
    }

    #[napi]
    pub fn app_autopilot_iterations_json(&self, key: String) -> String {
        match self.runtime.state.read().autopilot.get_iterations(&key) {
            Some(iters) => serde_json::to_string(iters).unwrap_or_default(),
            None => String::new(),
        }
    }

    #[napi]
    pub fn app_autopilot_thinking_json(&self, key: String) -> String {
        match self.runtime.state.read().autopilot.get_thinking(&key) {
            Some(t) => serde_json::to_string(t).unwrap_or_default(),
            None => String::new(),
        }
    }

    #[napi]
    pub fn app_autopilot_thinking_history_json(&self, key: String) -> String {
        match self.runtime.state.read().autopilot.get_thinking_history(&key) {
            Some(h) => serde_json::to_string(h).unwrap_or_else(|_| "[]".to_string()),
            None => "[]".to_string(),
        }
    }

    // Proto-bytes variants for the realtime mirror — reuse the *Request wrappers
    // so the renderer decodes via snapshotToController/Iteration, not by assigning
    // prost serde JSON (which flattens circuit_breaker_state, drifting from the
    // mutators' nested circuit_breaker).
    #[napi]
    pub fn app_autopilot_controllers_proto(&self) -> Vec<u8> {
        let controllers = self
            .runtime
            .state
            .read()
            .autopilot
            .controllers()
            .iter()
            .cloned()
            .map(to_snapshot)
            .collect();
        ReplaceCachedControllersRequest { controllers }.encode_to_vec()
    }

    // Empty bytes when the key has no iterations → renderer skips (preserves
    // cache), matching the old app_autopilot_iterations_json "" sentinel.
    #[napi]
    pub fn app_autopilot_iterations_proto(&self, key: String) -> Vec<u8> {
        let guard = self.runtime.state.read();
        match guard.autopilot.get_iterations(&key) {
            Some(iters) => {
                let iterations = iters.iter().cloned().map(to_iteration_snapshot).collect();
                ReplaceCachedIterationsRequest {
                    autopilot_controller_key: key,
                    iterations,
                }
                .encode_to_vec()
            }
            None => Vec::new(),
        }
    }

    // ── Fetch-mirror mutators → runtime.state baseline ──

    // Fetch→state baseline: decode wire ListAutopilotControllersResponse + fold
    // into runtime.state so the post-dispatch realtime snapshot carries it.
    #[napi]
    pub fn app_autopilot_apply_fetched_controllers(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListAutopilotControllersResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().autopilot.apply_fetched_controllers(resp.items);
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_apply_fetched_iterations(&self, key: String, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = GetIterationsResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().autopilot.apply_fetched_iterations(key, resp.items);
        Ok(())
    }

    // Single-object fetch (B): decode wire GetAutopilotController response +
    // upsert/set-current via the shared wire→state converter.
    #[napi]
    pub fn app_autopilot_apply_fetched_current_controller(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let wire = WireController::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().autopilot.apply_fetched_current_controller(wire);
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_insert_controller(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = InsertControllerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(c) = req.controller {
            self.runtime.state.write().autopilot.add_controller(from_snapshot(c));
        }
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_set_current_controller_proto(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = SetCurrentControllerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .autopilot
            .set_current_controller(req.controller.map(from_snapshot));
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_patch_controller(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = PatchControllerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(c) = req.controller {
            self.runtime
                .state
                .write()
                .autopilot
                .update_controller(&req.autopilot_controller_key, from_snapshot(c));
        }
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_remove_controller_proto(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = RemoveControllerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().autopilot.remove_controller(&req.autopilot_controller_key);
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_append_iteration(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = AppendIterationRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(iter) = req.iteration {
            self.runtime
                .state
                .write()
                .autopilot
                .add_iteration(req.autopilot_controller_key, from_iteration_snapshot(iter));
        }
        Ok(())
    }

    #[napi]
    pub fn app_autopilot_update_thinking_proto(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = UpdateThinkingRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Ok(v) = serde_json::from_str(&req.thinking_json) {
            self.runtime.state.write().autopilot.update_thinking(req.autopilot_controller_key, v);
        }
        Ok(())
    }
}
