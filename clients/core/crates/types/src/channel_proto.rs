// Hand-maintained `prost::Message` mirrors of
// `proto/channel/v1/channel.proto`. Tag numbers match the .proto
// byte-for-byte; `tools/validate_prost_tags` runs at build time to catch
// drift (watch list §8). NO `Serialize`/`Deserialize` derives on these
// structs — binary wire only (conventions §2.5, §3).
//
// Streaming endpoints (channel events) stay on Relay/WebSocket — unary RPC
// only here (runbook §"Service-specific deviations").
//
// content / mentions ride as opaque JSON strings (`content_json`,
// `mentions_json`) so the rich Block AST stays owned by the legacy serde
// DTO. A follow-up structured-content migration can promote them.

use std::collections::HashMap;

// =============================================================================
// Entities
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct Channel {
    #[prost(int64, tag = "1")] pub id: i64,
    #[prost(int64, tag = "2")] pub organization_id: i64,
    #[prost(string, tag = "3")] pub name: String,
    #[prost(string, optional, tag = "4")] pub description: Option<String>,
    #[prost(string, optional, tag = "5")] pub document: Option<String>,
    #[prost(int64, optional, tag = "6")] pub repository_id: Option<i64>,
    #[prost(int64, optional, tag = "7")] pub ticket_id: Option<i64>,
    #[prost(string, optional, tag = "8")] pub ticket_slug: Option<String>,
    #[prost(string, optional, tag = "9")] pub created_by_pod: Option<String>,
    #[prost(int64, optional, tag = "10")] pub created_by_user_id: Option<i64>,
    #[prost(string, tag = "11")] pub visibility: String,
    #[prost(bool, tag = "12")] pub is_archived: bool,
    #[prost(bool, tag = "13")] pub is_member: bool,
    #[prost(int64, tag = "14")] pub member_count: i64,
    #[prost(string, tag = "15")] pub created_at: String,
    #[prost(string, tag = "16")] pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelMessage {
    #[prost(int64, tag = "1")] pub id: i64,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(string, optional, tag = "3")] pub sender_pod: Option<String>,
    #[prost(int64, optional, tag = "4")] pub sender_user_id: Option<i64>,
    #[prost(string, tag = "5")] pub message_type: String,
    #[prost(string, tag = "6")] pub body: String,
    #[prost(string, optional, tag = "7")] pub content_json: Option<String>,
    #[prost(string, optional, tag = "8")] pub mentions_json: Option<String>,
    #[prost(int64, optional, tag = "9")] pub reply_to: Option<i64>,
    #[prost(string, optional, tag = "10")] pub edited_at: Option<String>,
    #[prost(bool, tag = "11")] pub is_deleted: bool,
    #[prost(string, tag = "12")] pub created_at: String,
    #[prost(message, optional, tag = "13")] pub sender_user: Option<ChannelMessageSenderUser>,
    #[prost(message, optional, tag = "14")] pub sender_pod_info: Option<ChannelMessageSenderPod>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelMessageSenderUser {
    #[prost(int64, tag = "1")] pub id: i64,
    #[prost(string, tag = "2")] pub username: String,
    #[prost(string, optional, tag = "3")] pub name: Option<String>,
    #[prost(string, optional, tag = "4")] pub avatar_url: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelMessageSenderPod {
    #[prost(string, tag = "1")] pub pod_key: String,
    #[prost(string, optional, tag = "2")] pub alias: Option<String>,
    #[prost(message, optional, tag = "3")] pub agent: Option<ChannelMessageSenderAgent>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelMessageSenderAgent {
    #[prost(string, tag = "1")] pub name: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelMember {
    #[prost(int64, tag = "1")] pub channel_id: i64,
    #[prost(int64, tag = "2")] pub user_id: i64,
    #[prost(string, tag = "3")] pub role: String,
    #[prost(bool, tag = "4")] pub is_muted: bool,
    #[prost(string, tag = "5")] pub joined_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelPod {
    #[prost(int64, tag = "1")] pub id: i64,
    #[prost(string, tag = "2")] pub pod_key: String,
    #[prost(string, optional, tag = "3")] pub alias: Option<String>,
    #[prost(string, tag = "4")] pub status: String,
    #[prost(string, tag = "5")] pub agent_status: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MentionRef {
    #[prost(string, tag = "1")] pub entity_type: String,
    #[prost(string, tag = "2")] pub entity_key: String,
}

