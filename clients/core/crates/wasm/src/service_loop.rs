use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::LoopService;
use agentsmesh_state::loop_state::LoopState;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmLoopService(pub(crate) LoopService);

#[wasm_bindgen]
impl WasmLoopService {
    pub(crate) fn new(client: Arc<ApiClient>, state: LoopState) -> Self {
        Self(LoopService::new(client, state))
    }

    pub fn loops_json(&self) -> String { self.0.loops_json() }

    pub fn current_loop_json(&self) -> JsValue {
        match self.0.current_loop_json() {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn runs_json(&self) -> String { self.0.runs_json() }

    pub fn get_loop_by_slug_json(&self, slug: &str) -> JsValue {
        match self.0.get_loop_by_slug_json(slug) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn set_loops(&self, json: &str) { self.0.set_loops(json); }
    pub fn set_current_loop(&self, json: &str) { self.0.set_current_loop(json); }

    pub fn update_loop_local(&self, slug: &str, json: &str) {
        self.0.update_loop_local(slug, json);
    }

    pub fn add_run(&self, json: &str) { self.0.add_run(json); }
    pub fn set_runs(&self, json: &str) { self.0.set_runs(json); }
    pub fn append_runs(&self, json: &str) { self.0.append_runs(json); }

    pub fn update_run_status(&self, run_id: i64, status: &str) {
        self.0.update_run_status(run_id, status);
    }

    pub fn clear_runs(&self) { self.0.clear_runs(); }

    pub async fn fetch_loops(
        &self, status: Option<String>, limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        self.0.fetch_loops(status, limit, offset).await
    }

    pub async fn fetch_loop(&self, slug: &str) -> Result<String, String> {
        self.0.fetch_loop(slug).await
    }

    pub async fn create_loop(&self, request_json: &str) -> Result<String, String> {
        self.0.create_loop(request_json).await
    }

    pub async fn update_loop(&self, slug: &str, request_json: &str) -> Result<String, String> {
        self.0.update_loop(slug, request_json).await
    }

    pub async fn delete_loop(&self, slug: &str) -> Result<(), String> {
        self.0.delete_loop(slug).await
    }

    pub async fn enable_loop(&self, slug: &str) -> Result<String, String> {
        self.0.enable_loop(slug).await
    }

    pub async fn disable_loop(&self, slug: &str) -> Result<String, String> {
        self.0.disable_loop(slug).await
    }

    pub async fn trigger_loop(&self, slug: &str) -> Result<String, String> {
        self.0.trigger_loop(slug).await
    }

    pub async fn fetch_runs(
        &self, slug: &str, status: Option<String>,
        limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        self.0.fetch_runs(slug, status, limit, offset).await
    }

    pub async fn cancel_run(&self, slug: &str, run_id: i64) -> Result<(), String> {
        self.0.cancel_run(slug, run_id).await
    }

    // -------- Connect-RPC (binary wire) --------

    #[wasm_bindgen(js_name = listLoopsConnect)]
    pub async fn list_loops_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_loops_connect(request).await
    }

    #[wasm_bindgen(js_name = getLoopConnect)]
    pub async fn get_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_loop_connect(request).await
    }

    #[wasm_bindgen(js_name = createLoopConnect)]
    pub async fn create_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.create_loop_connect(request).await
    }

    #[wasm_bindgen(js_name = updateLoopConnect)]
    pub async fn update_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_loop_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteLoopConnect)]
    pub async fn delete_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.delete_loop_connect(request).await
    }

    #[wasm_bindgen(js_name = enableLoopConnect)]
    pub async fn enable_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.loop_action_connect("enable", request).await
    }

    #[wasm_bindgen(js_name = disableLoopConnect)]
    pub async fn disable_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.loop_action_connect("disable", request).await
    }

    #[wasm_bindgen(js_name = triggerLoopConnect)]
    pub async fn trigger_loop_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.trigger_loop_connect(request).await
    }

    #[wasm_bindgen(js_name = listRunsConnect)]
    pub async fn list_runs_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_runs_connect(request).await
    }

    #[wasm_bindgen(js_name = cancelRunConnect)]
    pub async fn cancel_run_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.cancel_run_connect(request).await
    }
}
