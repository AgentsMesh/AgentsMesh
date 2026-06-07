use std::collections::HashMap;

use super::{ChannelSortMode, ChannelState};
use crate::channel_types::{Channel, MessagePreview};

impl ChannelState {
    pub fn get_channels(&self) -> &[Channel] {
        &self.channels
    }

    pub fn get_current_channel(&self) -> Option<&Channel> {
        self.current_channel.as_ref()
    }

    pub fn get_channel(&self, id: i64) -> Option<&Channel> {
        self.channels.iter().find(|c| c.id == id)
    }

    /// Replace channel list. Preserves client-derived state (unread/mention/
    /// last_message) that wire-side ListChannels won't carry — these are
    /// maintained by `on_new_message` and friends.
    pub fn set_channels(&mut self, channels: Vec<Channel>) {
        tracing::debug!(target: "channel", count = channels.len(), "set channels (baseline)");
        let mut prev: HashMap<i64, (u32, u32, Option<MessagePreview>, Option<String>)> =
            HashMap::with_capacity(self.channels.len());
        for c in &self.channels {
            prev.insert(
                c.id,
                (c.unread_count, c.mention_count, c.last_message.clone(), c.last_activity_at.clone()),
            );
        }
        self.channels = channels;
        for c in self.channels.iter_mut() {
            if let Some((u, m, lm, ts)) = prev.get(&c.id) {
                if c.unread_count == 0 { c.unread_count = *u; }
                if c.mention_count == 0 { c.mention_count = *m; }
                if c.last_message.is_none() { c.last_message = lm.clone(); }
                if c.last_activity_at.is_none() { c.last_activity_at = ts.clone(); }
            }
        }
        if let Some(repo) = &self.channel_repo {
            for ch in &self.channels {
                let _ = repo.save(ch);
            }
        }
    }

    pub fn set_current_channel(&mut self, id: Option<i64>) {
        self.current_channel = id.and_then(|id| self.channels.iter().find(|c| c.id == id).cloned());
    }

    pub fn add_channel(&mut self, channel: Channel) {
        tracing::info!(target: "channel", channel_id = channel.id, "add channel");
        if self.channels.iter().any(|c| c.id == channel.id) {
            return;
        }
        if let Some(repo) = &self.channel_repo {
            let _ = repo.save(&channel);
        }
        self.channels.insert(0, channel);
    }

    pub fn update_channel(&mut self, id: i64, channel: Channel) {
        tracing::info!(target: "channel", channel_id = id, "update channel");
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
        tracing::info!(target: "channel", channel_id = id, "remove channel");
        self.channels.retain(|c| c.id != id);
        if self.current_channel.as_ref().is_some_and(|c| c.id == id) {
            self.current_channel = None;
        }
        self.message_cache.remove(&id);
    }

    pub fn filter_channels(&self, query: &str, include_archived: bool) -> Vec<&Channel> {
        let q = query.to_lowercase();
        self.channels
            .iter()
            .filter(|c| include_archived || !c.is_archived)
            .filter(|c| {
                if q.is_empty() { return true; }
                c.name.to_lowercase().contains(&q)
                    || c.description.as_deref().unwrap_or("").to_lowercase().contains(&q)
            })
            .collect()
    }

    /// Atomically: set current channel + clear unread + clear mentions.
    pub fn select_channel(&mut self, id: Option<i64>) -> Option<&Channel> {
        tracing::debug!(target: "channel", channel_id = ?id, "select channel");
        self.set_current_channel(id);
        if let Some(id) = id {
            self.clear_channel_unread(id);
            self.clear_channel_mentions(id);
        }
        self.current_channel.as_ref()
    }

    pub fn sorted_channel_ids(&self, mode: ChannelSortMode, include_archived: bool) -> Vec<i64> {
        let mut entries: Vec<&Channel> = self
            .channels
            .iter()
            .filter(|c| include_archived || !c.is_archived)
            .collect();

        match mode {
            ChannelSortMode::LastMessage => {
                entries.sort_by(|a, b| {
                    match (a.last_activity_at.as_deref(), b.last_activity_at.as_deref()) {
                        (Some(ta), Some(tb)) => tb.cmp(ta),
                        (Some(_), None) => std::cmp::Ordering::Less,
                        (None, Some(_)) => std::cmp::Ordering::Greater,
                        (None, None) => b.updated_at.as_deref().cmp(&a.updated_at.as_deref()),
                    }
                });
            }
            ChannelSortMode::UnreadFirst => {
                entries.sort_by(|a, b| {
                    let unread_cmp = (b.unread_count > 0).cmp(&(a.unread_count > 0));
                    if unread_cmp != std::cmp::Ordering::Equal {
                        return unread_cmp;
                    }
                    b.last_activity_at.as_deref().cmp(&a.last_activity_at.as_deref())
                });
            }
            ChannelSortMode::Name => {
                entries.sort_by(|a, b| a.name.to_lowercase().cmp(&b.name.to_lowercase()));
            }
        }
        entries.into_iter().map(|c| c.id).collect()
    }
}
