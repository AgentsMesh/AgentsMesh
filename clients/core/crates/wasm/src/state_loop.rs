use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_state::loop_state::{LoopData, LoopRunData};
use agentsmesh_types::proto_loop_v1::{
    ListLoopsResponse, ListRunsResponse, Loop as ProtoLoop, LoopRun as ProtoLoopRun,
};
use agentsmesh_types::proto_loop_state_v1::{
    ClearCurrentLoopRequest, ClearLoopRunsRequest,
    InsertLoopRunRequest, PatchLoopFromActionRequest, PatchLoopRunStatusRequest,
    ReplaceCachedLoopsRequest, ReplaceCachedRunsRequest, SetCurrentLoopRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmLoopState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

fn loop_from_proto(p: ProtoLoop) -> LoopData {
    LoopData {
        id: p.id,
        slug: p.slug,
        name: p.name,
        description: p.description,
        schedule: None,
        is_enabled: false,
        status: Some(p.status),
        agent_slug: Some(p.agent_slug),
        permission_mode: Some(p.permission_mode),
        prompt_template: Some(p.prompt_template),
        config_overrides: serde_json::from_str(&p.config_overrides_json).ok(),
        prompt_variables: serde_json::from_str(&p.prompt_variables_json).ok(),
        execution_mode: Some(p.execution_mode),
        autopilot_config: serde_json::from_str(&p.autopilot_config_json).ok(),
        sandbox_strategy: Some(p.sandbox_strategy),
        session_persistence: Some(p.session_persistence),
        concurrency_policy: Some(p.concurrency_policy),
        max_concurrent_runs: Some(p.max_concurrent_runs),
        max_retained_runs: Some(p.max_retained_runs),
        timeout_minutes: Some(p.timeout_minutes),
        idle_timeout_sec: Some(p.idle_timeout_sec),
        total_runs: Some(p.total_runs),
        successful_runs: Some(p.successful_runs),
        failed_runs: Some(p.failed_runs),
        active_run_count: Some(p.active_run_count),
        last_run_at: p.last_run_at,
        created_at: Some(p.created_at),
        updated_at: Some(p.updated_at),
        cron_expression: p.cron_expression,
        callback_url: p.callback_url,
        repository_id: p.repository_id,
        runner_id: p.runner_id,
        branch_name: p.branch_name,
        ticket_id: p.ticket_id,
        credential_profile_id: p.credential_profile_id,
        avg_duration_sec: p.avg_duration_sec,
        used_env_bundles: p.used_env_bundles,
    }
}

fn run_from_proto(p: ProtoLoopRun) -> LoopRunData {
    LoopRunData {
        id: p.id,
        loop_slug: String::new(),
        run_number: Some(p.run_number),
        status: p.status,
        pod_key: p.pod_key,
        started_at: p.started_at,
        completed_at: p.completed_at,
        error_message: p.error_message,
        created_at: Some(p.created_at),
    }
}

fn json_str(v: &Option<serde_json::Value>) -> String {
    v.as_ref().map(|x| x.to_string()).unwrap_or_default()
}

// Inverse of loop_from_proto for the read side (B). LoopData carries the
// detail-page proto fields (cron/webhook/repo/runner/ticket/branch/credential/
// avg-duration); only the truly server-only fields (e.g. internal counters not
// in LoopData) fall through to ..Default.
fn loop_to_proto(l: &LoopData) -> ProtoLoop {
    ProtoLoop {
        id: l.id,
        slug: l.slug.clone(),
        name: l.name.clone(),
        description: l.description.clone(),
        agent_slug: l.agent_slug.clone().unwrap_or_default(),
        permission_mode: l.permission_mode.clone().unwrap_or_default(),
        prompt_template: l.prompt_template.clone().unwrap_or_default(),
        config_overrides_json: json_str(&l.config_overrides),
        prompt_variables_json: json_str(&l.prompt_variables),
        execution_mode: l.execution_mode.clone().unwrap_or_default(),
        autopilot_config_json: json_str(&l.autopilot_config),
        status: l.status.clone().unwrap_or_default(),
        sandbox_strategy: l.sandbox_strategy.clone().unwrap_or_default(),
        session_persistence: l.session_persistence.unwrap_or_default(),
        concurrency_policy: l.concurrency_policy.clone().unwrap_or_default(),
        max_concurrent_runs: l.max_concurrent_runs.unwrap_or_default(),
        max_retained_runs: l.max_retained_runs.unwrap_or_default(),
        timeout_minutes: l.timeout_minutes.unwrap_or_default(),
        idle_timeout_sec: l.idle_timeout_sec.unwrap_or_default(),
        total_runs: l.total_runs.unwrap_or_default(),
        successful_runs: l.successful_runs.unwrap_or_default(),
        failed_runs: l.failed_runs.unwrap_or_default(),
        active_run_count: l.active_run_count.unwrap_or_default(),
        last_run_at: l.last_run_at.clone(),
        created_at: l.created_at.clone().unwrap_or_default(),
        updated_at: l.updated_at.clone().unwrap_or_default(),
        cron_expression: l.cron_expression.clone(),
        callback_url: l.callback_url.clone(),
        repository_id: l.repository_id,
        runner_id: l.runner_id,
        branch_name: l.branch_name.clone(),
        ticket_id: l.ticket_id,
        credential_profile_id: l.credential_profile_id,
        avg_duration_sec: l.avg_duration_sec,
        used_env_bundles: l.used_env_bundles.clone(),
        ..Default::default()
    }
}

fn run_to_proto(r: &LoopRunData) -> ProtoLoopRun {
    ProtoLoopRun {
        id: r.id,
        run_number: r.run_number.unwrap_or_default(),
        status: r.status.clone(),
        pod_key: r.pod_key.clone(),
        started_at: r.started_at.clone(),
        completed_at: r.completed_at.clone(),
        error_message: r.error_message.clone(),
        created_at: r.created_at.clone().unwrap_or_default(),
        ..Default::default()
    }
}

impl WasmLoopState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmLoopState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::with_storage(crate::new_memory_backend()))),
        }
    }

    pub fn loops_json(&self) -> String {
        serde_json::to_string(self.state.read().loops.get_loops()).unwrap_or_default()
    }

    pub fn current_loop_json(&self) -> JsValue {
        match self.state.read().loops.get_current_loop() {
            Some(l) => JsValue::from_str(
                &serde_json::to_string(l).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn runs_json(&self) -> String {
        serde_json::to_string(self.state.read().loops.get_runs()).unwrap_or_default()
    }

    pub fn get_loop_by_slug_json(&self, slug: &str) -> JsValue {
        match self.state.read().loops.get_loop_by_slug(slug) {
            Some(l) => JsValue::from_str(
                &serde_json::to_string(l).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    // Read side (B, zero-JSON): prost-encode state into the same wrappers the
    // mutators decode, so the shared selectors decode bytes uniformly.
    pub fn loops_bytes(&self) -> Vec<u8> {
        let loops = self.state.read().loops.get_loops().iter().map(loop_to_proto).collect();
        ReplaceCachedLoopsRequest { loops }.encode_to_vec()
    }

    pub fn runs_bytes(&self) -> Vec<u8> {
        let runs = self.state.read().loops.get_runs().iter().map(run_to_proto).collect();
        ReplaceCachedRunsRequest { runs }.encode_to_vec()
    }

    pub fn current_loop_bytes(&self) -> Vec<u8> {
        match self.state.read().loops.get_current_loop() {
            Some(l) => SetCurrentLoopRequest { r#loop: Some(loop_to_proto(l)) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    // Fetch→state (B): decode wire ListLoops/ListRuns response + fold into state
    // via loop_from_proto/run_from_proto — no TS loopToProtoLoop/fromProtoLoop.
    pub fn apply_fetched_loops(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListLoopsResponse::decode(resp_bytes).map_err(decode_err)?;
        let loops = resp.items.into_iter().map(loop_from_proto).collect();
        self.state.write().loops.set_loops(loops);
        Ok(())
    }

    pub fn apply_fetched_runs(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListRunsResponse::decode(resp_bytes).map_err(decode_err)?;
        let runs = resp.items.into_iter().map(run_from_proto).collect();
        self.state.write().loops.set_runs(runs);
        Ok(())
    }

    pub fn apply_appended_runs(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListRunsResponse::decode(resp_bytes).map_err(decode_err)?;
        let runs: Vec<LoopRunData> = resp.items.into_iter().map(run_from_proto).collect();
        self.state.write().loops.append_runs(runs);
        Ok(())
    }

    pub fn set_current_loop(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = SetCurrentLoopRequest::decode(req_bytes).map_err(decode_err)?;
        let loop_data = req.r#loop.map(loop_from_proto);
        self.state.write().loops.set_current_loop(loop_data);
        Ok(())
    }

    // Fetch→state (B): decode the full wire GetLoop response (Loop) + fold via
    // loop_from_proto — no TS loopToProtoLoop round-trip (which dropped the
    // proto-only fields the lossy LoopData cannot carry).
    pub fn apply_fetched_current_loop(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let proto = ProtoLoop::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().loops.set_current_loop(Some(loop_from_proto(proto)));
        Ok(())
    }

    pub fn clear_current_loop(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let _ = ClearCurrentLoopRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().loops.set_current_loop(None);
        Ok(())
    }

    pub fn patch_loop_from_action(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchLoopFromActionRequest::decode(req_bytes).map_err(decode_err)?;
        let loop_data = req.r#loop.ok_or_else(|| JsValue::from_str("missing loop"))?;
        self.state.write().loops.update_loop(&req.slug, loop_from_proto(loop_data));
        Ok(())
    }

    pub fn insert_loop_run(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertLoopRunRequest::decode(req_bytes).map_err(decode_err)?;
        let run = req.run.ok_or_else(|| JsValue::from_str("missing run"))?;
        self.state.write().loops.add_run(run_from_proto(run));
        Ok(())
    }

    pub fn patch_loop_run_status(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchLoopRunStatusRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().loops.update_run_status(req.run_id, &req.status);
        Ok(())
    }

    pub fn clear_loop_runs(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let _ = ClearLoopRunsRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().loops.clear_runs();
        Ok(())
    }
}
