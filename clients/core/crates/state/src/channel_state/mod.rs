// ChannelState — client-side cache of channels + messages + per-channel
// derived counters (unread/mention/last_message/last_activity_at).
//
// The derived counters live INSIDE each Channel struct (proto fields tag
// 100+ in proto.channel_state.v1) rather than in side-channel HashMaps:
// this makes the schema the SSOT for UI render and lets `channels_json`
// from the wasm bridge produce sidebar-ready rows in one pass.
//
// Implementation is split across sibling files by responsibility:
//   * channels.rs  — CRUD / select / filter / sort
//   * messages.rs  — message cache, add / update / remove / paginate
//   * counters.rs  — unread + mention business logic on Channel struct
//   * preview.rs   — MessagePreview generation + last_message access
//   * members.rs   — channel_members / channel_pods cache
//   * user.rs      — current_user + enrich_sender

mod channels;
mod counters;
mod members;
mod messages;
mod preview;
mod user;
mod wire_convert;

use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_persistence::{ChannelRepo, MessageRepo, StorageBackend};
use agentsmesh_types::proto_pod_v1::Pod;

use crate::channel_types::{Channel, ChannelMember, ChannelMessage, User};

pub(crate) const MAX_MESSAGES_PER_CHANNEL: usize = 500;
pub(crate) const MAX_CACHED_CHANNELS: usize = 50;
pub(crate) const PREVIEW_MAX_CHARS: usize = 80;

pub struct ChannelMessageCache {
    pub messages: Vec<ChannelMessage>,
    pub has_more: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChannelSortMode {
    LastMessage,
    UnreadFirst,
    Name,
}

pub struct ChannelState {
    pub(super) channels: Vec<Channel>,
    pub(super) current_channel: Option<Channel>,
    pub(super) message_cache: HashMap<i64, ChannelMessageCache>,
    pub(super) members_by_channel: HashMap<i64, Vec<ChannelMember>>,
    pub(super) pods_by_channel: HashMap<i64, Vec<Pod>>,
    pub(super) current_user: Option<User>,
    pub(super) channel_repo: Option<ChannelRepo<Channel>>,
    pub(super) message_repo: Option<MessageRepo<ChannelMessage>>,
}

impl ChannelState {
    pub fn new() -> Self {
        Self {
            channels: Vec::new(),
            current_channel: None,
            message_cache: HashMap::new(),
            members_by_channel: HashMap::new(),
            pods_by_channel: HashMap::new(),
            current_user: None,
            channel_repo: None,
            message_repo: None,
        }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        let channel_repo = ChannelRepo::new(backend.clone());
        let message_repo = MessageRepo::new(backend);
        let channels = channel_repo.list_all().unwrap_or_default();
        Self {
            channels,
            current_channel: None,
            message_cache: HashMap::new(),
            members_by_channel: HashMap::new(),
            pods_by_channel: HashMap::new(),
            current_user: None,
            channel_repo: Some(channel_repo),
            message_repo: Some(message_repo),
        }
    }
}

impl Default for ChannelState {
    fn default() -> Self {
        Self::new()
    }
}
