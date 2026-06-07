use super::ChannelState;
use agentsmesh_types::proto_pod_v1::Pod;
use crate::channel_types::ChannelMember;

impl ChannelState {
    pub fn set_channel_members(&mut self, channel_id: i64, members: Vec<ChannelMember>) {
        tracing::debug!(target: "channel", channel_id, count = members.len(), "set channel members");
        self.members_by_channel.insert(channel_id, members);
    }

    pub fn get_channel_members(&self, channel_id: i64) -> Vec<ChannelMember> {
        self.members_by_channel.get(&channel_id).cloned().unwrap_or_default()
    }

    pub fn remove_channel_member(&mut self, channel_id: i64, user_id: i64) {
        tracing::info!(target: "channel", channel_id, user_id, "remove channel member");
        if let Some(list) = self.members_by_channel.get_mut(&channel_id) {
            list.retain(|m| m.user_id != user_id);
        }
    }

    pub fn clear_channel_members(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "clear channel members");
        self.members_by_channel.remove(&channel_id);
    }

    pub fn set_channel_pods(&mut self, channel_id: i64, pods: Vec<Pod>) {
        tracing::debug!(target: "channel", channel_id, count = pods.len(), "set channel pods");
        self.pods_by_channel.insert(channel_id, pods);
    }

    pub fn get_channel_pods(&self, channel_id: i64) -> Vec<Pod> {
        self.pods_by_channel.get(&channel_id).cloned().unwrap_or_default()
    }

    pub fn clear_channel_pods(&mut self, channel_id: i64) {
        tracing::debug!(target: "channel", channel_id, "clear channel pods");
        self.pods_by_channel.remove(&channel_id);
    }

    /// Mutate `channel.member_count` by `delta` (clamped at 0). Used by
    /// the realtime `channel:member_added` / `:member_removed` dispatch
    /// arms — the event itself doesn't carry the new count, only the
    /// delta. Returns true if the channel exists and was updated.
    pub fn patch_member_count(&mut self, channel_id: i64, delta: i32) -> bool {
        tracing::info!(target: "channel", channel_id, delta, "patch member count");
        if let Some(existing) = self.get_channel(channel_id).cloned() {
            let mut next = existing;
            let curr = next.member_count.unwrap_or(0);
            next.member_count = Some((curr + delta as i64).max(0));
            self.update_channel(channel_id, next);
            true
        } else {
            false
        }
    }
}