// =============================================================================
// Channel requests / responses
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelsRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, optional, tag = "2")] pub repository_id: Option<i64>,
    #[prost(string, optional, tag = "3")] pub ticket_slug: Option<String>,
    #[prost(bool, optional, tag = "4")] pub include_archived: Option<bool>,
    #[prost(int32, optional, tag = "5")] pub offset: Option<i32>,
    #[prost(int32, optional, tag = "6")] pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelsResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<Channel>,
    #[prost(int64, tag = "2")] pub total: i64,
    #[prost(int32, tag = "3")] pub limit: i32,
    #[prost(int32, tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(string, tag = "2")] pub name: String,
    #[prost(string, optional, tag = "3")] pub description: Option<String>,
    #[prost(string, optional, tag = "4")] pub document: Option<String>,
    #[prost(int64, optional, tag = "5")] pub repository_id: Option<i64>,
    #[prost(string, optional, tag = "6")] pub ticket_slug: Option<String>,
    #[prost(string, optional, tag = "7")] pub visibility: Option<String>,
    #[prost(int64, repeated, tag = "8")] pub member_ids: Vec<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(string, optional, tag = "3")] pub name: Option<String>,
    #[prost(string, optional, tag = "4")] pub description: Option<String>,
    #[prost(string, optional, tag = "5")] pub document: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ArchiveChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ArchiveChannelResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UnarchiveChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UnarchiveChannelResponse {
    #[prost(string, tag = "1")] pub message: String,
}

// =============================================================================
// Document
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetChannelDocumentRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetChannelDocumentResponse {
    #[prost(string, tag = "1")] pub document: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateChannelDocumentRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(string, tag = "3")] pub document: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateChannelDocumentResponse {
    #[prost(string, tag = "1")] pub document: String,
}

// =============================================================================
// Messages
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelMessagesRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(int64, optional, tag = "3")] pub before_id: Option<i64>,
    #[prost(int32, optional, tag = "4")] pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelMessagesResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<ChannelMessage>,
    #[prost(int64, tag = "2")] pub total: i64,
    #[prost(int32, tag = "3")] pub limit: i32,
    #[prost(int32, tag = "4")] pub offset: i32,
    #[prost(bool, tag = "5")] pub has_more: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct SearchChannelMessagesRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(string, tag = "3")] pub query: String,
    #[prost(int32, optional, tag = "4")] pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct SearchChannelMessagesResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<ChannelMessage>,
    #[prost(int64, tag = "2")] pub total: i64,
    #[prost(int32, tag = "3")] pub limit: i32,
    #[prost(int32, tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct SendChannelMessageRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(string, optional, tag = "3")] pub source: Option<String>,
    #[prost(map = "string, message", tag = "4")] pub mentions: HashMap<String, MentionRef>,
    #[prost(string, optional, tag = "5")] pub content_json: Option<String>,
    #[prost(string, optional, tag = "6")] pub attachment_key: Option<String>,
    #[prost(string, optional, tag = "7")] pub pod_key: Option<String>,
    #[prost(int64, optional, tag = "8")] pub reply_to: Option<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct EditChannelMessageRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(int64, tag = "3")] pub message_id: i64,
    #[prost(string, optional, tag = "4")] pub source: Option<String>,
    #[prost(map = "string, message", tag = "5")] pub mentions: HashMap<String, MentionRef>,
    #[prost(string, optional, tag = "6")] pub content_json: Option<String>,
    #[prost(string, optional, tag = "7")] pub attachment_key: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteChannelMessageRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(int64, tag = "3")] pub message_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeleteChannelMessageResponse {
    #[prost(string, tag = "1")] pub status: String,
}

