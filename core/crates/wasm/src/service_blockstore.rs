use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::BlockstoreService;
use agentsmesh_state::blockstore_state::BlockstoreState;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmBlockstoreService(pub(crate) BlockstoreService);

#[wasm_bindgen]
impl WasmBlockstoreService {
    pub(crate) fn new(client: Arc<ApiClient>, state: BlockstoreState) -> Self {
        Self(BlockstoreService::new(client, state))
    }

    // ── Mutations / fetches ──

    pub async fn apply_ops(&self, req_json: String) -> Result<String, String> {
        self.0.apply_ops(&req_json).await
    }

    pub async fn list_workspaces(&self) -> Result<String, String> {
        self.0.list_workspaces().await
    }

    pub async fn ensure_default_workspace(&self) -> Result<String, String> {
        self.0.ensure_default_workspace().await
    }

    pub async fn load_subtree(&self, workspace_id: String, root_id: String) -> Result<(), String> {
        self.0.load_subtree(&workspace_id, &root_id).await
    }

    pub async fn load_type_defs(&self, workspace_id: String) -> Result<(), String> {
        self.0.load_type_defs(&workspace_id).await
    }

    pub async fn catchup(&self, workspace_id: String) -> Result<(), String> {
        self.0.catchup(&workspace_id).await
    }

    pub async fn semantic_search(
        &self, workspace_id: String, req_json: String,
    ) -> Result<String, String> {
        self.0.semantic_search(&workspace_id, &req_json).await
    }

    pub fn apply_remote_op(&self, op_json: &str) -> Result<(), String> {
        self.0.apply_remote_op(op_json)
    }

    // ── Sync getters ──

    pub fn workspaces_json(&self) -> String { self.0.workspaces_json() }

    pub fn get_block_json(&self, id: &str) -> JsValue {
        match self.0.get_block_json(id) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn list_children_json(&self, parent_id: &str) -> String {
        self.0.list_children_json(parent_id)
    }

    pub fn list_backlinks_json(&self, target_id: &str) -> String {
        self.0.list_backlinks_json(target_id)
    }

    pub fn type_defs_json(&self, workspace_id: &str) -> String {
        self.0.type_defs_json(workspace_id)
    }

    pub fn last_op_id(&self, workspace_id: &str) -> i64 {
        self.0.last_op_id(workspace_id)
    }

    pub fn set_last_op_id(&self, workspace_id: &str, id: i64) {
        self.0.set_last_op_id(workspace_id, id);
    }

    pub fn blocks_json(&self) -> String { self.0.blocks_json() }
    pub fn refs_json(&self) -> String { self.0.refs_json() }
    pub fn nest_children_json(&self) -> String { self.0.nest_children_json() }
    pub fn backlinks_json(&self) -> String { self.0.backlinks_json() }
    pub fn last_op_ids_json(&self) -> String { self.0.last_op_ids_json() }
}
