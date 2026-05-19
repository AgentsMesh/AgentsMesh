use std::collections::{HashMap, HashSet};
use std::sync::Arc;

use agentsmesh_persistence::{ChannelRepo, MessageRepo, StorageBackend};
use agentsmesh_types::{Channel, ChannelMember, ChannelMessage, MessagePreview, Pod, User};

const MAX_MESSAGES_PER_CHANNEL: usize = 500;
const MAX_CACHED_CHANNELS: usize = 50;
const PREVIEW_MAX_CHARS: usize = 80;

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
    channels: Vec<Channel>,
    current_channel: Option<Channel>,
    message_cache: HashMap<i64, ChannelMessageCache>,
    unread_counts: HashMap<i64, u32>,
    mention_counts: HashMap<i64, u32>,
    last_messages: HashMap<i64, MessagePreview>,
    members_by_channel: HashMap<i64, Vec<ChannelMember>>,
    pods_by_channel: HashMap<i64, Vec<Pod>>,
    current_user: Option<User>,
    channel_repo: Option<ChannelRepo>,
    message_repo: Option<MessageRepo>,
}

impl ChannelState {
    pub fn new() -> Self {
        Self {
            channels: Vec::new(),
            current_channel: None,
            message_cache: HashMap::new(),
            unread_counts: HashMap::new(),
            mention_counts: HashMap::new(),
            last_messages: HashMap::new(),
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
            unread_counts: HashMap::new(),
            mention_counts: HashMap::new(),
            last_messages: HashMap::new(),
            members_by_channel: HashMap::new(),
            pods_by_channel: HashMap::new(),
            current_user: None,
            channel_repo: Some(channel_repo),
            message_repo: Some(message_repo),
        }
    }

    pub fn set_current_user_id(&mut self, user_id: Option<i64>) {
        match user_id {
            Some(id) => {
                if self.current_user.as_ref().map(|u| u.id) != Some(id) {
                    self.current_user = Some(User {
                        id, email: String::new(), username: String::new(),
                        name: None, avatar_url: None, is_email_verified: None,
                    });
                }
            }
            None => self.current_user = None,
        }
    }

    pub fn set_current_user(&mut self, user: Option<User>) {
        self.current_user = user;
    }

    pub fn current_user_id(&self) -> Option<i64> {
        self.current_user.as_ref().map(|u| u.id)
    }

    pub fn current_user(&self) -> Option<&User> {
        self.current_user.as_ref()
    }

    pub fn enrich_sender(&self, msg: &mut ChannelMessage) {
        if msg.sender_user.is_some() {
            return;
        }
        if let (Some(sender_id), Some(user)) = (msg.sender_user_id, &self.current_user) {
            if sender_id == user.id && !user.username.is_empty() {
                msg.sender_user = Some(user.clone());
            }
        }
    }

    pub fn get_channels(&self) -> &[Channel] {
        &self.channels
    }

    pub fn get_current_channel(&self) -> Option<&Channel> {
        self.current_channel.as_ref()
    }

    pub fn set_channels(&mut self, channels: Vec<Channel>) {
        self.channels = channels;
        if let Some(repo) = &self.channel_repo {
            for ch in &self.channels {
                let _ = repo.save(ch);
            }
        }
    }

    pub fn set_current_channel(&mut self, id: Option<i64>) {
        self.current_channel = id.and_then(|id| self.channels.iter().find(|c| c.id == id).cloned());
    }

    pub fn get_channel(&self, id: i64) -> Option<&Channel> {
        self.channels.iter().find(|c| c.id == id)
    }

    pub fn add_channel(&mut self, channel: Channel) {
        if self.channels.iter().any(|c| c.id == channel.id) {
            return;
        }
        if let Some(repo) = &self.channel_repo {
            let _ = repo.save(&channel);
        }
        self.channels.insert(0, channel);
    }

    pub fn update_channel(&mut self, id: i64, channel: Channel) {
        if let Some(existing) = self.channels.iter_mut().find(|c| c.id == id) {
            *existing = channel.clone();
            if let Some(repo) = &self.channel_repo {
                let _ = repo.save(existing);
            }
        }
        if self.current_channel.as_ref().is_some_and(|c| c.id == id) {
            self.current_channel = Some(channel);
        }
    }

