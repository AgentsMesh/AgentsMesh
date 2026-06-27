use std::collections::HashMap;

use super::ChannelState;

// Unread + mention counters live inline on Channel.unread_count /
// Channel.mention_count (proto fields tag 100/101). All counter APIs
// read/write those fields; no side-channel HashMap remains. Contract:
// callers must `set_channels` first — counters on a non-loaded channel
// are silently dropped (it's display state, the backend is the SSOT).

impl ChannelState {
    pub fn set_unread_counts(&mut self, counts: HashMap<i64, u32>) {
        tracing::debug!(target: "channel", count = counts.len(), "set unread counts (baseline)");
        for c in self.channels.iter_mut() {
            c.unread_count = counts.get(&c.id).copied().unwrap_or(0);
        }
        self.sync_current_channel_counters();
    }

    pub fn increment_unread(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "increment unread");
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            c.unread_count = c.unread_count.saturating_add(1);
        }
        self.sync_current_channel_counters();
    }

    pub fn clear_channel_unread(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "clear unread");
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            c.unread_count = 0;
            c.manually_unread = false; // opening/reading dismisses a "mark unread"
        }
        self.sync_current_channel_counters();
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 {
        self.channels
            .iter()
            .find(|c| c.id == channel_id)
            .map(|c| c.unread_count)
            .unwrap_or(0)
    }

    pub fn total_unread_count(&self) -> u32 {
        self.channels.iter().map(|c| c.unread_count).sum()
    }

    pub fn get_all_unread_counts(&self) -> HashMap<i64, u32> {
        self.channels.iter().map(|c| (c.id, c.unread_count)).collect()
    }

    pub fn set_mention_counts(&mut self, counts: HashMap<i64, u32>) {
        tracing::debug!(target: "channel", count = counts.len(), "set mention counts (baseline)");
        for c in self.channels.iter_mut() {
            c.mention_count = counts.get(&c.id).copied().unwrap_or(0);
        }
        self.sync_current_channel_counters();
    }

    pub fn increment_mention(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "increment mention");
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            c.mention_count = c.mention_count.saturating_add(1);
        }
        self.sync_current_channel_counters();
    }

    pub fn clear_channel_mentions(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "clear mentions");
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            c.mention_count = 0;
        }
        self.sync_current_channel_counters();
    }

    pub fn get_mention_count(&self, channel_id: i64) -> u32 {
        self.channels
            .iter()
            .find(|c| c.id == channel_id)
            .map(|c| c.mention_count)
            .unwrap_or(0)
    }

    pub fn total_mention_count(&self) -> u32 {
        self.channels.iter().map(|c| c.mention_count).sum()
    }

    pub fn get_all_mention_counts(&self) -> HashMap<i64, u32> {
        self.channels.iter().map(|c| (c.id, c.mention_count)).collect()
    }

    /// Set per-channel last-read cursors from a backend summary fetch. Sparse
    /// (only channels in the map are touched) and monotonic: it establishes the
    /// baseline when unknown but never rewinds a cursor a local advance_last_read
    /// already moved past, so a stale in-flight fetch can't re-show the divider
    /// over messages the user already read.
    pub fn set_last_read(&mut self, last_read: HashMap<i64, i64>) {
        for c in self.channels.iter_mut() {
            if let Some(&lr) = last_read.get(&c.id) {
                if c.last_read_message_id.map_or(true, |cur| lr > cur) {
                    c.last_read_message_id = Some(lr);
                }
            }
        }
        self.sync_current_channel_counters();
    }

    pub fn get_last_read(&self, channel_id: i64) -> Option<i64> {
        self.channels
            .iter()
            .find(|c| c.id == channel_id)
            .and_then(|c| c.last_read_message_id)
    }

    /// Advance the local read cursor when the user reads a channel (mirrors the
    /// backend MarkRead). Monotonic — keeps the unread divider correct on
    /// re-entry, since `set_last_read` only refreshes channels with unread > 0.
    pub fn advance_last_read(&mut self, channel_id: i64, message_id: i64) {
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            if message_id > c.last_read_message_id.unwrap_or(0) {
                c.last_read_message_id = Some(message_id);
            }
        }
        self.sync_current_channel_counters();
    }

    pub fn set_manually_unread(&mut self, flags: HashMap<i64, bool>) {
        for c in self.channels.iter_mut() {
            c.manually_unread = flags.get(&c.id).copied().unwrap_or(false);
        }
        self.sync_current_channel_counters();
    }

    pub fn get_manually_unread(&self, channel_id: i64) -> bool {
        self.channels
            .iter()
            .find(|c| c.id == channel_id)
            .map(|c| c.manually_unread)
            .unwrap_or(false)
    }

    pub fn get_all_manually_unread(&self) -> HashMap<i64, bool> {
        self.channels.iter().filter(|c| c.manually_unread).map(|c| (c.id, true)).collect()
    }

    /// Mirror current channel's counters from the main list. Without this,
    /// `get_current_channel()` would return a snapshot frozen at select time
    /// while the sidebar's badge counts kept ticking.
    fn sync_current_channel_counters(&mut self) {
        let Some(curr_id) = self.current_channel.as_ref().map(|c| c.id) else { return };
        let snap = self
            .channels
            .iter()
            .find(|c| c.id == curr_id)
            .map(|c| (c.unread_count, c.mention_count, c.last_read_message_id, c.manually_unread));
        if let (Some((u, m, lr, mu)), Some(c)) = (snap, self.current_channel.as_mut()) {
            c.unread_count = u;
            c.mention_count = m;
            c.last_read_message_id = lr;
            c.manually_unread = mu;
        }
    }
}
