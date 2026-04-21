use std::collections::HashMap;

use serde::{Deserialize, Serialize};

use crate::User;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Channel {
    pub id: i64,
    pub name: String,
    #[serde(default)]
    pub description: Option<String>,
    #[serde(default)]
    pub is_archived: bool,
    #[serde(default)]
    pub visibility: Option<String>,
    #[serde(default)]
    pub is_member: bool,
    #[serde(default)]
    pub member_count: Option<i64>,
    #[serde(default)]
    pub organization_id: Option<i64>,
    #[serde(default)]
    pub document: Option<String>,
    #[serde(default)]
    pub repository_id: Option<i64>,
    #[serde(default)]
    pub ticket_id: Option<i64>,
    #[serde(default)]
    pub ticket_slug: Option<String>,
    #[serde(default)]
    pub created_by_pod: Option<String>,
    #[serde(default)]
    pub created_by_user_id: Option<i64>,
    #[serde(default)]
    pub created_at: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SenderPodInfo {
    pub pod_key: String,
    #[serde(default)]
    pub alias: Option<String>,
    #[serde(default)]
    pub agent: Option<SenderAgentInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SenderAgentInfo {
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelMessage {
    pub id: i64,
    pub channel_id: i64,
    /// Plain-text projection of `content`, derived server-side. Always present.
    #[serde(default)]
    pub body: String,
    /// Structured message content (AST blocks + inline elements). JSON, opaque to Rust.
    #[serde(default)]
    pub content: Option<serde_json::Value>,
    /// Structured mentions (pods / users / channel). JSON, opaque to Rust.
    #[serde(default)]
    pub mentions: Option<serde_json::Value>,
    /// ID of the message being replied to, if any.
    #[serde(default)]
    pub reply_to: Option<i64>,
    #[serde(default)]
    pub sender_user: Option<User>,
    #[serde(default)]
    pub sender_user_id: Option<i64>,
    #[serde(default)]
    pub sender_pod: Option<String>,
    #[serde(default)]
    pub sender_pod_info: Option<SenderPodInfo>,
    #[serde(default)]
    pub message_type: Option<String>,
    /// Deprecated: kept to tolerate legacy payloads. Prefer `sender_pod`.
    #[serde(default)]
    pub pod_key: Option<String>,
    /// Deprecated: moved into `content`. Kept to tolerate legacy payloads.
    #[serde(default)]
    pub metadata: Option<serde_json::Value>,
    #[serde(default)]
    pub edited_at: Option<String>,
    #[serde(default)]
    pub is_deleted: Option<bool>,
    #[serde(default)]
    pub created_at: Option<String>,
}

/// Preview of the last message in a channel, used for channel list display.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessagePreview {
    pub sender_name: String,
    pub content_preview: String,
    #[serde(default)]
    pub message_type: Option<String>,
    pub timestamp: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateChannelRequest {
    pub name: String,
    pub description: Option<String>,
    pub document: Option<String>,
    pub repository_id: Option<i64>,
    pub ticket_slug: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateChannelRequest {
    pub name: Option<String>,
    pub description: Option<String>,
}

/// Request body for POST /channels/{id}/messages.
///
/// `content` is the structured AST (schema_version + kind + blocks[] +
/// inline elements). The backend derives `body` from it and stores both.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendChannelMessageRequest {
    pub content: serde_json::Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pod_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<i64>,
}

/// Request body for PUT /channels/{id}/messages/{msgId}. Same shape as
/// SendChannelMessageRequest minus addressing fields.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EditChannelMessageRequest {
    pub content: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JoinChannelPodRequest {
    pub pod_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MuteChannelRequest {
    pub muted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelListResponse {
    pub channels: Vec<Channel>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelMember {
    pub channel_id: i64,
    pub user_id: i64,
    pub role: String,
    pub is_muted: bool,
    pub joined_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelMemberListResponse {
    pub members: Vec<ChannelMember>,
    pub total: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InviteChannelMembersRequest {
    pub user_ids: Vec<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelMessageListResponse {
    pub messages: Vec<ChannelMessage>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelUnreadResponse {
    #[serde(default)]
    pub unread: HashMap<String, u32>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn channel_roundtrip() {
        let ch = Channel {
            id: 1, name: "general".into(), description: Some("General channel".into()),
            is_archived: false, organization_id: Some(10), document: None,
            repository_id: Some(42), ticket_id: None, ticket_slug: None,
            created_by_pod: None, created_by_user_id: None,
            created_at: None, updated_at: None,
        };
        let json = serde_json::to_string(&ch).unwrap();
        let decoded: Channel = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.name, "general");
        assert!(!decoded.is_archived);
        assert_eq!(decoded.organization_id, Some(10));
    }

    #[test]
    fn channel_minimal_json() {
        let json = r#"{"id":1,"name":"ch","is_archived":true}"#;
        let ch: Channel = serde_json::from_str(json).unwrap();
        assert!(ch.is_archived);
        assert!(ch.description.is_none());
        assert!(ch.repository_id.is_none());
        assert!(ch.organization_id.is_none());
    }

    #[test]
    fn channel_message_roundtrip() {
        let msg = ChannelMessage {
            id: 100, channel_id: 1,
            body: "Hello from agent".into(),
            content: Some(serde_json::json!({"schema_version": 1, "kind": "ast", "blocks": []})),
            mentions: None, reply_to: None,
            sender_user: Some(User { id: 1, email: "a@b.com".into(), username: "u".into(), name: None, avatar_url: None }),
            sender_user_id: Some(1), sender_pod: Some("pod-1".into()),
            sender_pod_info: Some(SenderPodInfo { pod_key: "pod-1".into(), alias: Some("my-agent".into()), agent: Some(SenderAgentInfo { name: "claude".into() }) }),
            message_type: Some("text".into()), pod_key: Some("pod-1".into()),
            metadata: None, edited_at: None, is_deleted: None,
            created_at: Some("2026-01-01T00:00:00Z".into()),
        };
        let json = serde_json::to_string(&msg).unwrap();
        let decoded: ChannelMessage = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.body, "Hello from agent");
        assert!(decoded.content.is_some());
        assert!(decoded.sender_user.is_some());
        assert_eq!(decoded.sender_pod_info.unwrap().agent.unwrap().name, "claude");
    }

    #[test]
    fn channel_message_minimal() {
        let json = r#"{"id":1,"channel_id":2,"body":"hi"}"#;
        let msg: ChannelMessage = serde_json::from_str(json).unwrap();
        assert_eq!(msg.channel_id, 2);
        assert_eq!(msg.body, "hi");
        assert!(msg.content.is_none());
        assert!(msg.mentions.is_none());
        assert!(msg.reply_to.is_none());
        assert!(msg.sender_user.is_none());
    }

    #[test]
    fn channel_message_with_structured_content() {
        let json = r#"{
            "id": 1, "channel_id": 2, "body": "Hey @alice",
            "content": {"schema_version": 1, "kind": "ast", "blocks": [
                {"type": "paragraph", "elements": [
                    {"type": "text", "text": "Hey "},
                    {"type": "mention", "entity_type": "user", "entity_key": "alice", "display": "alice"}
                ]}
            ]},
            "mentions": {"users": ["alice"]},
            "reply_to": 99
        }"#;
        let msg: ChannelMessage = serde_json::from_str(json).unwrap();
        assert_eq!(msg.body, "Hey @alice");
        assert!(msg.content.is_some());
        assert_eq!(msg.reply_to, Some(99));
        assert!(msg.mentions.is_some());
    }

    #[test]
    fn message_preview_roundtrip() {
        let preview = MessagePreview {
            sender_name: "alice".into(), content_preview: "Hello world...".into(),
            message_type: Some("text".into()), timestamp: "2026-01-01T00:00:00Z".into(),
        };
        let json = serde_json::to_string(&preview).unwrap();
        let decoded: MessagePreview = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.sender_name, "alice");
    }

    #[test]
    fn channel_unread_response_typed() {
        let json = r#"{"unread":{"1":5,"2":0,"3":12}}"#;
        let resp: ChannelUnreadResponse = serde_json::from_str(json).unwrap();
        assert_eq!(resp.unread.get("1"), Some(&5));
        assert_eq!(resp.unread.get("3"), Some(&12));
    }

    #[test]
    fn channel_unread_response_empty() {
        let json = r#"{"unread":{}}"#;
        let resp: ChannelUnreadResponse = serde_json::from_str(json).unwrap();
        assert!(resp.unread.is_empty());
    }
}
