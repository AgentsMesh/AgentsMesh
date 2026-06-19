use napi_derive::napi;

use agentsmesh_types::proto_runner_api_v1::{
    GetRunnerResponse, ListAvailableRunnersResponse, ListRunnersResponse,
};
use agentsmesh_types::proto_runner_state_v1::{
    PatchCachedRunnerRequest, RemoveCachedRunnerRequest, ReplaceAvailableRunnersRequest,
    ReplaceCachedRunnersRequest, SetCurrentRunnerRequest,
};
use prost::Message as _;

use crate::AppState;

// Runner state surface over the shared `runtime.state` (dispatch-hook SSOT),
// mirroring app_channel.rs / app_pod.rs. Keeps `runtime.state.runners` fed by
// both fetch baseline and EventBus dispatch so the post-dispatch snapshot
// (main/realtime.ts) reflects the full runner picture for the renderer mirror.
fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
}

#[napi]
impl AppState {
    // ── Snapshot reads ──

    #[napi]
    pub fn app_runners_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().runners.runners()).unwrap_or_default()
    }

    #[napi]
    pub fn app_available_runners_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().runners.available_runners())
            .unwrap_or_default()
    }

    #[napi]
    pub fn app_current_runner_json(&self) -> String {
        match self.runtime.state.read().runners.current_runner() {
            Some(r) => serde_json::to_string(r).unwrap_or_default(),
            None => String::new(),
        }
    }

    // Proto-bytes variants for the realtime snapshot mirror — reuse the *Request
    // wrappers so the renderer decodes via fromBinary + runnerToCache (the fetch
    // projection) for shape parity, not by assigning prost serde JSON.
    #[napi]
    pub fn app_runners_proto(&self) -> Vec<u8> {
        let runners = self.runtime.state.read().runners.runners().to_vec();
        ReplaceCachedRunnersRequest { runners }.encode_to_vec()
    }

    #[napi]
    pub fn app_available_runners_proto(&self) -> Vec<u8> {
        let runners = self.runtime.state.read().runners.available_runners().to_vec();
        ReplaceAvailableRunnersRequest { runners }.encode_to_vec()
    }

    #[napi]
    pub fn app_current_runner_proto(&self) -> Vec<u8> {
        let runner = self.runtime.state.read().runners.current_runner().cloned();
        SetCurrentRunnerRequest { runner }.encode_to_vec()
    }

    // ── Fetch-mirror mutators → runtime.state baseline ──

    #[napi]
    pub fn app_runner_set_current(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = SetCurrentRunnerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().runners.set_current_runner(req.runner);
        Ok(())
    }

    // Fetch→state (B): decode wire ListRunners(Available)Response + fold into
    // runtime.state. Wire Runner == cache Runner, so no conversion.
    #[napi]
    pub fn app_runner_apply_fetched(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListRunnersResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().runners.set_runners(resp.items);
        Ok(())
    }

    #[napi]
    pub fn app_runner_apply_fetched_available(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListAvailableRunnersResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().runners.set_available_runners(resp.items);
        Ok(())
    }

    // Single-object fetch (B): decode wire GetRunnerResponse + set current from
    // its runner field.
    #[napi]
    pub fn app_runner_apply_fetched_current(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = GetRunnerResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().runners.set_current_runner(resp.runner);
        Ok(())
    }

    #[napi]
    pub fn app_runner_patch(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = PatchCachedRunnerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(runner) = req.runner {
            self.runtime.state.write().runners.upsert_runner(runner);
        }
        Ok(())
    }

    #[napi]
    pub fn app_runner_remove(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = RemoveCachedRunnerRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().runners.remove_runner(req.runner_id);
        Ok(())
    }
}
