use agentsmesh_types::{Block, BlockOp, BlockRef, OpKind};
use serde_json::{json, Value};

use crate::blockstore_state::BlockstoreState;

pub fn apply_op(state: &mut BlockstoreState, op: &BlockOp) {
    match op.op {
        OpKind::CreateBlock => apply_create_block(state, op),
        OpKind::UpdateBlock => apply_update_block(state, op),
        OpKind::DeleteBlock => apply_delete_block(state, op),
        OpKind::AddRef => apply_add_ref(state, op),
        OpKind::RemoveRef => apply_remove_ref(state, op),
        OpKind::UpdateRef => apply_update_ref(state, op),
    }
}

fn apply_create_block(state: &mut BlockstoreState, op: &BlockOp) {
    let fwd = &op.forward;
    let Some(id) = fwd.get("id").and_then(Value::as_str) else { return };
    let Some(ty) = fwd.get("type").and_then(Value::as_str) else { return };
    let block = Block {
        id: id.to_string(),
        workspace_id: op.workspace_id.clone(),
        block_type: ty.to_string(),
        data: fwd.get("data").cloned().unwrap_or(json!({})),
        text: fwd.get("text").and_then(Value::as_str).map(str::to_string),
        meta: fwd.get("meta").cloned().unwrap_or(json!({})),
        created_by: op.actor_id,
        created_at: op.applied_at.clone(),
        updated_at: op.applied_at.clone(),
        deleted_at: None,
    };
    state.upsert_block(block);
}

fn apply_update_block(state: &mut BlockstoreState, op: &BlockOp) {
    let fwd = &op.forward;
    let Some(id) = fwd.get("id").and_then(Value::as_str) else { return };
    let mut patch = fwd.clone();
    if let Value::Object(ref mut m) = patch {
        m.insert("updated_at".into(), Value::String(op.applied_at.clone()));
    }
    state.update_block_fields(id, &patch);
}

fn apply_delete_block(state: &mut BlockstoreState, op: &BlockOp) {
    if let Some(id) = op.forward.get("id").and_then(Value::as_str) {
        state.remove_block(id);
    }
}

fn apply_add_ref(state: &mut BlockstoreState, op: &BlockOp) {
    let fwd = &op.forward;
    let Some(id) = fwd.get("id").and_then(Value::as_i64) else { return };
    let Some(from) = fwd.get("from").and_then(Value::as_str) else { return };
    let Some(to) = fwd.get("to").and_then(Value::as_str) else { return };
    let Some(rel) = fwd.get("rel").and_then(Value::as_str) else { return };
    let r = BlockRef {
        id,
        workspace_id: op.workspace_id.clone(),
        from_id: from.to_string(),
        to_id: to.to_string(),
        rel: rel.to_string(),
        order_key: fwd.get("order_key").and_then(Value::as_str).map(str::to_string),
        anchor: fwd.get("anchor").and_then(Value::as_str).map(str::to_string),
        meta: fwd.get("meta").cloned().unwrap_or(json!({})),
        created_by: op.actor_id,
        created_at: op.applied_at.clone(),
        updated_at: op.applied_at.clone(),
    };
    state.upsert_ref(r);
}

fn apply_remove_ref(state: &mut BlockstoreState, op: &BlockOp) {
    if let Some(rid) = op.forward.get("ref_id").and_then(Value::as_i64) {
        state.remove_ref(rid);
    }
}

fn apply_update_ref(state: &mut BlockstoreState, op: &BlockOp) {
    let fwd = &op.forward;
    let Some(rid) = fwd.get("ref_id").and_then(Value::as_i64) else { return };
    let mut patch = json!({ "updated_at": op.applied_at });
    if let Value::Object(ref mut m) = patch {
        if let Some(v) = fwd.get("from") { m.insert("from_id".into(), v.clone()); }
        for k in ["order_key", "anchor", "meta"] {
            if let Some(v) = fwd.get(k) { m.insert(k.into(), v.clone()); }
        }
    }
    state.update_ref_fields(rid, &patch);
}
