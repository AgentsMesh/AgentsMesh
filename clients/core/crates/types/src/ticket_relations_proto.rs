// Hand-maintained `prost::Message` mirrors of
// `proto/ticket_relations/v1/ticket_relations.proto`. Tag numbers match the
// .proto byte-for-byte; `tools/validate_prost_tags` runs at build time to
// catch drift (watch list §8). NO `Serialize`/`Deserialize` derives — binary
// wire only (conventions §2.5, §3).
//
// PR 986a38ca6 reconciliation: comment list envelope `{items, total, limit,
// offset}` survives the wire — we use the proto3 list shape here, and the
// adapter maps it to the legacy `{comments, ...}` shape for the UI.

// ============================================================
// Entities
// ============================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct RelatedTicketSummary {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub slug: String,
    #[prost(string, tag = "3")]
    pub title: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Relation {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, tag = "3")]
    pub source_ticket_id: i64,
    #[prost(int64, tag = "4")]
    pub target_ticket_id: i64,
    #[prost(string, tag = "5")]
    pub relation_type: String,
    #[prost(string, tag = "6")]
    pub created_at: String,
    #[prost(message, optional, tag = "7")]
    pub source_ticket: Option<RelatedTicketSummary>,
    #[prost(message, optional, tag = "8")]
    pub target_ticket: Option<RelatedTicketSummary>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MergeRequest {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int32, tag = "2")]
    pub mr_iid: i32,
    #[prost(string, tag = "3")]
    pub title: String,
    #[prost(string, tag = "4")]
    pub state: String,
    #[prost(string, tag = "5")]
    pub mr_url: String,
    #[prost(string, tag = "6")]
    pub source_branch: String,
    #[prost(string, tag = "7")]
    pub target_branch: String,
    #[prost(string, optional, tag = "8")]
    pub pipeline_status: Option<String>,
    #[prost(int64, optional, tag = "9")]
    pub pipeline_id: Option<i64>,
    #[prost(string, optional, tag = "10")]
    pub pipeline_url: Option<String>,
    #[prost(int64, optional, tag = "11")]
    pub ticket_id: Option<i64>,
    #[prost(int64, optional, tag = "12")]
    pub pod_id: Option<i64>,
    #[prost(string, optional, tag = "13")]
    pub merge_commit_sha: Option<String>,
    #[prost(string, optional, tag = "14")]
    pub merged_at: Option<String>,
    #[prost(int64, optional, tag = "15")]
    pub merged_by_id: Option<i64>,
    #[prost(string, tag = "16")]
    pub created_at: String,
    #[prost(string, tag = "17")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Commit {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, tag = "3")]
    pub ticket_id: i64,
    #[prost(int64, tag = "4")]
    pub repository_id: i64,
    #[prost(int64, optional, tag = "5")]
    pub pod_id: Option<i64>,
    #[prost(string, tag = "6")]
    pub commit_sha: String,
    #[prost(string, tag = "7")]
    pub commit_message: String,
    #[prost(string, optional, tag = "8")]
    pub commit_url: Option<String>,
    #[prost(string, optional, tag = "9")]
    pub author_name: Option<String>,
    #[prost(string, optional, tag = "10")]
    pub author_email: Option<String>,
    #[prost(string, optional, tag = "11")]
    pub committed_at: Option<String>,
    #[prost(string, tag = "12")]
    pub created_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CommentMention {
    #[prost(int64, tag = "1")]
    pub user_id: i64,
    #[prost(string, tag = "2")]
    pub username: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CommentUser {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub username: String,
    #[prost(string, optional, tag = "3")]
    pub name: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub avatar_url: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Comment {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub ticket_id: i64,
    #[prost(int64, tag = "3")]
    pub user_id: i64,
    #[prost(string, tag = "4")]
    pub content: String,
    #[prost(int64, optional, tag = "5")]
    pub parent_id: Option<i64>,
    #[prost(message, repeated, tag = "6")]
    pub mentions: Vec<CommentMention>,
    #[prost(string, tag = "7")]
    pub created_at: String,
    #[prost(string, tag = "8")]
    pub updated_at: String,
    #[prost(message, optional, tag = "9")]
    pub user: Option<CommentUser>,
    #[prost(message, repeated, tag = "10")]
    pub replies: Vec<Comment>,
}

// ============================================================
// Relations RPCs
// ============================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRelationsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRelationsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Relation>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateRelationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(string, tag = "3")]
    pub target_slug: String,
    #[prost(string, tag = "4")]
    pub relation_type: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteRelationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub relation_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteRelationResponse {}

// ============================================================
// Merge requests RPCs
// ============================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMergeRequestsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMergeRequestsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<MergeRequest>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

// ============================================================
// Commits RPCs
// ============================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListCommitsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListCommitsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Commit>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LinkCommitRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(string, tag = "3")]
    pub commit_sha: String,
    #[prost(string, optional, tag = "4")]
    pub commit_message: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub commit_url: Option<String>,
    #[prost(string, optional, tag = "6")]
    pub author_name: Option<String>,
    #[prost(string, optional, tag = "7")]
    pub author_email: Option<String>,
    #[prost(string, optional, tag = "8")]
    pub committed_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UnlinkCommitRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub commit_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UnlinkCommitResponse {}

// ============================================================
// Comments RPCs
// ============================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListCommentsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
    #[prost(int32, optional, tag = "4")]
    pub offset: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListCommentsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Comment>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateCommentRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(string, tag = "3")]
    pub content: String,
    #[prost(int64, optional, tag = "4")]
    pub parent_id: Option<i64>,
    #[prost(message, repeated, tag = "5")]
    pub mentions: Vec<CommentMention>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateCommentRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub comment_id: i64,
    #[prost(string, tag = "4")]
    pub content: String,
    #[prost(message, repeated, tag = "5")]
    pub mentions: Vec<CommentMention>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteCommentRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub comment_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteCommentResponse {}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn relation_round_trip() {
        let original = Relation {
            id: 7,
            organization_id: 42,
            source_ticket_id: 100,
            target_ticket_id: 200,
            relation_type: "blocks".into(),
            created_at: "2026-05-12T00:00:00Z".into(),
            source_ticket: None,
            target_ticket: None,
        };
        let bytes = original.encode_to_vec();
        let decoded = Relation::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn comment_with_mentions_round_trip() {
        let original = Comment {
            id: 1,
            ticket_id: 100,
            user_id: 50,
            content: "Hey @alice see this".into(),
            parent_id: Some(99),
            mentions: vec![CommentMention { user_id: 51, username: "alice".into() }],
            created_at: "2026-05-12T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
            user: Some(CommentUser {
                id: 50,
                username: "bob".into(),
                name: Some("Bob Smith".into()),
                avatar_url: None,
            }),
            replies: vec![],
        };
        let bytes = original.encode_to_vec();
        let decoded = Comment::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.mentions.len(), 1);
        assert_eq!(decoded.user.as_ref().unwrap().username, "bob");
    }

    #[test]
    fn comment_with_nested_replies_round_trip() {
        // Backend ListComments preloads top-level comments with their Replies
        // populated; tag 10 must round-trip recursively (watch list §6).
        let reply = Comment {
            id: 2,
            ticket_id: 100,
            user_id: 51,
            content: "reply".into(),
            parent_id: Some(1),
            mentions: vec![],
            created_at: "2026-05-12T01:00:00Z".into(),
            updated_at: "2026-05-12T01:00:00Z".into(),
            user: None,
            replies: vec![],
        };
        let original = Comment {
            id: 1,
            ticket_id: 100,
            user_id: 50,
            content: "top".into(),
            parent_id: None,
            mentions: vec![],
            created_at: "2026-05-12T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
            user: None,
            replies: vec![reply],
        };
        let bytes = original.encode_to_vec();
        let decoded = Comment::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.replies.len(), 1);
        assert_eq!(decoded.replies[0].id, 2);
    }

    #[test]
    fn relation_with_summary_round_trip() {
        let original = Relation {
            id: 7,
            organization_id: 42,
            source_ticket_id: 100,
            target_ticket_id: 200,
            relation_type: "blocks".into(),
            created_at: "2026-05-12T00:00:00Z".into(),
            source_ticket: Some(RelatedTicketSummary {
                id: 100,
                slug: "T-100".into(),
                title: "source".into(),
            }),
            target_ticket: Some(RelatedTicketSummary {
                id: 200,
                slug: "T-200".into(),
                title: "target".into(),
            }),
        };
        let bytes = original.encode_to_vec();
        let decoded = Relation::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.source_ticket.as_ref().unwrap().slug, "T-100");
    }

    #[test]
    fn list_comments_envelope_round_trip() {
        // PR 986a38ca6 lineage: pagination MUST survive. Backend emitted
        // {comments, total, limit, offset}; we wire as {items, total, limit,
        // offset} per the uniform list envelope (conventions §8), the adapter
        // maps it back to the legacy key for the UI.
        let original = ListCommentsResponse {
            items: vec![Comment {
                id: 1,
                ticket_id: 100,
                user_id: 50,
                content: "hi".into(),
                parent_id: None,
                mentions: vec![],
                created_at: "2026-05-12T00:00:00Z".into(),
                updated_at: "2026-05-12T00:00:00Z".into(),
                user: None,
                replies: vec![],
            }],
            total: 13,
            limit: 50,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListCommentsResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.total, 13);
        assert_eq!(decoded.limit, 50);
    }

    #[test]
    fn commit_optional_fields_round_trip() {
        let original = Commit {
            id: 1,
            organization_id: 42,
            ticket_id: 100,
            repository_id: 7,
            pod_id: Some(99),
            commit_sha: "abc123".into(),
            commit_message: "fix bug".into(),
            commit_url: Some("https://github.com/acme/x/commit/abc".into()),
            author_name: Some("Alice".into()),
            author_email: Some("alice@example.com".into()),
            committed_at: Some("2026-05-12T00:00:00Z".into()),
            created_at: "2026-05-12T00:00:00Z".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = Commit::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn list_comments_offset_zero_distinguishable_from_absent() {
        // Conventions §5: `optional int32 offset` must distinguish absent
        // from explicit 0.
        let with_zero = ListCommentsRequest {
            org_slug: "acme".into(),
            ticket_slug: "T-1".into(),
            limit: Some(50),
            offset: Some(0),
        };
        let absent = ListCommentsRequest {
            org_slug: "acme".into(),
            ticket_slug: "T-1".into(),
            limit: Some(50),
            offset: None,
        };
        assert_ne!(
            with_zero.encode_to_vec(),
            absent.encode_to_vec(),
            "explicit zero must encode different bytes from absent",
        );
    }

    #[test]
    fn delete_response_empty_encoding() {
        let r = DeleteRelationResponse {};
        assert!(r.encode_to_vec().is_empty());
        let u = UnlinkCommitResponse {};
        assert!(u.encode_to_vec().is_empty());
        let c = DeleteCommentResponse {};
        assert!(c.encode_to_vec().is_empty());
    }
}
