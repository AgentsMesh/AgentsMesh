use std::sync::Arc;

use agentsmesh_state::acp_types::*;
use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_acp_state_v1::{
    AddPermissionRequestRequest, UpdateConfigurationRequest, UpdatePlanRequest,
    UpdateToolCallRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmAcpSessionManager {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmAcpSessionManager {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmAcpSessionManager {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::new())),
        }
    }

    pub fn get_session_json(&self, pod_key: &str) -> JsValue {
        match self.state.read().acp.get_session(pod_key) {
            Some(s) => JsValue::from_str(
                &serde_json::to_string(s).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn add_content_chunk(&self, pod_key: &str, text: &str, role: &str) {
        self.state.write().acp.add_content_chunk(pod_key, text, role);
    }

    pub fn mark_last_message_complete(&self, pod_key: &str) {
        self.state.write().acp.mark_last_message_complete(pod_key);
    }

    pub fn update_tool_call(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = UpdateToolCallRequest::decode(req_bytes).map_err(decode_err)?;
        if let Ok(tc) = serde_json::from_str::<AcpToolCall>(&req.tool_call_json) {
            self.state.write().acp.update_tool_call(&req.pod_key, tc);
        }
        Ok(())
    }

    pub fn set_tool_call_result(
        &self,
        pod_key: &str,
        tool_call_id: &str,
        success: bool,
        result_text: Option<String>,
        error_message: Option<String>,
    ) {
        self.state.write().acp.set_tool_call_result(
            pod_key,
            tool_call_id,
            success,
            result_text,
            error_message,
        );
    }

    pub fn update_plan(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = UpdatePlanRequest::decode(req_bytes).map_err(decode_err)?;
        if let Ok(steps) = serde_json::from_str::<Vec<AcpPlanStep>>(&req.steps_json) {
            self.state.write().acp.update_plan(&req.pod_key, steps);
        }
        Ok(())
    }

    pub fn add_thinking(&self, pod_key: &str, text: &str) {
        self.state.write().acp.add_thinking(pod_key, text);
    }

    pub fn add_permission_request(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = AddPermissionRequestRequest::decode(req_bytes).map_err(decode_err)?;
        if let Ok(perm) = serde_json::from_str::<AcpPermissionRequest>(&req.request_json) {
            self.state.write().acp.add_permission_request(&req.pod_key, perm);
        }
        Ok(())
    }

    pub fn remove_permission_request(&self, pod_key: &str, request_id: &str) {
        self.state.write().acp.remove_permission_request(pod_key, request_id);
    }

    pub fn update_session_state(&self, pod_key: &str, state_str: &str) {
        let s = AcpState::from_str_lossy(state_str);
        self.state.write().acp.update_session_state(pod_key, s);
    }

    pub fn add_log(&self, pod_key: &str, level: &str, message: &str) {
        self.state.write().acp.add_log(pod_key, level, message);
    }

    pub fn update_configuration(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = UpdateConfigurationRequest::decode(req_bytes).map_err(decode_err)?;
        match serde_json::from_str::<AcpConfiguration>(&req.config_json) {
            Ok(cfg) => self.state.write().acp.update_configuration(&req.pod_key, cfg),
            Err(e) => tracing::warn!(
                target: "acp",
                pod_key = %req.pod_key,
                error = %e,
                "update_configuration: failed to parse config_json"
            ),
        }
        Ok(())
    }

    pub fn clear_session(&self, pod_key: &str) {
        self.state.write().acp.clear_session(pod_key);
    }
}
