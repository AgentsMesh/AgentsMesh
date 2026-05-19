use std::sync::{Arc, RwLock};

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::blockstore_state::BlockstoreState;
use agentsmesh_types::{
    ApplyOpsRequest, ApplyOpsResult, Block, BlockOp, OpEnvelope, OpKind,
    SearchHit, SemanticSearchRequest, Workspace,
};
use serde_json::Value;

pub struct BlockstoreService {
    client: Arc<ApiClient>,
    state: RwLock<BlockstoreState>,
}

impl BlockstoreService {
    pub fn new(client: Arc<ApiClient>, state: BlockstoreState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub async fn apply_ops(&self, req_json: &str) -> Result<String, String> {
        let req: ApplyOpsRequest = serde_json::from_str(req_json)
            .map_err(|e| format!("invalid ApplyOpsRequest JSON: {e}"))?;
        let res: ApplyOpsResult = self.client.blocks_apply_ops(&req).await
            .map_err(crate::wire)?;
        self.apply_local_ops(&req, &res);
        serde_json::to_string(&res).map_err(crate::wire)
    }

    fn apply_local_ops(&self, req: &ApplyOpsRequest, res: &ApplyOpsResult) {
        if res.was_replay { return; }
        let mut state = self.state.write().unwrap();
        let ts = chrono_like_now();
        for (idx, env) in req.ops.iter().enumerate() {
            let Some(&op_id) = res.op_ids.get(idx) else { continue };
            // Ref-level ops (AddRef/RemoveRef/UpdateRef) carry a server-assigned
            // ref_id that the client cannot synthesize. Applying locally with
            // op_id as a stand-in creates a ghost ref; when the authoritative
            // op arrives via WS/catchup the real ref_id is different, so the
            // ghost isn't replaced — it accumulates and causes duplicate nest
            // children (same to_id under two distinct ref ids). Skip locally;
            // the UI reflects the mutation once the server broadcasts the real op.
            if matches!(env.op, OpKind::AddRef | OpKind::RemoveRef | OpKind::UpdateRef) {
                continue;
            }
            let synthetic = synthesize_op(&req.workspace_id, op_id, env, &ts);
            state.apply_remote_op(&synthetic);
        }
    }

    pub async fn list_workspaces(&self) -> Result<String, String> {
        let list: Vec<Workspace> = self.client.blocks_list_workspaces().await
            .map_err(crate::wire)?;
        self.state.write().unwrap().replace_workspaces(list.clone());
        serde_json::to_string(&serde_json::json!({ "workspaces": list }))
            .map_err(crate::wire)
    }

    pub async fn ensure_default_workspace(&self) -> Result<String, String> {
        let ws: Workspace = self.client.blocks_ensure_default_workspace().await
            .map_err(crate::wire)?;
        self.state.write().unwrap().upsert_workspace(ws.clone());
        serde_json::to_string(&ws).map_err(crate::wire)
    }

    pub async fn load_subtree(&self, workspace_id: &str, root_id: &str) -> Result<(), String> {
        let res = self.client.blocks_get_subtree(workspace_id, root_id, 64).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for b in res.blocks { state.upsert_block(b); }
        for r in res.refs { state.upsert_ref(r); }
        if state.last_op_id.get(workspace_id).is_none() {
            state.set_last_op_id(workspace_id, 0);
        }
        Ok(())
    }

    pub async fn load_type_defs(&self, workspace_id: &str) -> Result<(), String> {
        let blocks: Vec<Block> = self.client.blocks_list_type_defs(workspace_id).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for b in blocks { state.upsert_block(b); }
        Ok(())
    }

    pub async fn catchup(&self, workspace_id: &str) -> Result<(), String> {
        let after = self.state.read().unwrap().get_last_op_id(workspace_id);
        let ops: Vec<BlockOp> = self.client.blocks_catchup_ops(workspace_id, after, 500).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for op in &ops { state.apply_remote_op(op); }
        Ok(())
    }

    pub async fn semantic_search(&self, workspace_id: &str, req_json: &str) -> Result<String, String> {
        let req: SemanticSearchRequest = serde_json::from_str(req_json)
            .map_err(|e| format!("invalid search request: {e}"))?;
        let hits: Vec<SearchHit> = self.client.blocks_semantic_search(workspace_id, &req).await
            .map_err(crate::wire)?;
        serde_json::to_string(&serde_json::json!({ "hits": hits })).map_err(crate::wire)
    }

    pub fn apply_remote_op(&self, op_json: &str) -> Result<(), String> {
        let op: BlockOp = serde_json::from_str(op_json)
            .map_err(|e| format!("invalid op JSON: {e}"))?;
        self.state.write().unwrap().apply_remote_op(&op);
        Ok(())
    }

    pub fn workspaces_json(&self) -> String {
        self.state.read().unwrap().workspaces_json()
    }

    pub fn get_block_json(&self, id: &str) -> Option<String> {
        self.state.read().unwrap().get_block_json(id)
    }

    pub fn list_children_json(&self, parent_id: &str) -> String {
        self.state.read().unwrap().list_children_json(parent_id)
    }

    pub fn list_backlinks_json(&self, target_id: &str) -> String {
        self.state.read().unwrap().list_backlinks_json(target_id)
    }

    pub fn type_defs_json(&self, workspace_id: &str) -> String {
        self.state.read().unwrap().type_defs_json(workspace_id)
    }

    pub fn last_op_id(&self, workspace_id: &str) -> i64 {
        self.state.read().unwrap().get_last_op_id(workspace_id)
    }

    pub fn set_last_op_id(&self, workspace_id: &str, id: i64) {
        self.state.write().unwrap().set_last_op_id(workspace_id, id);
    }

    pub fn blocks_json(&self) -> String {
        self.state.read().unwrap().blocks_json()
    }

    pub fn refs_json(&self) -> String {
        self.state.read().unwrap().refs_json()
    }

    pub fn nest_children_json(&self) -> String {
        self.state.read().unwrap().nest_children_json()
    }

    pub fn backlinks_json(&self) -> String {
        self.state.read().unwrap().backlinks_json()
    }

    pub fn last_op_ids_json(&self) -> String {
        self.state.read().unwrap().last_op_ids_json()
    }
}

fn synthesize_op(workspace_id: &str, id: i64, env: &OpEnvelope, applied_at: &str) -> BlockOp {
    let mut forward = env.payload.clone();
    if matches!(env.op, OpKind::AddRef) {
        if let Value::Object(ref mut m) = forward {
            m.entry("id").or_insert(Value::from(id));
        }
    }
    BlockOp {
        id,
        workspace_id: workspace_id.to_string(),
        idempotency_key: None,
        actor_type: agentsmesh_types::ActorType::User,
        actor_id: 0,
        op: env.op.clone(),
        target_block: None,
        target_ref: None,
        payload: env.payload.clone(),
        forward,
        inverse: Value::Null,
        parent_op_id: None,
        applied_at: applied_at.to_string(),
    }
}

fn chrono_like_now() -> String {
    // Services crate is WASM-compatible: avoid chrono. The server replaces this
    // with its authoritative timestamp via catchup_ops.
    "".into()
}