    pub fn remove_channel(&mut self, id: i64) {
        self.channels.retain(|c| c.id != id);
        if self.current_channel.as_ref().is_some_and(|c| c.id == id) {
            self.current_channel = None;
        }
        self.message_cache.remove(&id);
        self.unread_counts.remove(&id);
        self.mention_counts.remove(&id);
        self.last_messages.remove(&id);
    }

    pub fn filter_channels(&self, query: &str, include_archived: bool) -> Vec<&Channel> {
        let q = query.to_lowercase();
        self.channels.iter()
            .filter(|c| include_archived || !c.is_archived)
            .filter(|c| {
                if q.is_empty() { return true; }
                c.name.to_lowercase().contains(&q)
                    || c.description.as_deref().unwrap_or("").to_lowercase().contains(&q)
            })
            .collect()
    }

    pub fn select_channel(&mut self, id: Option<i64>) -> Option<&Channel> {
        self.set_current_channel(id);
        if let Some(id) = id {
            self.clear_channel_unread(id);
            self.clear_channel_mentions(id);
        }
        self.current_channel.as_ref()
    }

    pub fn sorted_channel_ids(&self, mode: ChannelSortMode, include_archived: bool) -> Vec<i64> {
        let mut entries: Vec<(i64, &Channel)> = self.channels.iter()
            .filter(|c| include_archived || !c.is_archived)
            .map(|c| (c.id, c))
            .collect();

        match mode {
            ChannelSortMode::LastMessage => {
                entries.sort_by(|a, b| {
                    let ta = self.last_messages.get(&a.0).map(|m| m.timestamp.as_str());
                    let tb = self.last_messages.get(&b.0).map(|m| m.timestamp.as_str());
                    match (ta, tb) {
                        (Some(ta), Some(tb)) => tb.cmp(ta),
                        (Some(_), None) => std::cmp::Ordering::Less,
                        (None, Some(_)) => std::cmp::Ordering::Greater,
                        (None, None) => {
                            let ua = a.1.updated_at.as_deref();
                            let ub = b.1.updated_at.as_deref();
                            ub.cmp(&ua)
                        }
                    }
                });
            }
            ChannelSortMode::UnreadFirst => {
                entries.sort_by(|a, b| {
                    let ua = self.unread_counts.get(&a.0).copied().unwrap_or(0);
                    let ub = self.unread_counts.get(&b.0).copied().unwrap_or(0);
                    let unread_cmp = (ub > 0).cmp(&(ua > 0));
                    if unread_cmp != std::cmp::Ordering::Equal {
                        return unread_cmp;
                    }
                    let ta = self.last_messages.get(&a.0).map(|m| m.timestamp.as_str());
                    let tb = self.last_messages.get(&b.0).map(|m| m.timestamp.as_str());
                    tb.cmp(&ta)
                });
            }
            ChannelSortMode::Name => {
                entries.sort_by(|a, b| a.1.name.to_lowercase().cmp(&b.1.name.to_lowercase()));
            }
        }

        entries.into_iter().map(|(id, _)| id).collect()
    }

    pub fn get_last_message(&self, channel_id: i64) -> Option<&MessagePreview> {
        self.last_messages.get(&channel_id)
    }

    pub fn set_last_message(&mut self, channel_id: i64, preview: MessagePreview) {
        self.last_messages.insert(channel_id, preview);
    }

    pub fn make_preview(msg: &ChannelMessage) -> MessagePreview {
        let sender = msg.sender_user.as_ref()
            .map(|u| u.name.as_deref().unwrap_or(&u.username).to_string())
            .or_else(|| msg.sender_pod_info.as_ref().map(|p| {
                p.agent.as_ref().map(|a| a.name.clone())
                    .unwrap_or_else(|| p.alias.clone().unwrap_or_else(|| p.pod_key.clone()))
            }))
            .unwrap_or_default();

        let preview = match msg.message_type.as_deref() {
            Some("code") => "[Code]".to_string(),
            Some("command") => "[Command]".to_string(),
            Some("system") => truncate_str(&msg.body, PREVIEW_MAX_CHARS),
            _ => truncate_str(&msg.body, PREVIEW_MAX_CHARS),
        };

        MessagePreview {
            sender_name: sender,
            content_preview: preview,
            message_type: msg.message_type.clone(),
            timestamp: msg.created_at.clone().unwrap_or_default(),
        }
    }

