// Hand-maintained `prost::Message` mirrors of
// `proto/ticket/v1/ticket.proto`. Tag numbers match the .proto
// byte-for-byte; `tools/validate_prost_tags` runs at build time to catch
// drift (watch list §8). NO `Serialize`/`Deserialize` derives on these
// structs — binary wire only (conventions §2.5, §3).
//
// Historical drift: 986a38ca6 — the REST envelope was `{tickets, total,
// limit, offset}` while skill-registries' was `{items, total, limit,
// offset}`. The proto SSOT unifies to `{items, total, limit, offset}`
// across all 26 services per conventions §8.

#[derive(Clone, PartialEq, prost::Message)]
pub struct Ticket {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int32, tag = "2")]
    pub number: i32,
    #[prost(string, tag = "3")]
    pub slug: String,
    #[prost(string, tag = "4")]
    pub title: String,
    #[prost(string, optional, tag = "5")]
    pub content: Option<String>,
    #[prost(string, tag = "6")]
    pub status: String,
    #[prost(string, tag = "7")]
    pub priority: String,
    #[prost(string, optional, tag = "8")]
    pub severity: Option<String>,
    #[prost(double, optional, tag = "9")]
    pub estimate: Option<f64>,
    #[prost(string, optional, tag = "10")]
    pub due_date: Option<String>,
    #[prost(string, optional, tag = "11")]
    pub started_at: Option<String>,
    #[prost(string, optional, tag = "12")]
    pub completed_at: Option<String>,
    #[prost(int64, optional, tag = "13")]
    pub repository_id: Option<i64>,
    #[prost(int64, optional, tag = "14")]
    pub parent_ticket_id: Option<i64>,
    #[prost(string, optional, tag = "15")]
    pub parent_ticket_slug: Option<String>,
    #[prost(int64, optional, tag = "16")]
    pub reporter_id: Option<i64>,
    #[prost(string, tag = "17")]
    pub created_at: String,
    #[prost(string, tag = "18")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Label {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, optional, tag = "3")]
    pub repository_id: Option<i64>,
    #[prost(string, tag = "4")]
    pub name: String,
    #[prost(string, tag = "5")]
    pub color: String,
    #[prost(string, optional, tag = "6")]
    pub created_at: Option<String>,
    #[prost(string, optional, tag = "7")]
    pub updated_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Board {
    #[prost(message, repeated, tag = "1")]
    pub columns: Vec<BoardColumn>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct BoardColumn {
    #[prost(string, tag = "1")]
    pub status: String,
    #[prost(message, repeated, tag = "2")]
    pub tickets: Vec<Ticket>,
    #[prost(int64, tag = "3")]
    pub total_count: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListTicketsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, optional, tag = "2")]
    pub repository_id: Option<i64>,
    #[prost(string, optional, tag = "3")]
    pub status: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub priority: Option<String>,
    #[prost(int64, optional, tag = "5")]
    pub assignee_id: Option<i64>,
    #[prost(string, repeated, tag = "6")]
    pub labels: Vec<String>,
    #[prost(string, optional, tag = "7")]
    pub query: Option<String>,
    #[prost(int32, optional, tag = "8")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "9")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListTicketsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Ticket>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetTicketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateTicketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub title: String,
    #[prost(string, optional, tag = "3")]
    pub content: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub status: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub priority: Option<String>,
    #[prost(int64, optional, tag = "6")]
    pub repository_id: Option<i64>,
    #[prost(int64, repeated, tag = "7")]
    pub assignee_ids: Vec<i64>,
    #[prost(string, repeated, tag = "8")]
    pub labels: Vec<String>,
    #[prost(string, optional, tag = "9")]
    pub parent_ticket_slug: Option<String>,
    #[prost(string, optional, tag = "10")]
    pub due_date: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateTicketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(string, optional, tag = "3")]
    pub title: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub content: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub status: Option<String>,
    #[prost(string, optional, tag = "6")]
    pub priority: Option<String>,
    #[prost(int64, optional, tag = "7")]
    pub repository_id: Option<i64>,
    #[prost(int64, repeated, tag = "8")]
    pub assignee_ids: Vec<i64>,
    #[prost(string, repeated, tag = "9")]
    pub labels: Vec<String>,
    #[prost(string, optional, tag = "10")]
    pub due_date: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteTicketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteTicketResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateTicketStatusRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(string, tag = "3")]
    pub status: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateTicketStatusResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetActiveTicketsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, optional, tag = "2")]
    pub repository_id: Option<i64>,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetBoardRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, optional, tag = "2")]
    pub repository_id: Option<i64>,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
    #[prost(string, optional, tag = "4")]
    pub priority: Option<String>,
    #[prost(int64, optional, tag = "5")]
    pub assignee_id: Option<i64>,
    #[prost(string, optional, tag = "6")]
    pub query: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetSubTicketsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AddAssigneeRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub user_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AddAssigneeResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveAssigneeRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub user_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveAssigneeResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListLabelsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, optional, tag = "2")]
    pub repository_id: Option<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListLabelsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Label>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateLabelRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(string, tag = "3")]
    pub color: String,
    #[prost(int64, optional, tag = "4")]
    pub repository_id: Option<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateLabelRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub id: i64,
    #[prost(string, optional, tag = "3")]
    pub name: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub color: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteLabelRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteLabelResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AddLabelRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub label_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AddLabelResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveLabelRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub label_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveLabelResponse {}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_ticket() -> Ticket {
        Ticket {
            id: 42,
            number: 7,
            slug: "ACME-7".into(),
            title: "Fix login bug".into(),
            content: Some("Users can't login after 2FA".into()),
            status: "in_progress".into(),
            priority: "high".into(),
            severity: Some("major".into()),
            estimate: Some(3.5),
            due_date: Some("2026-05-20".into()),
            started_at: Some("2026-05-10T00:00:00Z".into()),
            completed_at: None,
            repository_id: Some(11),
            parent_ticket_id: Some(1),
            parent_ticket_slug: Some("ACME-1".into()),
            reporter_id: Some(99),
            created_at: "2026-05-08T00:00:00Z".into(),
            updated_at: "2026-05-10T00:00:00Z".into(),
        }
    }

    #[test]
    fn ticket_preserves_every_backend_field() {
        let original = sample_ticket();
        let bytes = original.encode_to_vec();
        let decoded = Ticket::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.severity.as_deref(), Some("major"));
        assert_eq!(decoded.estimate, Some(3.5));
        assert_eq!(decoded.parent_ticket_slug.as_deref(), Some("ACME-1"));
    }

    #[test]
    fn list_response_envelope_round_trip() {
        let original = ListTicketsResponse {
            items: vec![sample_ticket()],
            total: 42,
            limit: 20,
            offset: 20,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListTicketsResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.total, 42);
        assert_eq!(decoded.limit, 20);
        assert_eq!(decoded.offset, 20);
    }

    #[test]
    fn create_request_round_trip_with_assignees_and_labels() {
        let original = CreateTicketRequest {
            org_slug: "acme".into(),
            title: "feat: x".into(),
            content: Some("plan".into()),
            status: Some("backlog".into()),
            priority: Some("medium".into()),
            repository_id: Some(11),
            assignee_ids: vec![1, 2, 3],
            labels: vec!["bug".into(), "ui".into()],
            parent_ticket_slug: Some("ACME-1".into()),
            due_date: Some("2026-06-01".into()),
        };
        let bytes = original.encode_to_vec();
        let decoded = CreateTicketRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.assignee_ids, vec![1, 2, 3]);
        assert_eq!(decoded.labels, vec!["bug", "ui"]);
    }

    #[test]
    fn update_request_optionals_distinguishable() {
        // PATCH semantics: omitted vs explicit field must encode differently
        // so handler can tell "leave untouched" apart from "set to zero".
        let omit = UpdateTicketRequest {
            org_slug: "acme".into(),
            ticket_slug: "ACME-7".into(),
            title: Some("new".into()),
            content: None,
            status: None,
            priority: None,
            repository_id: None,
            assignee_ids: vec![],
            labels: vec![],
            due_date: None,
        };
        let clear_repo = UpdateTicketRequest {
            org_slug: "acme".into(),
            ticket_slug: "ACME-7".into(),
            title: Some("new".into()),
            content: None,
            status: None,
            priority: None,
            // 0 explicitly clears the repository association
            // (tickets.go:209 inlines this semantic).
            repository_id: Some(0),
            assignee_ids: vec![],
            labels: vec![],
            due_date: None,
        };
        assert_ne!(
            omit.encode_to_vec(),
            clear_repo.encode_to_vec(),
            "omitted vs explicit 0 must encode differently"
        );
    }

    #[test]
    fn board_round_trip() {
        let original = Board {
            columns: vec![
                BoardColumn {
                    status: "todo".into(),
                    tickets: vec![sample_ticket()],
                    total_count: 1,
                },
                BoardColumn {
                    status: "done".into(),
                    tickets: vec![],
                    total_count: 0,
                },
            ],
        };
        let bytes = original.encode_to_vec();
        let decoded = Board::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.columns.len(), 2);
    }

    #[test]
    fn label_round_trip() {
        let original = Label {
            id: 1,
            organization_id: 7,
            repository_id: Some(11),
            name: "bug".into(),
            color: "#ff0000".into(),
            created_at: Some("2026-05-08T00:00:00Z".into()),
            updated_at: None,
        };
        let bytes = original.encode_to_vec();
        let decoded = Label::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn optional_offset_zero_distinguishable_from_absent() {
        // Conventions §5: explicit offset=0 vs absent must encode differently.
        let with_zero = ListTicketsRequest {
            org_slug: "acme".into(),
            repository_id: None,
            status: None,
            priority: None,
            assignee_id: None,
            labels: vec![],
            query: None,
            offset: Some(0),
            limit: None,
        };
        let absent = ListTicketsRequest {
            org_slug: "acme".into(),
            repository_id: None,
            status: None,
            priority: None,
            assignee_id: None,
            labels: vec![],
            query: None,
            offset: None,
            limit: None,
        };
        assert_ne!(
            with_zero.encode_to_vec(),
            absent.encode_to_vec(),
            "explicit zero must encode different bytes from absent field"
        );
    }

    #[test]
    fn delete_request_response_round_trip() {
        let req = DeleteTicketRequest {
            org_slug: "acme".into(),
            ticket_slug: "ACME-7".into(),
        };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, DeleteTicketRequest::decode(&*req_bytes).unwrap());

        let resp = DeleteTicketResponse {};
        let resp_bytes = resp.encode_to_vec();
        assert!(resp_bytes.is_empty(), "empty message encodes to zero bytes");
        assert_eq!(resp, DeleteTicketResponse::decode(&*resp_bytes).unwrap());
    }
}
