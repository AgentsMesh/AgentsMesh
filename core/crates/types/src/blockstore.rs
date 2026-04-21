use serde::{Deserialize, Serialize};
use serde_json::Value;

pub type JsonMap = serde_json::Map<String, Value>;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum ActorType {
    User,
    Agent,
    System,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "camelCase")]
pub enum OpKind {
    CreateBlock,
    UpdateBlock,
    DeleteBlock,
    AddRef,
    RemoveRef,
    UpdateRef,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    pub id: String,
    pub workspace_id: String,
    #[serde(rename = "type")]
    pub block_type: String,
    #[serde(default)]
    pub data: Value,
    #[serde(default)]
    pub text: Option<String>,
    #[serde(default)]
    pub meta: Value,
    #[serde(default)]
    pub created_by: i64,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
    #[serde(default)]
    pub deleted_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockRef {
    pub id: i64,
    pub workspace_id: String,
    pub from_id: String,
    pub to_id: String,
    pub rel: String,
    #[serde(default)]
    pub order_key: Option<String>,
    #[serde(default)]
    pub anchor: Option<String>,
    #[serde(default)]
    pub meta: Value,
    #[serde(default)]
    pub created_by: i64,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockOp {
    pub id: i64,
    pub workspace_id: String,
    #[serde(default)]
    pub idempotency_key: Option<String>,
    pub actor_type: ActorType,
    pub actor_id: i64,
    pub op: OpKind,
    #[serde(default)]
    pub target_block: Option<String>,
    #[serde(default)]
    pub target_ref: Option<i64>,
    #[serde(default)]
    pub payload: Value,
    #[serde(default)]
    pub forward: Value,
    #[serde(default)]
    pub inverse: Value,
    #[serde(default)]
    pub parent_op_id: Option<i64>,
    #[serde(default)]
    pub applied_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Workspace {
    pub id: String,
    pub organization_id: i64,
    pub slug: String,
    pub name: String,
    #[serde(default)]
    pub root_block_id: Option<String>,
    #[serde(default)]
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OpEnvelope {
    pub op: OpKind,
    #[serde(default)]
    pub payload: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApplyOpsRequest {
    pub workspace_id: String,
    pub ops: Vec<OpEnvelope>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub idempotency_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_op_id: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApplyOpsResult {
    pub op_ids: Vec<i64>,
    pub was_replay: bool,
    #[serde(default)]
    pub parent_op_id: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ChildrenResult {
    #[serde(default)]
    pub blocks: Vec<Block>,
    #[serde(default)]
    pub refs: Vec<BlockRef>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchHit {
    pub block_id: String,
    #[serde(rename = "type")]
    pub block_type: String,
    pub snippet: String,
    pub score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SemanticSearchRequest {
    pub query: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub top_k: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub min_score: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none", rename = "type")]
    pub block_type: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn block_roundtrip() {
        let raw = r#"{
            "id": "b1",
            "workspace_id": "ws1",
            "type": "page",
            "data": {"title": "hi"},
            "meta": {},
            "created_by": 1,
            "created_at": "2026-04-19T00:00:00Z",
            "updated_at": "2026-04-19T00:00:00Z"
        }"#;
        let b: Block = serde_json::from_str(raw).unwrap();
        assert_eq!(b.id, "b1");
        assert_eq!(b.block_type, "page");
    }

    #[test]
    fn op_kind_camel_case() {
        let k: OpKind = serde_json::from_str("\"createBlock\"").unwrap();
        assert_eq!(k, OpKind::CreateBlock);
        let s = serde_json::to_string(&OpKind::AddRef).unwrap();
        assert_eq!(s, "\"addRef\"");
    }

    #[test]
    fn apply_ops_request_skips_empty() {
        let req = ApplyOpsRequest {
            workspace_id: "ws".into(),
            ops: vec![],
            idempotency_key: None,
            parent_op_id: None,
        };
        let s = serde_json::to_string(&req).unwrap();
        assert!(!s.contains("idempotency_key"));
        assert!(!s.contains("parent_op_id"));
    }
}
