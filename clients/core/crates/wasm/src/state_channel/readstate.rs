// Unread / mention / last-read counters + member/pod replace for
// WasmChannelState. Split from the parent module for the file-size limit.
use std::collections::HashMap;

use agentsmesh_types::proto_channel_state_v1::{
    RemoveChannelMemberRequest, ReplaceChannelMembersRequest, ReplaceChannelPodsRequest,
    ReplaceChannelUnreadCountsRequest,
};
use prost::Message;
use wasm_bindgen::prelude::*;

use super::{decode_err, WasmChannelState};

#[wasm_bindgen]
impl WasmChannelState {
    pub fn replace_channel_unread_counts(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceChannelUnreadCountsRequest::decode(req_bytes).map_err(decode_err)?;
        let counts: HashMap<i64, u32> = req.counts.into_iter().collect();
        let mentions: HashMap<i64, u32> = req.mentions.into_iter().collect();
        let last_read: HashMap<i64, i64> = req.last_read.into_iter().collect();
        let manually_unread: HashMap<i64, bool> = req.manually_unread.into_iter().collect();
        let mut guard = self.state.write();
        guard.channels.set_unread_counts(counts);
        guard.channels.set_mention_counts(mentions);
        guard.channels.set_last_read(last_read);
        guard.channels.set_manually_unread(manually_unread);
        Ok(())
    }

    /// Last-read message id for a channel (0 = nothing read). Returned as f64
    /// to stay a plain JS number — message ids are well within safe-integer
    /// range. The UI snapshots this on entry to anchor the unread divider.
    // -1 = no known cursor (distinct from a genuine 0 cursor "read nothing yet").
    // The divider hook needs the two apart: 0 anchors at the first message, -1
    // suppresses the divider for a channel the summary fetch never reported.
    pub fn get_last_read_id(&self, channel_id: i64) -> f64 {
        self.state.read().channels.get_last_read(channel_id).unwrap_or(-1) as f64
    }

    pub fn advance_last_read(&self, channel_id: i64, message_id: i64) {
        self.state.write().channels.advance_last_read(channel_id, message_id);
    }

    pub fn increment_unread(&self, channel_id: i64) {
        self.state.write().channels.increment_unread(channel_id);
    }

    pub fn clear_channel_unread(&self, channel_id: i64) {
        self.state.write().channels.clear_channel_unread(channel_id);
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 {
        self.state.read().channels.get_unread_count(channel_id)
    }

    pub fn total_unread_count(&self) -> u32 {
        self.state.read().channels.total_unread_count()
    }

    pub fn unread_counts_json(&self) -> String {
        let counts = self.state.read().channels.get_all_unread_counts();
        serde_json::to_string(&counts).unwrap_or_else(|_| "{}".to_string())
    }

    // Read side (B, zero-JSON): encode the unread map into the same wrapper the
    // mutator decodes, so the selector reads via fromBinary, not serde JSON.
    pub fn unread_counts_bytes(&self) -> Vec<u8> {
        let guard = self.state.read();
        let counts = guard.channels.get_all_unread_counts();
        let manually_unread = guard.channels.get_all_manually_unread();
        // mentions/last_read have their own selectors; manually_unread rides
        // along here so the sidebar badge reads it on the same unread tick.
        ReplaceChannelUnreadCountsRequest { counts, manually_unread, ..Default::default() }.encode_to_vec()
    }

    pub fn increment_mention(&self, channel_id: i64) {
        self.state.write().channels.increment_mention(channel_id);
    }

    pub fn clear_channel_mentions(&self, channel_id: i64) {
        self.state.write().channels.clear_channel_mentions(channel_id);
    }

    pub fn get_mention_count(&self, channel_id: i64) -> u32 {
        self.state.read().channels.get_mention_count(channel_id)
    }

    pub fn total_mention_count(&self) -> u32 {
        self.state.read().channels.total_mention_count()
    }

    pub fn mention_counts_json(&self) -> String {
        let counts = self.state.read().channels.get_all_mention_counts();
        serde_json::to_string(&counts).unwrap_or_else(|_| "{}".to_string())
    }

    pub fn channel_members_json(&self, channel_id: i64) -> String {
        let members = self.state.read().channels.get_channel_members(channel_id);
        serde_json::to_string(&members).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn channel_pods_json(&self, channel_id: i64) -> String {
        let pods = self.state.read().channels.get_channel_pods(channel_id);
        serde_json::to_string(&pods).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn replace_channel_pods(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceChannelPodsRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().channels.set_channel_pods(req.channel_id, req.pods);
        Ok(())
    }

    pub fn replace_channel_members(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceChannelMembersRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().channels.set_channel_members(req.channel_id, req.members);
        Ok(())
    }

    pub fn remove_channel_member(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = RemoveChannelMemberRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().channels.remove_channel_member(req.channel_id, req.user_id);
        Ok(())
    }
}