// =============================================================================
// Read state / mute
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct MarkChannelReadRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub channel_id: i64,
    #[prost(int64, tag = "3")] pub message_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MarkChannelReadResponse {
    #[prost(string, tag = "1")] pub status: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetChannelUnreadCountsRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetChannelUnreadCountsResponse {
    #[prost(map = "string, int64", tag = "1")] pub unread: HashMap<String, i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MuteChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(bool, tag = "3")] pub muted: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MuteChannelResponse {
    #[prost(string, tag = "1")] pub status: String,
}

// =============================================================================
// Members
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelMembersRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(int32, optional, tag = "3")] pub limit: Option<i32>,
    #[prost(int32, optional, tag = "4")] pub offset: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelMembersResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<ChannelMember>,
    #[prost(int64, tag = "2")] pub total: i64,
    #[prost(int32, tag = "3")] pub limit: i32,
    #[prost(int32, tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct JoinChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct JoinChannelResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LeaveChannelRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LeaveChannelResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InviteChannelMembersRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(int64, repeated, tag = "3")] pub user_ids: Vec<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InviteChannelMembersResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveChannelMemberRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(int64, tag = "3")] pub user_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RemoveChannelMemberResponse {
    #[prost(string, tag = "1")] pub message: String,
}