    pub fn add_message(&mut self, channel_id: i64, message: ChannelMessage) -> bool {
        let cache = self
            .message_cache
            .entry(channel_id)
            .or_insert_with(|| ChannelMessageCache { messages: Vec::new(), has_more: false });

        if cache.messages.iter().any(|m| m.id == message.id) {
            return false;
        }
        if let Some(repo) = &self.message_repo {
            let _ = repo.save_message(&message);
        }
        cache.messages.push(message);
        if cache.messages.len() > MAX_MESSAGES_PER_CHANNEL {
            cache.messages.drain(..cache.messages.len() - MAX_MESSAGES_PER_CHANNEL);
            cache.has_more = true;
        }
        self.evict_stale_channels(channel_id);
        true
    }

    pub fn on_new_message(&mut self, mut msg: ChannelMessage) -> bool {
        let channel_id = msg.channel_id;

        self.enrich_sender(&mut msg);

        self.last_messages.insert(channel_id, Self::make_preview(&msg));

        self.add_message(channel_id, msg)
    }

    pub fn update_message(&mut self, channel_id: i64, message: ChannelMessage) {
        if let Some(cache) = self.message_cache.get_mut(&channel_id) {
            if let Some(m) = cache.messages.iter_mut().find(|m| m.id == message.id) {
                if !message.body.is_empty() { m.body = message.body; }
                if message.content.is_some() { m.content = message.content; }
                if message.mentions.is_some() { m.mentions = message.mentions; }
                if message.reply_to.is_some() { m.reply_to = message.reply_to; }
                if message.edited_at.is_some() { m.edited_at = message.edited_at; }
                if message.metadata.is_some() { m.metadata = message.metadata; }
                if message.sender_user.is_some() { m.sender_user = message.sender_user; }
                if message.sender_user_id.is_some() { m.sender_user_id = message.sender_user_id; }
                if message.sender_pod.is_some() { m.sender_pod = message.sender_pod; }
                if message.sender_pod_info.is_some() { m.sender_pod_info = message.sender_pod_info; }
                if message.message_type.is_some() { m.message_type = message.message_type; }
                if message.is_deleted.is_some() { m.is_deleted = message.is_deleted; }
                if let Some(repo) = &self.message_repo {
                    let _ = repo.save_message(m);
                }
            }
        }
    }

    pub fn remove_message(&mut self, channel_id: i64, message_id: i64) {
        if let Some(cache) = self.message_cache.get_mut(&channel_id) {
            cache.messages.retain(|m| m.id != message_id);
            if let Some(repo) = &self.message_repo {
                let _ = repo.delete_message(message_id);
            }
        }
    }

    pub fn get_messages(&self, channel_id: i64) -> Option<&ChannelMessageCache> {
        self.message_cache.get(&channel_id)
    }

    pub fn set_messages(&mut self, channel_id: i64, messages: Vec<ChannelMessage>, has_more: bool) {
        if let Some(repo) = &self.message_repo {
            for msg in &messages {
                let _ = repo.save_message(msg);
            }
        }
        if let Some(newest) = messages.last() {
            self.last_messages.insert(channel_id, Self::make_preview(newest));
        }
        let msg_len = messages.len();
        let (truncated, overflow) = if msg_len > MAX_MESSAGES_PER_CHANNEL {
            (messages[msg_len - MAX_MESSAGES_PER_CHANNEL..].to_vec(), true)
        } else {
            (messages, false)
        };
        self.message_cache.insert(channel_id, ChannelMessageCache { messages: truncated, has_more: has_more || overflow });
        self.evict_stale_channels(channel_id);
    }

    pub fn prepend_messages(&mut self, channel_id: i64, older: Vec<ChannelMessage>, has_more: bool) {
        let cache = self.message_cache.entry(channel_id)
            .or_insert_with(|| ChannelMessageCache { messages: Vec::new(), has_more: false });

        let existing_ids: HashSet<i64> = cache.messages.iter().map(|m| m.id).collect();
        let mut merged: Vec<ChannelMessage> = older.into_iter()
            .filter(|m| !existing_ids.contains(&m.id))
            .collect();
        merged.extend(cache.messages.drain(..));
        merged.sort_by_key(|m| m.id);

        if let Some(repo) = &self.message_repo {
            for msg in &merged {
                if !existing_ids.contains(&msg.id) {
                    let _ = repo.save_message(msg);
                }
            }
        }

        if merged.len() > MAX_MESSAGES_PER_CHANNEL {
            merged.drain(..merged.len() - MAX_MESSAGES_PER_CHANNEL);
        }

        cache.messages = merged;
        cache.has_more = has_more;
        self.evict_stale_channels(channel_id);
    }

