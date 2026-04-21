use std::cmp::Ordering;
use std::collections::HashMap;

use agentsmesh_types::{Block, BlockOp, BlockRef, Workspace};
use serde::Serialize;
use serde_json::Value;

use crate::blockstore_apply::apply_op;

#[derive(Debug, Default)]
pub struct BlockstoreState {
    pub workspaces: HashMap<String, Workspace>,
    pub blocks: HashMap<String, Block>,
    pub refs: HashMap<i64, BlockRef>,
    /// parent block id → ordered ref ids (sorted by order_key, then ref id)
    pub nest_children: HashMap<String, Vec<i64>>,
    /// target block id → ref ids (non-nest)
    pub backlinks: HashMap<String, Vec<i64>>,
    /// per-workspace op watermark for WS catchup
    pub last_op_id: HashMap<String, i64>,
}

#[derive(Serialize)]
struct SubtreeView<'a> {
    blocks: Vec<&'a Block>,
    refs: Vec<&'a BlockRef>,
}

impl BlockstoreState {
    pub fn new() -> Self {
        Self::default()
    }

    // ── Workspaces ──

    pub fn upsert_workspace(&mut self, ws: Workspace) {
        self.workspaces.insert(ws.id.clone(), ws);
    }

    pub fn replace_workspaces(&mut self, list: Vec<Workspace>) {
        self.workspaces.clear();
        for w in list {
            self.workspaces.insert(w.id.clone(), w);
        }
    }

    // ── Blocks ──

    pub fn upsert_block(&mut self, b: Block) {
        self.blocks.insert(b.id.clone(), b);
    }

    pub fn update_block_fields(&mut self, id: &str, patch: &Value) -> bool {
        let Some(b) = self.blocks.get_mut(id) else { return false };
        if let Some(v) = patch.get("data") { b.data = v.clone(); }
        if let Some(v) = patch.get("text") { b.text = v.as_str().map(str::to_string); }
        if let Some(v) = patch.get("meta") { b.meta = v.clone(); }
        if let Some(v) = patch.get("updated_at").and_then(Value::as_str) {
            b.updated_at = v.to_string();
        }
        true
    }

    pub fn remove_block(&mut self, id: &str) {
        if self.blocks.remove(id).is_none() { return; }
        // Cascade: drop every ref touching this block and clean indexes.
        let doomed: Vec<i64> = self.refs.iter()
            .filter(|(_, r)| r.from_id == id || r.to_id == id)
            .map(|(k, _)| *k).collect();
        for rid in doomed {
            if let Some(r) = self.refs.remove(&rid) {
                self.unindex_ref(&r);
            }
        }
        self.nest_children.remove(id);
        self.backlinks.remove(id);
    }

    // ── Refs ──

    pub fn upsert_ref(&mut self, r: BlockRef) {
        if let Some(prev) = self.refs.get(&r.id).cloned() {
            self.unindex_ref(&prev);
        }
        self.refs.insert(r.id, r.clone());
        self.index_ref(&r);
    }

    pub fn remove_ref(&mut self, ref_id: i64) {
        if let Some(r) = self.refs.remove(&ref_id) {
            self.unindex_ref(&r);
        }
    }

    pub fn update_ref_fields(&mut self, ref_id: i64, patch: &Value) -> bool {
        let Some(prev) = self.refs.get(&ref_id).cloned() else { return false };
        let mut next = prev.clone();
        if let Some(v) = patch.get("from_id").and_then(Value::as_str) { next.from_id = v.into(); }
        if let Some(v) = patch.get("order_key") {
            next.order_key = v.as_str().map(str::to_string);
        }
        if let Some(v) = patch.get("anchor") {
            next.anchor = v.as_str().map(str::to_string);
        }
        if let Some(v) = patch.get("meta") { next.meta = v.clone(); }
        if let Some(v) = patch.get("updated_at").and_then(Value::as_str) {
            next.updated_at = v.to_string();
        }
        self.unindex_ref(&prev);
        self.refs.insert(ref_id, next.clone());
        self.index_ref(&next);
        true
    }

    fn index_ref(&mut self, r: &BlockRef) {
        if r.rel == "nest" {
            let list = self.nest_children.entry(r.from_id.clone()).or_default();
            if !list.contains(&r.id) {
                list.push(r.id);
                let refs = &self.refs;
                list.sort_by(|a, b| compare_order_key(refs.get(a), refs.get(b)));
            }
        } else {
            let list = self.backlinks.entry(r.to_id.clone()).or_default();
            if !list.contains(&r.id) { list.push(r.id); }
        }
    }

