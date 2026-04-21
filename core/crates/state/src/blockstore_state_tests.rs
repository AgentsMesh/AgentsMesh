use agentsmesh_types::{Block, BlockOp, BlockRef, OpKind, ActorType, Workspace};
use serde_json::json;

use crate::blockstore_state::BlockstoreState;

fn mk_block(id: &str, ws: &str, ty: &str) -> Block {
    Block {
        id: id.into(), workspace_id: ws.into(), block_type: ty.into(),
        data: json!({}), text: None, meta: json!({}),
        created_by: 1, created_at: "2026-04-19T00:00:00Z".into(),
        updated_at: "2026-04-19T00:00:00Z".into(), deleted_at: None,
    }
}

fn mk_ref(id: i64, from: &str, to: &str, rel: &str, order: Option<&str>) -> BlockRef {
    BlockRef {
        id, workspace_id: "ws".into(),
        from_id: from.into(), to_id: to.into(), rel: rel.into(),
        order_key: order.map(str::to_string), anchor: None, meta: json!({}),
        created_by: 1, created_at: "t".into(), updated_at: "t".into(),
    }
}

fn mk_op(id: i64, kind: OpKind, forward: serde_json::Value) -> BlockOp {
    BlockOp {
        id, workspace_id: "ws".into(), idempotency_key: None,
        actor_type: ActorType::User, actor_id: 1, op: kind,
        target_block: None, target_ref: None,
        payload: json!({}), forward, inverse: json!({}),
        parent_op_id: None, applied_at: "2026-04-19T00:00:00Z".into(),
    }
}

#[test]
fn upsert_block_stores_in_map() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("b1", "ws", "page"));
    assert!(s.blocks.contains_key("b1"));
}

#[test]
fn nest_ref_indexes_and_sorts_by_order_key() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("p", "ws", "page"));
    s.upsert_block(mk_block("c1", "ws", "paragraph"));
    s.upsert_block(mk_block("c2", "ws", "paragraph"));
    s.upsert_ref(mk_ref(2, "p", "c2", "nest", Some("m")));
    s.upsert_ref(mk_ref(1, "p", "c1", "nest", Some("a")));
    let children = s.nest_children.get("p").unwrap();
    assert_eq!(children, &vec![1, 2]);
}

#[test]
fn non_nest_ref_goes_to_backlinks() {
    let mut s = BlockstoreState::new();
    s.upsert_ref(mk_ref(10, "src", "dst", "mention", None));
    assert_eq!(s.backlinks.get("dst").unwrap(), &vec![10]);
    assert!(s.nest_children.get("src").is_none());
}

#[test]
fn remove_block_cascades_refs() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("a", "ws", "page"));
    s.upsert_block(mk_block("b", "ws", "page"));
    s.upsert_ref(mk_ref(1, "a", "b", "nest", Some("x")));
    s.remove_block("b");
    assert!(!s.refs.contains_key(&1));
    assert!(s.nest_children.get("a").is_none());
}

#[test]
fn apply_create_block_op() {
    let mut s = BlockstoreState::new();
    let op = mk_op(1, OpKind::CreateBlock, json!({
        "id": "b1", "type": "page", "data": {"title": "hi"},
    }));
    s.apply_remote_op(&op);
    assert_eq!(s.blocks.get("b1").unwrap().block_type, "page");
    assert_eq!(s.get_last_op_id("ws"), 1);
}

#[test]
fn apply_add_ref_op() {
    let mut s = BlockstoreState::new();
    let op = mk_op(5, OpKind::AddRef, json!({
        "id": 7, "from": "p", "to": "c", "rel": "nest", "order_key": "m",
    }));
    s.apply_remote_op(&op);
    assert!(s.refs.contains_key(&7));
    assert_eq!(s.nest_children.get("p").unwrap(), &vec![7]);
}

#[test]
fn apply_delete_block_removes_it() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("b1", "ws", "page"));
    let op = mk_op(2, OpKind::DeleteBlock, json!({"id": "b1"}));
    s.apply_remote_op(&op);
    assert!(!s.blocks.contains_key("b1"));
}

#[test]
fn workspaces_roundtrip_json() {
    let mut s = BlockstoreState::new();
    s.upsert_workspace(Workspace {
        id: "ws1".into(), organization_id: 1, slug: "main".into(),
        name: "Main".into(), root_block_id: None, created_at: "t".into(),
    });
    let json = s.workspaces_json();
    assert!(json.contains("ws1"));
}

#[test]
fn list_children_returns_blocks_in_order() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("p", "ws", "page"));
    s.upsert_block(mk_block("c1", "ws", "paragraph"));
    s.upsert_block(mk_block("c2", "ws", "paragraph"));
    s.upsert_ref(mk_ref(2, "p", "c2", "nest", Some("m")));
    s.upsert_ref(mk_ref(1, "p", "c1", "nest", Some("a")));
    let json = s.list_children_json("p");
    let parsed: serde_json::Value = serde_json::from_str(&json).unwrap();
    let blocks = parsed.get("blocks").unwrap().as_array().unwrap();
    assert_eq!(blocks[0].get("id").unwrap(), "c1");
    assert_eq!(blocks[1].get("id").unwrap(), "c2");
}

#[test]
fn type_defs_json_filters_by_workspace_and_type() {
    let mut s = BlockstoreState::new();
    s.upsert_block(mk_block("t1", "ws", "block_type_def"));
    s.upsert_block(mk_block("t2", "ws2", "block_type_def"));
    s.upsert_block(mk_block("p", "ws", "page"));
    let json = s.type_defs_json("ws");
    assert!(json.contains("t1"));
    assert!(!json.contains("t2"));
    assert!(!json.contains("\"id\":\"p\""));
}
