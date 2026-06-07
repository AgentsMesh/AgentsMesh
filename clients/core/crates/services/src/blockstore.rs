use std::sync::{Arc, RwLock};

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::blockstore_state::BlockstoreState;
use agentsmesh_state::blockstore_types::{
    ApplyOpsRequest, ApplyOpsResult, BlockOp, OpKind, Workspace,
};
use agentsmesh_types::proto_blockstore_v1 as blockstore_proto;
use agentsmesh_types::proto_blockstore_state_v1 as blockstore_state_proto;
use prost::Message;

use crate::blockstore_proto_convert::{
    block_from_proto, block_op_from_proto, block_ref_from_proto, chrono_like_now,
    op_envelope_from_proto, synthesize_op, workspace_from_proto,
};

pub struct BlockstoreService {
    client: Arc<ApiClient>,
    state: RwLock<BlockstoreState>,
}

impl BlockstoreService {
    pub fn new(client: Arc<ApiClient>, state: BlockstoreState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub(crate) fn client(&self) -> &ApiClient { &self.client }

    fn org_slug(&self) -> String { self.client.current_org_slug() }

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

    // ── Fetches (cache + return JSON) ──

    pub async fn list_workspaces(&self) -> Result<String, String> {
        let req = blockstore_proto::ListWorkspacesRequest { org_slug: self.org_slug() };
        tracing::debug!(target: "blockstore", org_slug = %req.org_slug, "list workspaces");
        let resp = self.client.blockstore_list_workspaces_connect(&req).await
            .map_err(crate::wire)?;
        let list: Vec<Workspace> = resp.items.into_iter().map(workspace_from_proto).collect();
        self.state.write().unwrap().replace_workspaces(list.clone());
        serde_json::to_string(&serde_json::json!({ "workspaces": list }))
            .map_err(crate::wire)
    }

    pub async fn ensure_default_workspace(&self) -> Result<String, String> {
        let req = blockstore_proto::EnsureDefaultWorkspaceRequest { org_slug: self.org_slug() };
        tracing::info!(target: "blockstore", org_slug = %req.org_slug, "ensure default workspace");
        let resp = self.client.blockstore_ensure_default_workspace_connect(&req).await
            .map_err(crate::wire)?;
        let ws = workspace_from_proto(resp);
        self.state.write().unwrap().upsert_workspace(ws.clone());
        serde_json::to_string(&ws).map_err(crate::wire)
    }

    pub async fn load_subtree(&self, workspace_id: &str, root_id: &str) -> Result<(), String> {
        let req = blockstore_proto::GetSubtreeRequest {
            org_slug: self.org_slug(),
            workspace_id: workspace_id.to_string(),
            root_id: root_id.to_string(),
            max_depth: Some(64),
        };
        tracing::debug!(target: "blockstore", workspace_id, root_id, "load subtree");
        let resp = self.client.blockstore_get_subtree_connect(&req).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for b in resp.blocks { state.upsert_block(block_from_proto(b)?); }
        for r in resp.refs { state.upsert_ref(block_ref_from_proto(r)?); }
        // Seed watermark so WS subscription recognises this workspace.
        if state.last_op_id.get(workspace_id).is_none() {
            state.set_last_op_id(workspace_id, 0);
        }
        Ok(())
    }

    pub async fn load_type_defs(&self, workspace_id: &str) -> Result<(), String> {
        let req = blockstore_proto::ListTypeDefsRequest {
            org_slug: self.org_slug(),
            workspace_id: workspace_id.to_string(),
        };
        tracing::debug!(target: "blockstore", workspace_id, "load type defs");
        let resp = self.client.blockstore_list_type_defs_connect(&req).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for b in resp.items { state.upsert_block(block_from_proto(b)?); }
        Ok(())
    }

    pub async fn catchup(&self, workspace_id: &str) -> Result<(), String> {
        let after = self.state.read().unwrap().get_last_op_id(workspace_id);
        let req = blockstore_proto::StreamOpsRequest {
            org_slug: self.org_slug(),
            workspace_id: workspace_id.to_string(),
            after: Some(after),
            limit: Some(500),
        };
        tracing::debug!(target: "blockstore", workspace_id, after, "catchup ops");
        let resp = self.client.blockstore_stream_ops_connect(&req).await
            .map_err(crate::wire)?;
        let mut state = self.state.write().unwrap();
        for op in resp.items {
            let op = block_op_from_proto(op)?;
            state.apply_remote_op(&op);
        }
        Ok(())
    }

    // ── Remote op stream (realtime) ──

    pub fn apply_remote_op(&self, op_json: &str) -> Result<(), String> {
        let op: BlockOp = serde_json::from_str(op_json)
            .map_err(|e| format!("invalid op JSON: {e}"))?;
        self.state.write().unwrap().apply_remote_op(&op);
        Ok(())
    }

    // ── Getters (sync, return JSON) ──

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

    // ── Bulk state population (consumed by JS Connect adapter callers who
    // fetched via the binary wire and need to push results into the local
    // cache). Each method accepts a prost-encoded request envelope carrying
    // the typed proto.blockstore.v1.* container. Per-blocktype data/meta
    // fields inside Block / BlockRef remain JSON strings (200+ subschemas).

    pub fn replace_workspaces(&self, req_bytes: &[u8]) -> Result<(), String> {
        let req = blockstore_state_proto::ReplaceWorkspacesRequest::decode(req_bytes)
            .map_err(|e| format!("decode ReplaceWorkspacesRequest: {e}"))?;
        let list: Vec<Workspace> = req.workspaces.into_iter().map(workspace_from_proto).collect();
        self.state.write().unwrap().replace_workspaces(list);
        Ok(())
    }

    pub fn upsert_workspace(&self, req_bytes: &[u8]) -> Result<(), String> {
        let req = blockstore_state_proto::UpsertWorkspaceRequest::decode(req_bytes)
            .map_err(|e| format!("decode UpsertWorkspaceRequest: {e}"))?;
        let ws = req.workspace.ok_or_else(|| "missing workspace".to_string())?;
        self.state.write().unwrap().upsert_workspace(workspace_from_proto(ws));
        Ok(())
    }

    pub fn upsert_blocks(&self, req_bytes: &[u8]) -> Result<(), String> {
        let req = blockstore_state_proto::UpsertBlocksRequest::decode(req_bytes)
            .map_err(|e| format!("decode UpsertBlocksRequest: {e}"))?;
        let mut state = self.state.write().unwrap();
        for b in req.blocks {
            state.upsert_block(block_from_proto(b)?);
        }
        Ok(())
    }

    pub fn upsert_refs(&self, req_bytes: &[u8]) -> Result<(), String> {
        let req = blockstore_state_proto::UpsertRefsRequest::decode(req_bytes)
            .map_err(|e| format!("decode UpsertRefsRequest: {e}"))?;
        let mut state = self.state.write().unwrap();
        for r in req.refs {
            state.upsert_ref(block_ref_from_proto(r)?);
        }
        Ok(())
    }

    /// Project an ApplyOps envelope/result pair into the local cache.
    /// Mirrors `apply_ops_connect`'s side effect so JS callers using the
    /// Connect path keep the same local-replay semantics.
    pub fn project_local_ops(&self, req_bytes: &[u8]) -> Result<(), String> {
        let envelope = blockstore_state_proto::ProjectLocalOpsRequest::decode(req_bytes)
            .map_err(|e| format!("decode ProjectLocalOpsRequest: {e}"))?;
        let proto_req = envelope.request.ok_or_else(|| "missing request".to_string())?;
        let proto_res = envelope.result.ok_or_else(|| "missing result".to_string())?;
        let req = ApplyOpsRequest {
            workspace_id: proto_req.workspace_id,
            ops: proto_req.ops.iter().map(op_envelope_from_proto).collect::<Result<_, String>>()?,
            idempotency_key: proto_req.idempotency_key,
            parent_op_id: proto_req.parent_op_id,
        };
        let res = ApplyOpsResult {
            op_ids: proto_res.op_ids,
            was_replay: proto_res.was_replay,
            parent_op_id: proto_res.parent_op_id,
        };
        self.apply_local_ops(&req, &res);
        Ok(())
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

// =============================================================================
// Connect-RPC bridge methods. Binary in (prost-encoded), binary out — same wire
// the wasm/node-bridge layers speak.
// =============================================================================

macro_rules! connect_bridge {
    ($name:ident, $req:ident, $client_call:ident) => {
        pub async fn $name(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
            let req = blockstore_proto::$req::decode(request_bytes)
                .map_err(|e| format!("decode {}: {e}", stringify!($req)))?;
            tracing::debug!(target: "blockstore", rpc = stringify!($req));
            let resp = self.client().$client_call(&req).await.map_err(crate::wire)?;
            Ok(resp.encode_to_vec())
        }
    };
}

impl BlockstoreService {
    connect_bridge!(apply_ops_connect, ApplyOpsRequest, blockstore_apply_ops_connect);
    connect_bridge!(list_workspaces_connect, ListWorkspacesRequest, blockstore_list_workspaces_connect);
    connect_bridge!(ensure_default_workspace_connect, EnsureDefaultWorkspaceRequest, blockstore_ensure_default_workspace_connect);
    connect_bridge!(create_workspace_connect, CreateWorkspaceRequest, blockstore_create_workspace_connect);
    connect_bridge!(delete_workspace_connect, DeleteWorkspaceRequest, blockstore_delete_workspace_connect);
    connect_bridge!(get_block_connect, GetBlockRequest, blockstore_get_block_connect);
    connect_bridge!(list_children_connect, ListChildrenRequest, blockstore_list_children_connect);
    connect_bridge!(list_backlinks_connect, ListBacklinksRequest, blockstore_list_backlinks_connect);
    connect_bridge!(get_subtree_connect, GetSubtreeRequest, blockstore_get_subtree_connect);
    connect_bridge!(stream_ops_connect, StreamOpsRequest, blockstore_stream_ops_connect);
    connect_bridge!(export_workspace_connect, ExportWorkspaceRequest, blockstore_export_workspace_connect);
    connect_bridge!(list_type_defs_connect, ListTypeDefsRequest, blockstore_list_type_defs_connect);
    connect_bridge!(get_block_at_connect, GetBlockAtRequest, blockstore_get_block_at_connect);
    connect_bridge!(semantic_search_connect, SemanticSearchRequest, blockstore_semantic_search_connect);
    connect_bridge!(memory_retrieve_connect, MemoryRetrieveRequest, blockstore_memory_retrieve_connect);
}