    fn unindex_ref(&mut self, r: &BlockRef) {
        let bucket = if r.rel == "nest" { &mut self.nest_children } else { &mut self.backlinks };
        let key = if r.rel == "nest" { &r.from_id } else { &r.to_id };
        if let Some(list) = bucket.get_mut(key) {
            list.retain(|id| *id != r.id);
            if list.is_empty() { bucket.remove(key); }
        }
    }

    // ── Ops ──

    pub fn apply_remote_op(&mut self, op: &BlockOp) {
        apply_op(self, op);
        if op.id > *self.last_op_id.get(&op.workspace_id).unwrap_or(&0) {
            self.last_op_id.insert(op.workspace_id.clone(), op.id);
        }
    }

    pub fn set_last_op_id(&mut self, workspace_id: &str, id: i64) {
        self.last_op_id.insert(workspace_id.to_string(), id);
    }

    // ── Reads (serialized for WASM) ──

    pub fn workspaces_json(&self) -> String {
        serde_json::to_string(&self.workspaces).unwrap_or_else(|_| "{}".into())
    }

    pub fn get_block_json(&self, id: &str) -> Option<String> {
        self.blocks.get(id).map(|b| serde_json::to_string(b).unwrap_or_default())
    }

    pub fn list_children_json(&self, parent_id: &str) -> String {
        let ids = self.nest_children.get(parent_id).cloned().unwrap_or_default();
        let blocks: Vec<&Block> = ids.iter()
            .filter_map(|rid| self.refs.get(rid))
            .filter_map(|r| self.blocks.get(&r.to_id)).collect();
        let refs: Vec<&BlockRef> = ids.iter().filter_map(|rid| self.refs.get(rid)).collect();
        serde_json::to_string(&SubtreeView { blocks, refs }).unwrap_or_else(|_| "{}".into())
    }

    pub fn list_backlinks_json(&self, target_id: &str) -> String {
        let ids = self.backlinks.get(target_id).cloned().unwrap_or_default();
        let refs: Vec<&BlockRef> = ids.iter().filter_map(|rid| self.refs.get(rid)).collect();
        serde_json::to_string(&serde_json::json!({ "refs": refs })).unwrap_or_else(|_| "{}".into())
    }

    pub fn type_defs_json(&self, workspace_id: &str) -> String {
        let blocks: Vec<&Block> = self.blocks.values()
            .filter(|b| b.workspace_id == workspace_id && b.block_type == "block_type_def")
            .collect();
        serde_json::to_string(&serde_json::json!({ "blocks": blocks })).unwrap_or_else(|_| "{}".into())
    }

    pub fn get_last_op_id(&self, workspace_id: &str) -> i64 {
        *self.last_op_id.get(workspace_id).unwrap_or(&0)
    }

    /// Serialise ALL blocks as `{[id]: Block}` for JS-side selectors that
    /// need the whole map (sidebar tree traversal, view filtering).
    pub fn blocks_json(&self) -> String {
        serde_json::to_string(&self.blocks).unwrap_or_else(|_| "{}".into())
    }

    /// Serialise ALL refs as `{[id]: BlockRef}`. Consumers: SortableNest uses
    /// `refs[id].order_key` for before/after insertion heuristics.
    pub fn refs_json(&self) -> String {
        serde_json::to_string(&self.refs).unwrap_or_else(|_| "{}".into())
    }

    /// Serialise the nest-children index as `{[parentId]: [refId]}`.
    pub fn nest_children_json(&self) -> String {
        serde_json::to_string(&self.nest_children).unwrap_or_else(|_| "{}".into())
    }

    /// Serialise the backlinks index as `{[targetId]: [refId]}`.
    pub fn backlinks_json(&self) -> String {
        serde_json::to_string(&self.backlinks).unwrap_or_else(|_| "{}".into())
    }

    /// Serialise per-workspace last-op watermarks as `{[workspaceId]: opId}`.
    /// Used by the reconnect catchup loop that iterates all loaded workspaces.
    pub fn last_op_ids_json(&self) -> String {
        serde_json::to_string(&self.last_op_id).unwrap_or_else(|_| "{}".into())
    }
}

pub fn compare_order_key(a: Option<&BlockRef>, b: Option<&BlockRef>) -> Ordering {
    let ak = a.and_then(|r| r.order_key.as_deref());
    let bk = b.and_then(|r| r.order_key.as_deref());
    match (ak, bk) {
        (Some(x), Some(y)) if x == y => a.map(|r| r.id).unwrap_or(0).cmp(&b.map(|r| r.id).unwrap_or(0)),
        (Some(x), Some(y)) => x.cmp(y),
        (None, None) => a.map(|r| r.id).unwrap_or(0).cmp(&b.map(|r| r.id).unwrap_or(0)),
        (None, Some(_)) => Ordering::Greater,
        (Some(_), None) => Ordering::Less,
    }
}