// =============================================================================
// Channel pods
// =============================================================================

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelPodsRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListChannelPodsResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<ChannelPod>,
    #[prost(int64, tag = "2")] pub total: i64,
    #[prost(int32, tag = "3")] pub limit: i32,
    #[prost(int32, tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct JoinChannelPodRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(string, tag = "3")] pub pod_key: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct JoinChannelPodResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LeaveChannelPodRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int64, tag = "2")] pub id: i64,
    #[prost(string, tag = "3")] pub pod_key: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LeaveChannelPodResponse {
    #[prost(string, tag = "1")] pub message: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_channel() -> Channel {
        Channel {
            id: 42, organization_id: 7, name: "general".into(),
            description: Some("General chat".into()), document: None,
            repository_id: Some(99), ticket_id: None, ticket_slug: None,
            created_by_pod: None, created_by_user_id: Some(1),
            visibility: "public".into(), is_archived: false,
            is_member: true, member_count: 5,
            created_at: "2026-05-12T00:00:00Z".into(),
            updated_at: "2026-05-12T01:00:00Z".into(),
        }
    }

    fn sample_message() -> ChannelMessage {
        ChannelMessage {
            id: 100, channel_id: 42,
            sender_pod: None, sender_user_id: Some(7),
            message_type: "text".into(), body: "hello".into(),
            content_json: Some(r#"{"schema_version":1,"kind":"ast","blocks":[]}"#.into()),
            mentions_json: None, reply_to: None, edited_at: None,
            is_deleted: false, created_at: "2026-05-12T00:00:00Z".into(),
            sender_user: Some(ChannelMessageSenderUser {
                id: 7, username: "alice".into(),
                name: Some("Alice".into()), avatar_url: None,
            }),
            sender_pod_info: None,
        }
    }

    #[test]
    fn channel_round_trip_preserves_every_field() {
        let original = sample_channel();
        let bytes = original.encode_to_vec();
        let decoded = Channel::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn channel_message_round_trip_with_nested_sender() {
        let original = sample_message();
        let bytes = original.encode_to_vec();
        let decoded = ChannelMessage::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert!(decoded.sender_user.is_some());
    }

    #[test]
    fn list_channels_response_round_trip() {
        let original = ListChannelsResponse {
            items: vec![sample_channel()], total: 1, limit: 50, offset: 0,
        };
        let bytes = original.encode_to_vec();
        assert_eq!(original, ListChannelsResponse::decode(&*bytes).unwrap());
    }

    #[test]
    fn list_messages_response_preserves_has_more() {
        let original = ListChannelMessagesResponse {
            items: vec![sample_message()], total: 1, limit: 50, offset: 0,
            has_more: true,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListChannelMessagesResponse::decode(&*bytes).unwrap();
        assert_eq!(decoded.has_more, true,
            "has_more must round-trip — drift here regresses cursor pagination");
        assert_eq!(original, decoded);
    }

    #[test]
    fn list_members_response_round_trip() {
        let original = ListChannelMembersResponse {
            items: vec![ChannelMember {
                channel_id: 1, user_id: 7, role: "creator".into(),
                is_muted: false, joined_at: "2026-05-12T00:00:00Z".into(),
            }],
            total: 1, limit: 50, offset: 0,
        };
        let bytes = original.encode_to_vec();
        assert_eq!(original, ListChannelMembersResponse::decode(&*bytes).unwrap());
    }

    #[test]
    fn list_pods_response_round_trip() {
        let original = ListChannelPodsResponse {
            items: vec![ChannelPod {
                id: 1, pod_key: "pod-abc".into(),
                alias: Some("my-agent".into()),
                status: "running".into(), agent_status: "idle".into(),
            }],
            total: 1, limit: 0, offset: 0,
        };
        let bytes = original.encode_to_vec();
        assert_eq!(original, ListChannelPodsResponse::decode(&*bytes).unwrap());
    }

    #[test]
    fn send_message_request_carries_mentions_map() {
        let mut mentions = HashMap::new();
        mentions.insert("alice".into(), MentionRef {
            entity_type: "user".into(), entity_key: "7".into(),
        });
        let original = SendChannelMessageRequest {
            org_slug: "acme".into(), channel_id: 1,
            source: Some("hello @alice".into()),
            mentions, content_json: None,
            attachment_key: None, pod_key: None, reply_to: None,
        };
        let bytes = original.encode_to_vec();
        let decoded = SendChannelMessageRequest::decode(&*bytes).unwrap();
        assert_eq!(decoded.mentions.len(), 1);
        assert_eq!(decoded.mentions["alice"].entity_key, "7");
    }

    #[test]
    fn unread_counts_response_carries_map() {
        let mut unread = HashMap::new();
        unread.insert("1".into(), 5);
        unread.insert("2".into(), 0);
        let original = GetChannelUnreadCountsResponse { unread };
        let bytes = original.encode_to_vec();
        let decoded = GetChannelUnreadCountsResponse::decode(&*bytes).unwrap();
        assert_eq!(decoded.unread.get("1"), Some(&5));
        assert_eq!(decoded.unread.get("2"), Some(&0));
    }

    #[test]
    fn list_channels_request_offset_zero_distinguishable_from_absent() {
        let with_zero = ListChannelsRequest {
            org_slug: "acme".into(), repository_id: None,
            ticket_slug: None, include_archived: None,
            offset: Some(0), limit: None,
        };
        let absent = ListChannelsRequest {
            org_slug: "acme".into(), repository_id: None,
            ticket_slug: None, include_archived: None,
            offset: None, limit: None,
        };
        assert_ne!(with_zero.encode_to_vec(), absent.encode_to_vec(),
            "explicit zero offset must encode differently from absent");
    }

    #[test]
    fn ack_responses_round_trip() {
        let arc = ArchiveChannelResponse { message: "Channel archived".into() };
        assert_eq!(arc, ArchiveChannelResponse::decode(&*arc.encode_to_vec()).unwrap());
        let unr = UnarchiveChannelResponse { message: "Channel unarchived".into() };
        assert_eq!(unr, UnarchiveChannelResponse::decode(&*unr.encode_to_vec()).unwrap());
        let del = DeleteChannelMessageResponse { status: "deleted".into() };
        assert_eq!(del, DeleteChannelMessageResponse::decode(&*del.encode_to_vec()).unwrap());
        let mrk = MarkChannelReadResponse { status: "ok".into() };
        assert_eq!(mrk, MarkChannelReadResponse::decode(&*mrk.encode_to_vec()).unwrap());
    }
}