    pub fn set_unread_counts(&mut self, counts: HashMap<i64, u32>) {
        self.unread_counts = counts;
    }

    pub fn increment_unread(&mut self, channel_id: i64) {
        *self.unread_counts.entry(channel_id).or_insert(0) += 1;
    }

    pub fn clear_channel_unread(&mut self, channel_id: i64) {
        self.unread_counts.insert(channel_id, 0);
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 {
        self.unread_counts.get(&channel_id).copied().unwrap_or(0)
    }

    pub fn total_unread_count(&self) -> u32 {
        let scope: std::collections::HashSet<i64> = self.channels.iter().map(|c| c.id).collect();
        self.unread_counts
            .iter()
            .filter_map(|(id, n)| if scope.contains(id) { Some(*n) } else { None })
            .sum()
    }

    pub fn get_all_unread_counts(&self) -> &HashMap<i64, u32> {
        &self.unread_counts
    }

    pub fn increment_mention(&mut self, channel_id: i64) {
        *self.mention_counts.entry(channel_id).or_insert(0) += 1;
    }

    pub fn clear_channel_mentions(&mut self, channel_id: i64) {
        self.mention_counts.insert(channel_id, 0);
    }

    pub fn get_mention_count(&self, channel_id: i64) -> u32 {
        self.mention_counts.get(&channel_id).copied().unwrap_or(0)
    }

    pub fn total_mention_count(&self) -> u32 {
        let scope: std::collections::HashSet<i64> = self.channels.iter().map(|c| c.id).collect();
        self.mention_counts
            .iter()
            .filter_map(|(id, n)| if scope.contains(id) { Some(*n) } else { None })
            .sum()
    }

    pub fn get_all_mention_counts(&self) -> &HashMap<i64, u32> {
        &self.mention_counts
    }

    pub fn set_mention_counts(&mut self, counts: HashMap<i64, u32>) {
        self.mention_counts = counts;
    }

    fn evict_stale_channels(&mut self, keep_id: i64) {
        while self.message_cache.len() > MAX_CACHED_CHANNELS {
            let evict_key = self
                .message_cache
                .keys()
                .find(|&&k| k != keep_id)
                .copied();
            match evict_key {
                Some(k) => { self.message_cache.remove(&k); }
                None => break,
            }
        }
    }

    pub fn set_channel_members(&mut self, channel_id: i64, members: Vec<ChannelMember>) {
        self.members_by_channel.insert(channel_id, members);
    }

    pub fn get_channel_members(&self, channel_id: i64) -> Vec<ChannelMember> {
        self.members_by_channel.get(&channel_id).cloned().unwrap_or_default()
    }

    pub fn remove_channel_member(&mut self, channel_id: i64, user_id: i64) {
        if let Some(list) = self.members_by_channel.get_mut(&channel_id) {
            list.retain(|m| m.user_id != user_id);
        }
    }

    pub fn clear_channel_members(&mut self, channel_id: i64) {
        self.members_by_channel.remove(&channel_id);
    }

    pub fn set_channel_pods(&mut self, channel_id: i64, pods: Vec<Pod>) {
        self.pods_by_channel.insert(channel_id, pods);
    }

    pub fn get_channel_pods(&self, channel_id: i64) -> Vec<Pod> {
        self.pods_by_channel.get(&channel_id).cloned().unwrap_or_default()
    }

    pub fn clear_channel_pods(&mut self, channel_id: i64) {
        self.pods_by_channel.remove(&channel_id);
    }
}

impl Default for ChannelState {
    fn default() -> Self {
        Self::new()
    }
}

fn truncate_str(s: &str, max_chars: usize) -> String {
    let chars: Vec<char> = s.chars().take(max_chars + 1).collect();
    if chars.len() > max_chars {
        let mut result: String = chars[..max_chars].iter().collect();
        result.push('…');
        result
    } else {
        s.to_string()
    }
}
