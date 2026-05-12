use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::PodService;
use agentsmesh_state::pod_state::PodState;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmPodService(pub(crate) PodService);

#[wasm_bindgen]
impl WasmPodService {
    pub(crate) fn new(client: Arc<ApiClient>, state: PodState) -> Self {
        Self(PodService::new(client, state))
    }

    pub fn pods_json(&self) -> String { self.0.pods_json() }

    pub fn current_pod_json(&self) -> JsValue {
        match self.0.current_pod_json() {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn get_pod_json(&self, pod_key: &str) -> JsValue {
        match self.0.get_pod_json(pod_key) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn upsert_pod(&self, pod_json: &str, timestamp: Option<i64>) {
        self.0.upsert_pod(pod_json, timestamp);
    }

    pub fn set_pods(&self, pods_json: &str) { self.0.set_pods(pods_json); }

    pub fn set_current_pod(&self, pod_json: &str) { self.0.set_current_pod(pod_json); }

    pub fn update_pod_status(
        &self, pod_key: &str, status: &str,
        agent_status: Option<String>, error_code: Option<String>,
        error_message: Option<String>, timestamp: Option<i64>,
    ) {
        self.0.update_pod_status(pod_key, status, agent_status, error_code, error_message, timestamp);
    }

    pub fn update_pod_title(&self, pod_key: &str, title: &str, timestamp: Option<i64>) {
        self.0.update_pod_title(pod_key, title, timestamp);
    }

    pub fn update_pod_alias(&self, pod_key: &str, alias: &str) {
        self.0.update_pod_alias(pod_key, alias);
    }

    pub fn update_agent_status(&self, pod_key: &str, agent_status: &str) {
        self.0.update_agent_status(pod_key, agent_status);
    }

    pub fn remove_pod(&self, pod_key: &str) { self.0.remove_pod(pod_key); }

    pub async fn fetch_pods(
        &self, status: Option<String>, runner_id: Option<i64>,
        created_by_id: Option<i64>, limit: Option<i64>, offset: Option<i64>,
    ) -> Result<String, String> {
        self.0.fetch_pods(status, runner_id, created_by_id, limit, offset).await
    }

    pub async fn fetch_sidebar_pods(
        &self, filter: &str, user_id: Option<i64>,
    ) -> Result<String, String> {
        self.0.fetch_sidebar_pods(filter, user_id).await
    }

    pub async fn load_more_pods(
        &self, filter: &str, user_id: Option<i64>, offset: i64,
    ) -> Result<String, String> {
        self.0.load_more_pods(filter, user_id, offset).await
    }

    pub async fn fetch_pod(&self, pod_key: &str) -> Result<String, String> {
        self.0.fetch_pod(pod_key).await
    }

    pub async fn create_pod(&self, request_json: &str) -> Result<String, String> {
        self.0.create_pod(request_json).await
    }

    pub async fn terminate_pod(&self, pod_key: &str) -> Result<(), String> {
        self.0.terminate_pod(pod_key).await
    }

    pub async fn update_pod_alias_api(
        &self, pod_key: &str, alias: Option<String>,
    ) -> Result<(), String> {
        self.0.update_pod_alias_api(pod_key, alias).await
    }

    pub async fn get_pod_connection(&self, pod_key: &str) -> Result<String, String> {
        self.0.get_pod_connection(pod_key).await
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes (Uint8Array on the JS
    // side) and returns prost-encoded bytes — TS callers encode via
    // @bufbuild/protobuf .toBinary() and decode via .fromBinary().

    pub async fn list_pods_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_pods_connect(request_bytes).await
    }

    pub async fn get_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_pod_connect(request_bytes).await
    }

    pub async fn create_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.create_pod_connect(request_bytes).await
    }

    pub async fn terminate_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.terminate_pod_connect(request_bytes).await
    }

    pub async fn update_pod_alias_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_pod_alias_connect(request_bytes).await
    }

    pub async fn update_pod_perpetual_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_pod_perpetual_connect(request_bytes).await
    }

    pub async fn get_pod_connection_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_pod_connection_connect(request_bytes).await
    }

    pub async fn send_pod_prompt_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.send_pod_prompt_connect(request_bytes).await
    }

    pub async fn list_pods_by_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_pods_by_ticket_connect(request_bytes).await
    }
}
