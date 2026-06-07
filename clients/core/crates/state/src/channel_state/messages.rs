use std::collections::HashSet;

use super::{ChannelMessageCache, ChannelState, MAX_CACHED_CHANNELS, MAX_MESSAGES_PER_CHANNEL};
use crate::channel_types::ChannelMessage;

impl ChannelState {
    pub fn add_message(&mut self, channel_id: i64, message: ChannelMessage) -> bool {
        tracing::debug!(target: "channel", channel_id, msg_id = message.id, "add message");
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

    /// Handle a new incoming message (from a realtime event). Rust Core is
    /// the SSOT for ALL derived state here — front-ends only project the
    /// result, they never reconstruct these rules:
    ///   1. enrich sender (fill current_user when sender is self)
    ///   2. compute + store the channel-list preview / last_activity_at
    ///   3. persist to the message repo
    ///   4. append to the in-memory cache
    ///   5. increment unread — UNLESS the message is the current user's own,
    ///      or they're actively viewing the channel
    ///   6. increment mention — same gating, when the current user (or
    ///      @channel) is mentioned
    ///
    /// Returns true when the message was newly added (false on dup id).
    pub fn on_new_message(&mut self, mut msg: ChannelMessage) -> bool {
        let channel_id = msg.channel_id;
        tracing::debug!(target: "channel", channel_id, msg_id = msg.id, "on new message");
        self.enrich_sender(&mut msg);
        let preview = Self::make_preview(&msg);
        self.set_last_message(channel_id, preview);

        // Read derived-count inputs before `msg` is moved into add_message.
        let is_self =
            msg.sender_user_id.is_some() && msg.sender_user_id == self.current_user_id();
        let is_active = self.current_channel.as_ref().map(|c| c.id) == Some(channel_id);
        let mentions_me = self.message_mentions_current_user(&msg);

        let added = self.add_message(channel_id, msg);
        if added && !is_self && !is_active {
            self.increment_unread(channel_id);
            if mentions_me {
                self.increment_mention(channel_id);
            }
        }
        added
    }

    /// True when `msg.mentions_json` targets the current user — either an
    /// explicit user-id mention or an `@channel` broadcast. Mirrors the
    /// web `MessageMentions` shape `{ pods?: string[], users?: i64[],
    /// channel?: bool }`. Returns false when no current user is set.
    fn message_mentions_current_user(&self, msg: &ChannelMessage) -> bool {
        let Some(uid) = self.current_user_id() else { return false };
        let Some(json) = msg.mentions_json.as_ref() else { return false };
        let Ok(v) = serde_json::from_str::<serde_json::Value>(json) else { return false };
        if v.get("channel").and_then(|c| c.as_bool()).unwrap_or(false) {
            return true;
        }
        v.get("users")
            .and_then(|u| u.as_array())
            .map(|arr| arr.iter().any(|x| x.as_i64() == Some(uid)))
            .unwrap_or(false)
    }

    pub fn update_message(&mut self, channel_id: i64, message: ChannelMessage) {
        tracing::debug!(target: "channel", channel_id, msg_id = message.id, "update message");
        if let Some(cache) = self.message_cache.get_mut(&channel_id) {
            if let Some(m) = cache.messages.iter_mut().find(|m| m.id == message.id) {
                if !message.body.is_empty() { m.body = message.body; }
                if message.content_json.is_some() { m.content_json = message.content_json; }
                if message.mentions_json.is_some() { m.mentions_json = message.mentions_json; }
                if message.reply_to.is_some() { m.reply_to = message.reply_to; }
                if message.edited_at.is_some() { m.edited_at = message.edited_at; }
                if message.metadata_json.is_some() { m.metadata_json = message.metadata_json; }
                if message.sender_user.is_some() { m.sender_user = message.sender_user; }
                if message.sender_user_id.is_some() { m.sender_user_id = message.sender_user_id; }
                if message.sender_pod.is_some() { m.sender_pod = message.sender_pod; }
                if message.sender_pod_info.is_some() { m.sender_pod_info = message.sender_pod_info; }
                if message.message_type.is_some() { m.message_type = message.message_type; }
                if message.is_deleted.is_some() { m.is_deleted = message.is_deleted; }
                if let Some(repo) = &self.message_repo { let _ = repo.save_message(m); }
            }
        }
    }

    pub fn remove_message(&mut self, channel_id: i64, message_id: i64) {
        tracing::debug!(target: "channel", channel_id, msg_id = message_id, "remove message");
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
        tracing::debug!(target: "channel", channel_id, count = messages.len(), has_more, "set messages (baseline)");
        if let Some(repo) = &self.message_repo {
            for msg in &messages {
                let _ = repo.save_message(msg);
            }
        }
        if let Some(newest) = messages.last() {
            let preview = Self::make_preview(newest);
            self.set_last_message(channel_id, preview);
        }
        let msg_len = messages.len();
        let (truncated, overflow) = if msg_len > MAX_MESSAGES_PER_CHANNEL {
            (messages[msg_len - MAX_MESSAGES_PER_CHANNEL..].to_vec(), true)
        } else {
            (messages, false)
        };
        self.message_cache.insert(
            channel_id,
            ChannelMessageCache { messages: truncated, has_more: has_more || overflow },
        );
        self.evict_stale_channels(channel_id);
    }

    pub fn prepend_messages(&mut self, channel_id: i64, older: Vec<ChannelMessage>, has_more: bool) {
        tracing::debug!(target: "channel", channel_id, count = older.len(), has_more, "prepend messages");
        let cache = self
            .message_cache
            .entry(channel_id)
            .or_insert_with(|| ChannelMessageCache { messages: Vec::new(), has_more: false });

        let existing_ids: HashSet<i64> = cache.messages.iter().map(|m| m.id).collect();
        let mut merged: Vec<ChannelMessage> = older
            .into_iter()
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

    pub(crate) fn evict_stale_channels(&mut self, keep_id: i64) {
        while self.message_cache.len() > MAX_CACHED_CHANNELS {
            let evict_key = self.message_cache.keys().find(|&&k| k != keep_id).copied();
            match evict_key {
                Some(k) => { self.message_cache.remove(&k); }
                None => break,
            }
        }
    }
}
