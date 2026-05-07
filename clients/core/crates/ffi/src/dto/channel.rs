use std::collections::HashMap;

use agentsmesh_types::{
    Channel, ChannelListResponse, ChannelMember, ChannelMemberListResponse, ChannelMessage,
    ChannelMessageListResponse, ChannelUnreadResponse, CreateChannelRequest,
    EditChannelMessageRequest, InviteChannelMembersRequest, JoinChannelPodRequest,
    MessagePreview, MuteChannelRequest, SendChannelMessageRequest, SenderAgentInfo, SenderPodInfo,
    UpdateChannelRequest,
};

use super::UserDto;

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelDto {
    pub id: i64,
    pub name: String,
    pub description: Option<String>,
    pub is_archived: bool,
    pub visibility: Option<String>,
    pub is_member: bool,
    pub member_count: Option<i64>,
    pub organization_id: Option<i64>,
    pub document: Option<String>,
    pub repository_id: Option<i64>,
    pub ticket_id: Option<i64>,
    pub ticket_slug: Option<String>,
    pub created_by_pod: Option<String>,
    pub created_by_user_id: Option<i64>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

impl From<Channel> for ChannelDto {
    fn from(c: Channel) -> Self {
        Self {
            id: c.id,
            name: c.name,
            description: c.description,
            is_archived: c.is_archived,
            visibility: c.visibility,
            is_member: c.is_member,
            member_count: c.member_count,
            organization_id: c.organization_id,
            document: c.document,
            repository_id: c.repository_id,
            ticket_id: c.ticket_id,
            ticket_slug: c.ticket_slug,
            created_by_pod: c.created_by_pod,
            created_by_user_id: c.created_by_user_id,
            created_at: c.created_at,
            updated_at: c.updated_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelListResponseDto {
    pub channels: Vec<ChannelDto>,
}

impl From<ChannelListResponse> for ChannelListResponseDto {
    fn from(r: ChannelListResponse) -> Self {
        Self {
            channels: r.channels.into_iter().map(ChannelDto::from).collect(),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct SenderAgentInfoDto {
    pub name: String,
}

impl From<SenderAgentInfo> for SenderAgentInfoDto {
    fn from(a: SenderAgentInfo) -> Self {
        Self { name: a.name }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct SenderPodInfoDto {
    pub pod_key: String,
    pub alias: Option<String>,
    pub agent: Option<SenderAgentInfoDto>,
}

impl From<SenderPodInfo> for SenderPodInfoDto {
    fn from(p: SenderPodInfo) -> Self {
        Self {
            pod_key: p.pod_key,
            alias: p.alias,
            agent: p.agent.map(Into::into),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelMessageDto {
    pub id: i64,
    pub channel_id: i64,
    pub body: String,
    /// Structured AST (schema_version/kind/blocks) — opaque JSON, Swift decodes.
    pub content_json: Option<String>,
    /// Structured mentions — opaque JSON.
    pub mentions_json: Option<String>,
    pub reply_to: Option<i64>,
    pub sender_user: Option<UserDto>,
    pub sender_user_id: Option<i64>,
    pub sender_pod: Option<String>,
    pub sender_pod_info: Option<SenderPodInfoDto>,
    pub message_type: Option<String>,
    pub edited_at: Option<String>,
    pub is_deleted: Option<bool>,
    pub created_at: Option<String>,
}

fn value_to_json_opt(v: Option<serde_json::Value>) -> Option<String> {
    v.and_then(|val| serde_json::to_string(&val).ok())
}

impl From<ChannelMessage> for ChannelMessageDto {
    fn from(m: ChannelMessage) -> Self {
        Self {
            id: m.id,
            channel_id: m.channel_id,
            body: m.body,
            content_json: value_to_json_opt(m.content),
            mentions_json: value_to_json_opt(m.mentions),
            reply_to: m.reply_to,
            sender_user: m.sender_user.map(UserDto::from),
            sender_user_id: m.sender_user_id,
            sender_pod: m.sender_pod,
            sender_pod_info: m.sender_pod_info.map(Into::into),
            message_type: m.message_type,
            edited_at: m.edited_at,
            is_deleted: m.is_deleted,
            created_at: m.created_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelMessageListResponseDto {
    pub messages: Vec<ChannelMessageDto>,
}

impl From<ChannelMessageListResponse> for ChannelMessageListResponseDto {
    fn from(r: ChannelMessageListResponse) -> Self {
        Self {
            messages: r.messages.into_iter().map(ChannelMessageDto::from).collect(),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct MessagePreviewDto {
    pub sender_name: String,
    pub content_preview: String,
    pub message_type: Option<String>,
    pub timestamp: String,
}

impl From<MessagePreview> for MessagePreviewDto {
    fn from(p: MessagePreview) -> Self {
        Self {
            sender_name: p.sender_name,
            content_preview: p.content_preview,
            message_type: p.message_type,
            timestamp: p.timestamp,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateChannelRequestDto {
    pub name: String,
    pub description: Option<String>,
    pub document: Option<String>,
    pub repository_id: Option<i64>,
    pub ticket_slug: Option<String>,
}

impl From<CreateChannelRequestDto> for CreateChannelRequest {
    fn from(d: CreateChannelRequestDto) -> Self {
        Self {
            name: d.name,
            description: d.description,
            document: d.document,
            repository_id: d.repository_id,
            ticket_slug: d.ticket_slug,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateChannelRequestDto {
    pub name: Option<String>,
    pub description: Option<String>,
}

impl From<UpdateChannelRequestDto> for UpdateChannelRequest {
    fn from(d: UpdateChannelRequestDto) -> Self {
        Self {
            name: d.name,
            description: d.description,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelMemberDto {
    pub channel_id: i64,
    pub user_id: i64,
    pub role: String,
    pub is_muted: bool,
    pub joined_at: String,
}

impl From<ChannelMember> for ChannelMemberDto {
    fn from(m: ChannelMember) -> Self {
        Self {
            channel_id: m.channel_id,
            user_id: m.user_id,
            role: m.role,
            is_muted: m.is_muted,
            joined_at: m.joined_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelMemberListResponseDto {
    pub members: Vec<ChannelMemberDto>,
    pub total: i64,
}

impl From<ChannelMemberListResponse> for ChannelMemberListResponseDto {
    fn from(r: ChannelMemberListResponse) -> Self {
        Self {
            members: r.members.into_iter().map(ChannelMemberDto::from).collect(),
            total: r.total,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChannelUnreadResponseDto {
    pub unread: HashMap<String, u32>,
}

impl From<ChannelUnreadResponse> for ChannelUnreadResponseDto {
    fn from(r: ChannelUnreadResponse) -> Self {
        Self { unread: r.unread }
    }
}

/// Build a strong-typed `SendChannelMessageRequest`. Accepts either:
///   - the new wrapped shape `{source, mentions, content, attachment_key, ...}`
///     (any combination, all fields optional), or
///   - a bare `MessageContent` AST (legacy callers) which is rewrapped as
///     `{content: <ast>}` so it reaches the new server-side path.
/// Shape detection is structural: presence of any new top-level key wins,
/// otherwise the value is treated as a raw AST.
pub(crate) fn send_message_req(
    content_json: String,
    pod_key: Option<String>,
    reply_to: Option<i64>,
) -> Result<SendChannelMessageRequest, serde_json::Error> {
    let value: serde_json::Value = serde_json::from_str(&content_json)?;
    let mut req = if has_request_shape(&value) {
        serde_json::from_value::<SendChannelMessageRequest>(value)?
    } else {
        SendChannelMessageRequest { content: Some(value), ..Default::default() }
    };
    if pod_key.is_some() {
        req.pod_key = pod_key;
    }
    if reply_to.is_some() {
        req.reply_to = reply_to;
    }
    Ok(req)
}

pub(crate) fn edit_message_req(
    content_json: String,
) -> Result<EditChannelMessageRequest, serde_json::Error> {
    let value: serde_json::Value = serde_json::from_str(&content_json)?;
    if has_request_shape(&value) {
        serde_json::from_value(value)
    } else {
        Ok(EditChannelMessageRequest { content: Some(value), ..Default::default() })
    }
}

fn has_request_shape(v: &serde_json::Value) -> bool {
    let Some(obj) = v.as_object() else { return false };
    obj.contains_key("source")
        || obj.contains_key("content")
        || obj.contains_key("mentions")
        || obj.contains_key("attachment_key")
        || obj.contains_key("pod_key")
        || obj.contains_key("reply_to")
}

pub(crate) fn join_channel_pod_req(pod_key: String) -> JoinChannelPodRequest {
    JoinChannelPodRequest { pod_key }
}

pub(crate) fn mute_channel_req(muted: bool) -> MuteChannelRequest {
    MuteChannelRequest { muted }
}

pub(crate) fn invite_channel_members_req(user_ids: Vec<i64>) -> InviteChannelMembersRequest {
    InviteChannelMembersRequest { user_ids }
}
