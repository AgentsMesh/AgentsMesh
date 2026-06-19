// wire(proto.autopilot.v1) → state(autopilot_state::*) projection for the
// fetch→state path. Fetch responses carry wire types; this folds them into the
// in-memory model so set_controllers/set_iterations own the result — replacing
// the TS fromProtoController + controllerToProto round-trip (autopilotConnect.ts
// + autopilotProtoMap.ts).
//
// Field-set mismatch (wire ≠ state): the wire `AutopilotController` carries
// id / user_takeover / started_at / last_iteration_at that the state model has
// no slot for (the state schema is the UI cache shape, not the wire entity) —
// they drop here, matching the pre-existing TS net effect. The state-only
// config fields (status / iteration_timeout_sec / *_threshold /
// approval_timeout_min / control_agent_slug / updated_at) have no wire source
// and default to None; realtime patch events fill them later.

use agentsmesh_types::proto_autopilot_v1::{
    AutopilotController as WireController, AutopilotIteration as WireIteration,
};

use crate::autopilot_state::{AutopilotController, AutopilotIteration, AutopilotState};

fn opt(s: String) -> Option<String> {
    if s.is_empty() { None } else { Some(s) }
}

pub fn wire_controller_to_state(w: WireController) -> AutopilotController {
    let (cb_state, cb_reason) = w
        .circuit_breaker
        .map(|cb| (opt(cb.state), opt(cb.reason)))
        .unwrap_or((None, None));
    AutopilotController {
        autopilot_controller_key: w.autopilot_controller_key,
        pod_key: w.pod_key,
        status: None,
        phase: opt(w.phase),
        prompt: opt(w.prompt),
        max_iterations: Some(w.max_iterations as i64),
        iteration_timeout_sec: None,
        no_progress_threshold: None,
        same_error_threshold: None,
        approval_timeout_min: None,
        current_iteration: Some(w.current_iteration as i64),
        control_agent_slug: None,
        circuit_breaker_state: cb_state,
        circuit_breaker_reason: cb_reason,
        created_at: opt(w.created_at),
        updated_at: None,
    }
}

// controller_key comes from the caller's fetch key, not the wire field — it is
// the same key set_iterations buckets under, so the two stay consistent (the
// old TS path stuffed a stringified "0" controller id here).
fn wire_iteration_to_state(key: &str, w: WireIteration) -> AutopilotIteration {
    AutopilotIteration {
        id: w.id,
        controller_key: key.to_string(),
        iteration_number: Some(w.iteration_number),
        status: opt(w.status),
        result: opt(w.result),
        started_at: w.started_at,
        completed_at: w.completed_at,
    }
}

impl AutopilotState {
    pub fn apply_fetched_controllers(&mut self, wire: Vec<WireController>) {
        let controllers = wire.into_iter().map(wire_controller_to_state).collect();
        self.set_controllers(controllers);
    }

    pub fn apply_fetched_iterations(&mut self, key: String, wire: Vec<WireIteration>) {
        let iters = wire.into_iter().map(|w| wire_iteration_to_state(&key, w)).collect();
        self.set_iterations(key, iters);
    }

    // Single-object fetch (B): upsert the controller into the cache and anchor it
    // as current — mirrors the store's insert+set_current dispatch pair.
    pub fn apply_fetched_current_controller(&mut self, wire: WireController) {
        let ctrl = wire_controller_to_state(wire);
        let key = ctrl.autopilot_controller_key.clone();
        if self.controllers().iter().any(|c| c.autopilot_controller_key == key) {
            self.update_controller(&key, ctrl.clone());
        } else {
            self.add_controller(ctrl.clone());
        }
        self.set_current_controller(Some(ctrl));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use agentsmesh_types::proto_autopilot_v1::CircuitBreaker;

    fn wire_ctrl() -> WireController {
        WireController {
            id: 42,
            autopilot_controller_key: "ctrl-1".into(),
            pod_key: "pod-xyz".into(),
            phase: "running".into(),
            current_iteration: 3,
            max_iterations: 10,
            circuit_breaker: Some(CircuitBreaker { state: "closed".into(), reason: String::new() }),
            user_takeover: true,
            prompt: "fix the bug".into(),
            started_at: Some("s".into()),
            last_iteration_at: Some("li".into()),
            created_at: "c".into(),
        }
    }

    #[test]
    fn controller_maps_core_fields_and_drops_wire_only() {
        let s = wire_controller_to_state(wire_ctrl());
        assert_eq!(s.autopilot_controller_key, "ctrl-1");
        assert_eq!(s.pod_key, "pod-xyz");
        assert_eq!(s.phase.as_deref(), Some("running"));
        assert_eq!(s.prompt.as_deref(), Some("fix the bug"));
        assert_eq!(s.current_iteration, Some(3)); // wire i32 → state Option<i64>
        assert_eq!(s.max_iterations, Some(10));
        assert_eq!(s.created_at.as_deref(), Some("c"));
        // wire-only fields have no state slot
        assert_eq!(s.status, None);
        assert_eq!(s.control_agent_slug, None);
        assert_eq!(s.updated_at, None);
    }

    #[test]
    fn circuit_breaker_unfolds_empty_reason_to_none() {
        let s = wire_controller_to_state(wire_ctrl());
        assert_eq!(s.circuit_breaker_state.as_deref(), Some("closed"));
        assert_eq!(s.circuit_breaker_reason, None); // empty string → None
    }

    #[test]
    fn circuit_breaker_keeps_reason_when_set() {
        let mut w = wire_ctrl();
        w.circuit_breaker = Some(CircuitBreaker { state: "open".into(), reason: "too many".into() });
        let s = wire_controller_to_state(w);
        assert_eq!(s.circuit_breaker_state.as_deref(), Some("open"));
        assert_eq!(s.circuit_breaker_reason.as_deref(), Some("too many"));
    }

    #[test]
    fn apply_fetched_controllers_replaces_list() {
        let mut st = AutopilotState::new();
        st.apply_fetched_controllers(vec![wire_ctrl()]);
        assert_eq!(st.controllers().len(), 1);
        assert_eq!(st.controllers()[0].autopilot_controller_key, "ctrl-1");
    }

    #[test]
    fn iteration_uses_caller_key_not_wire() {
        let mut st = AutopilotState::new();
        let w = WireIteration {
            id: 7,
            controller_key: "wire-ignored".into(),
            iteration_number: 2,
            status: "completed".into(),
            result: String::new(),
            started_at: Some("t".into()),
            completed_at: Some("done".into()),
        };
        st.apply_fetched_iterations("ctrl-1".into(), vec![w]);
        let iters = st.get_iterations("ctrl-1").expect("iterations");
        assert_eq!(iters.len(), 1);
        assert_eq!(iters[0].id, 7);
        assert_eq!(iters[0].controller_key, "ctrl-1"); // caller key, not "wire-ignored"
        assert_eq!(iters[0].iteration_number, Some(2));
        assert_eq!(iters[0].status.as_deref(), Some("completed"));
        assert_eq!(iters[0].result, None); // empty string → None
        assert_eq!(iters[0].completed_at.as_deref(), Some("done"));
    }
}
