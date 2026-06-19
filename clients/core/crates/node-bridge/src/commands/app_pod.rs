use napi_derive::napi;

use agentsmesh_types::proto_pod_v1::ListPodsResponse;
use agentsmesh_types::proto_pod_state_v1::{
    AppendCachedPodsRequest, InsertCreatedPodRequest, MarkPodTerminatedRequest,
    PatchPodPerpetualRequest, ReplaceCachedPodsRequest,
};
use prost::Message as _;

use crate::AppState;

// Pod state surface over the shared `runtime.state` (the dispatch-hook SSOT),
// mirroring the channel pattern in `app_channel.rs`. The legacy `pod_*`
// commands read the per-service `PodService` cache, disjoint from the realtime
// dispatch target; these `app_*` commands keep `runtime.state.pods` fed by both
// fetch baseline and EventBus dispatch so a post-dispatch snapshot is complete.
fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
}

#[napi]
impl AppState {
    // ── Snapshot reads ──

    #[napi]
    pub fn app_pods_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().pods.pods()).unwrap_or_default()
    }

    // Single pod for the surgical realtime mirror. Empty string when the pod
    // isn't in runtime.state (e.g. a brand-new pod whose full payload arrives
    // via refetch) — the renderer then skips the upsert and lets fetchPod fill it.
    #[napi]
    pub fn app_get_pod_json(&self, pod_key: String) -> String {
        match self.runtime.state.read().pods.get_pod(&pod_key) {
            Some(pod) => serde_json::to_string(pod).unwrap_or_default(),
            None => String::new(),
        }
    }

    // Proto-bytes variant for the realtime snapshot mirror — the renderer decodes
    // via fromBinary + podToCache (the fetch projection) for shape parity, not by
    // merging prost serde JSON. app_get_pod_json stays for the e2e list/detail
    // consistency probe.
    #[napi]
    pub fn app_get_pod_proto(&self, pod_key: String) -> Vec<u8> {
        match self.runtime.state.read().pods.get_pod(&pod_key) {
            Some(pod) => pod.encode_to_vec(),
            None => Vec::new(),
        }
    }

    // ── Fetch / user-action mirror mutators → runtime.state baseline ──

    // Fetch→state (desktop main): decode wire ListPodsResponse + fold into
    // state, mirroring the wasm apply_fetched_pods. Wire Pod == cache Pod, so
    // it's identity — the renderer fans the same wire bytes here.
    #[napi]
    pub fn app_pod_apply_fetched_pods(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListPodsResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().pods.set_pods(resp.items);
        Ok(())
    }

    #[napi]
    pub fn app_pod_apply_appended_pods(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListPodsResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        let mut guard = self.runtime.state.write();
        for pod in resp.items {
            guard.pods.upsert_pod(pod, None);
        }
        Ok(())
    }

    #[napi]
    pub fn app_pod_insert_created(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = InsertCreatedPodRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(pod) = req.pod {
            let ts = if req.client_timestamp_ms == 0 { None } else { Some(req.client_timestamp_ms) };
            self.runtime.state.write().pods.upsert_pod(pod, ts);
        }
        Ok(())
    }

    #[napi]
    pub fn app_pod_mark_terminated(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = MarkPodTerminatedRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .pods
            .update_pod_status(&req.pod_key, "terminated", None, None, None, None);
        Ok(())
    }

    #[napi]
    pub fn app_pod_patch_perpetual(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = PatchPodPerpetualRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .pods
            .patch_perpetual(&req.pod_key, req.perpetual);
        Ok(())
    }

    #[napi]
    pub fn app_pod_remove(&self, pod_key: String) {
        self.runtime.state.write().pods.remove_pod(&pod_key);
    }
}
