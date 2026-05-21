use agentsmesh_state::acp_session::AcpSessionManager;
use agentsmesh_state::acp_types::*;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmAcpSessionManager {
    inner: AcpSessionManager,
}

#[wasm_bindgen]
impl WasmAcpSessionManager {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            inner: AcpSessionManager::new(),
        }
    }

    pub fn get_session_json(&self, pod_key: &str) -> JsValue {
        match self.inner.get_session(pod_key) {
            Some(s) => JsValue::from_str(
                &serde_json::to_string(s).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn add_content_chunk(
        &mut self,
        pod_key: &str,
        text: &str,
        role: &str,
    ) {
        self.inner.add_content_chunk(pod_key, text, role);
    }

    pub fn mark_last_message_complete(&mut self, pod_key: &str) {
        self.inner.mark_last_message_complete(pod_key);
    }

    pub fn update_tool_call(&mut self, pod_key: &str, tool_call_json: &str) {
        if let Ok(tc) = serde_json::from_str::<AcpToolCall>(tool_call_json) {
            self.inner.update_tool_call(pod_key, tc);
        }
    }

    pub fn set_tool_call_result(
        &mut self,
        pod_key: &str,
        tool_call_id: &str,
        success: bool,
        result_text: Option<String>,
        error_message: Option<String>,
    ) {
        self.inner.set_tool_call_result(
            pod_key,
            tool_call_id,
            success,
            result_text,
            error_message,
        );
    }

    pub fn update_plan(&mut self, pod_key: &str, steps_json: &str) {
        if let Ok(steps) =
            serde_json::from_str::<Vec<AcpPlanStep>>(steps_json)
        {
            self.inner.update_plan(pod_key, steps);
        }
    }

    pub fn add_thinking(&mut self, pod_key: &str, text: &str) {
        self.inner.add_thinking(pod_key, text);
    }

    pub fn add_permission_request(
        &mut self,
        pod_key: &str,
        request_json: &str,
    ) {
        if let Ok(req) =
            serde_json::from_str::<AcpPermissionRequest>(request_json)
        {
            self.inner.add_permission_request(pod_key, req);
        }
    }

    pub fn remove_permission_request(
        &mut self,
        pod_key: &str,
        request_id: &str,
    ) {
        self.inner.remove_permission_request(pod_key, request_id);
    }

    pub fn update_session_state(&mut self, pod_key: &str, state_str: &str) {
        let state = AcpState::from_str_lossy(state_str);
        self.inner.update_session_state(pod_key, state);
    }

    pub fn add_log(&mut self, pod_key: &str, level: &str, message: &str) {
        self.inner.add_log(pod_key, level, message);
    }

    pub fn update_configuration(&mut self, pod_key: &str, config_json: &str) {
        if let Ok(cfg) = serde_json::from_str::<AcpConfiguration>(config_json) {
            self.inner.update_configuration(pod_key, cfg);
        }
    }

    pub fn clear_session(&mut self, pod_key: &str) {
        self.inner.clear_session(pod_key);
    }
}
