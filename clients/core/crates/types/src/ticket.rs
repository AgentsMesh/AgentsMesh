use serde::{Deserialize, Serialize};

use crate::{TicketPriority, TicketStatus};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Ticket {
    // Backend's primary key. Required by buildTicketContext (clients/web/src/components/tickets/buildTicketContext.ts:11),
    // which short-circuits to undefined when falsy — breaking the ticket→spawn-pod flow.
    // `default` keeps minimal-JSON tests deserializing; prod backend always sends a non-zero id.
    #[serde(default)]
    pub id: i64,
    pub slug: String,
    pub title: String,
    pub content: Option<String>,
    pub status: TicketStatus,
    pub priority: TicketPriority,
    pub repository_id: Option<i64>,
    pub parent_slug: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BoardColumn {
    pub status: TicketStatus,
    pub tickets: Vec<Ticket>,
    #[serde(alias = "count", default)]
    pub total_count: i64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct BoardResponse {
    pub columns: Vec<BoardColumn>,
    #[serde(default)]
    pub priority_counts: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Label {
    pub id: i64,
    pub name: String,
    pub color: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketComment {
    pub id: i64,
    pub ticket_slug: Option<String>,
    pub content: String,
    pub parent_id: Option<i64>,
    pub author: Option<crate::User>,
    pub mentions: Option<Vec<String>>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketRelation {
    pub id: i64,
    pub source_slug: Option<String>,
    pub target_slug: Option<String>,
    pub relation_type: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketCommit {
    pub id: i64,
    pub ticket_slug: Option<String>,
    pub commit_sha: String,
    pub commit_message: Option<String>,
    pub commit_url: Option<String>,
    pub author_name: Option<String>,
    pub author_email: Option<String>,
    pub committed_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketListResponse {
    pub tickets: Vec<Ticket>,
    pub total: Option<i64>,
    #[serde(default)]
    pub limit: Option<i64>,
    #[serde(default)]
    pub offset: Option<i64>,
}



#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabelListResponse {
    pub labels: Vec<Label>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json;

    #[test]
    fn ticket_roundtrip() {
        let ticket = Ticket {
            id: 1,
            slug: "TICKET-1".into(),
            title: "Fix login bug".into(),
            content: Some("Users can't login".into()),
            status: TicketStatus::Todo,
            priority: TicketPriority::High,
            repository_id: Some(1),
            parent_slug: None,
            created_at: Some("2026-01-01T00:00:00Z".into()),
            updated_at: None,
        };
        let json = serde_json::to_string(&ticket).unwrap();
        let decoded: Ticket = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.slug, "TICKET-1");
        assert_eq!(decoded.priority, TicketPriority::High);
        assert_eq!(decoded.status, TicketStatus::Todo);
    }

    #[test]
    fn ticket_minimal_json() {
        let json = r#"{"slug":"T-1","title":"t","status":"backlog","priority":"low"}"#;
        let ticket: Ticket = serde_json::from_str(json).unwrap();
        assert_eq!(ticket.slug, "T-1");
        assert_eq!(ticket.status, TicketStatus::Backlog);
        assert_eq!(ticket.priority, TicketPriority::Low);
        assert!(ticket.content.is_none());
        assert!(ticket.parent_slug.is_none());
    }

    #[test]
    fn board_column_roundtrip() {
        let col = BoardColumn {
            status: TicketStatus::InProgress,
            tickets: vec![Ticket {
                id: 1,
                slug: "T-1".into(),
                title: "task".into(),
                content: None,
                status: TicketStatus::InProgress,
                priority: TicketPriority::Medium,
                repository_id: None,
                parent_slug: None,
                created_at: None,
                updated_at: None,
            }],
            total_count: 1,
        };
        let json = serde_json::to_string(&col).unwrap();
        let decoded: BoardColumn = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.tickets.len(), 1);
        assert_eq!(decoded.total_count, 1);
    }

    #[test]
    fn board_column_empty_tickets() {
        let json = r#"{"status":"done","tickets":[],"total_count":0}"#;
        let col: BoardColumn = serde_json::from_str(json).unwrap();
        assert!(col.tickets.is_empty());
        assert_eq!(col.status, TicketStatus::Done);
    }

    #[test]
    fn label_roundtrip() {
        let label = Label {
            id: 1,
            name: "bug".into(),
            color: "#ff0000".into(),
        };
        let json = serde_json::to_string(&label).unwrap();
        let decoded: Label = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.name, "bug");
        assert_eq!(decoded.color, "#ff0000");
    }

    #[test]
    fn ticket_list_relay_preserves_pagination() {
        let backend = r#"{
            "tickets": [{"slug":"T-1","title":"t","status":"backlog","priority":"low"}],
            "total": 42, "limit": 20, "offset": 20
        }"#;
        let typed: TicketListResponse = serde_json::from_str(backend).unwrap();
        let relayed = serde_json::to_string(&typed).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
        assert_eq!(parsed["total"], serde_json::json!(42));
        assert_eq!(parsed["limit"], serde_json::json!(20));
        assert_eq!(parsed["offset"], serde_json::json!(20));
    }

    // Regression: id must survive the backend → Rust → JS round-trip. Without
    // `pub id` on Ticket, serde silently drops the field during from_str,
    // and current_ticket_json() then emits a ticket without id. JS-side
    // buildTicketContext returns undefined when !ticket.id, which collapses
    // the spawn-pod-from-ticket flow into the workspace scenario (no prompt
    // seeded, no ticket_slug sent). See TICKET-146.
    #[test]
    fn ticket_id_survives_backend_roundtrip() {
        let backend_envelope = r#"{
            "id": 123,
            "organization_id": 5,
            "number": 42,
            "slug": "AM-42",
            "title": "x",
            "content": "body",
            "status": "todo",
            "priority": "high",
            "repository_id": 7,
            "reporter_id": 1,
            "created_at": "2026-05-16T00:00:00Z",
            "updated_at": "2026-05-16T00:00:00Z"
        }"#;
        let t: Ticket = serde_json::from_str(backend_envelope).unwrap();
        assert_eq!(t.id, 123);
        let relayed = serde_json::to_string(&t).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
        assert_eq!(parsed["id"], serde_json::json!(123));
    }
}
