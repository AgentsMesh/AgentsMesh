use napi_derive::napi;
use std::collections::HashMap;

use agentsmesh_state::channel_types::ChannelMessage;
use agentsmesh_types::proto_channel_v1::{
    Channel, ListChannelMembersResponse, ListChannelMessagesResponse, ListChannelPodsResponse,
    ListChannelsResponse,
};
use agentsmesh_types::proto_channel_state_v1::{
    ApplyChannelMessageEditedEventRequest, InsertChannelMessageRequest, InsertChannelRequest,
    PatchChannelMemberCountRequest, ReplaceChannelUnreadCountsRequest,
};
use prost::Message as _;

use crate::AppState;

// Channel state surface over the shared `runtime.state` (the dispatch-hook
// SSOT). The legacy `channel_*` commands operate on the per-service
// `ChannelService` cache; on desktop that cache is disjoint from the realtime
// dispatch target, so the renderer's realtime mirror starves. These `app_*`
// commands read/write the SAME `AppState` the EventBus dispatch mutates, so a
// post-dispatch snapshot read reflects realtime + fetched baseline together.
fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
}

#[napi]
impl AppState {
    // ── Snapshot reads (main pushes these to the renderer after dispatch) ──

    #[napi]
    pub fn app_channels_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().channels.get_channels()).unwrap_or_default()
    }

    #[napi]
    pub fn app_channel_messages_json(&self, channel_id: i64) -> String {
        match self.runtime.state.read().channels.get_messages(channel_id) {
            Some(cache) => serde_json::to_string(&serde_json::json!({
                "messages": cache.messages,
                "has_more": cache.has_more,
            }))
            .unwrap_or_default(),
            None => String::new(),
        }
    }

    #[napi]
    pub fn app_channel_unread_counts_json(&self) -> String {
        serde_json::to_string(&self.runtime.state.read().channels.get_all_unread_counts())
            .unwrap_or_else(|_| "{}".to_string())
    }

    #[napi]
    pub fn app_channel_mention_counts_json(&self) -> String {
        serde_json::to_string(&self.runtime.state.read().channels.get_all_mention_counts())
            .unwrap_or_else(|_| "{}".to_string())
    }

    // ── Fetch→state (desktop main): decode wire response + fold into state,
    //    replacing the renderer's wire-JSON-direct-store path. ──

    #[napi]
    pub fn app_channel_apply_fetched_channels(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let resp = ListChannelsResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().channels.apply_fetched_channels(resp.items);
        Ok(())
    }

    // Single-object fetch (B): decode wire GetChannel response (Channel) + upsert.
    #[napi]
    pub fn app_channel_apply_fetched_channel(&self, resp_bytes: Vec<u8>) -> napi::Result<()> {
        let channel = Channel::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime.state.write().channels.apply_fetched_channel(channel);
        Ok(())
    }

    #[napi]
    pub fn app_channel_apply_fetched_messages(
        &self,
        channel_id: i64,
        resp_bytes: Vec<u8>,
    ) -> napi::Result<()> {
        let resp = ListChannelMessagesResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .channels
            .apply_fetched_messages(channel_id, resp.items, resp.has_more);
        Ok(())
    }

    #[napi]
    pub fn app_channel_apply_fetched_messages_prepend(
        &self,
        channel_id: i64,
        resp_bytes: Vec<u8>,
    ) -> napi::Result<()> {
        let resp = ListChannelMessagesResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .channels
            .apply_fetched_messages_prepend(channel_id, resp.items, resp.has_more);
        Ok(())
    }

    #[napi]
    pub fn app_channel_apply_fetched_members(
        &self,
        channel_id: i64,
        resp_bytes: Vec<u8>,
    ) -> napi::Result<()> {
        let resp = ListChannelMembersResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .channels
            .apply_fetched_members(channel_id, resp.items);
        Ok(())
    }

    #[napi]
    pub fn app_channel_apply_fetched_pods(
        &self,
        channel_id: i64,
        resp_bytes: Vec<u8>,
    ) -> napi::Result<()> {
        let resp = ListChannelPodsResponse::decode(&resp_bytes[..]).map_err(decode_err)?;
        self.runtime
            .state
            .write()
            .channels
            .apply_fetched_pods(channel_id, resp.items);
        Ok(())
    }

    #[napi]
    pub fn app_channel_insert_channel(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = InsertChannelRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(channel) = req.channel {
            let id = channel.id;
            let mut guard = self.runtime.state.write();
            if guard.channels.get_channel(id).is_some() {
                guard.channels.update_channel(id, channel);
            } else {
                guard.channels.add_channel(channel);
            }
        }
        Ok(())
    }

    #[napi]
    pub fn app_channel_insert_message(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = InsertChannelMessageRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        if let Some(msg) = req.message {
            self.runtime
                .state
                .write()
                .channels
                .add_message(req.channel_id, msg);
        }
        Ok(())
    }

    #[napi]
    pub fn app_channel_apply_message_edited(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req =
            ApplyChannelMessageEditedEventRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        let mentions_json = if req.mentions.is_empty() {
            None
        } else {
            serde_json::to_string(&req.mentions).ok()
        };
        let patch = ChannelMessage {
            id: req.message_id,
            channel_id: req.channel_id,
            body: req.body,
            content_json: req.content,
            mentions_json,
            edited_at: Some(req.edited_at),
            ..ChannelMessage::default()
        };
        self.runtime
            .state
            .write()
            .channels
            .update_message(req.channel_id, patch);
        Ok(())
    }

    #[napi]
    pub fn app_channel_replace_unread_counts(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = ReplaceChannelUnreadCountsRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        let counts: HashMap<i64, u32> = req.counts.into_iter().collect();
        self.runtime.state.write().channels.set_unread_counts(counts);
        Ok(())
    }

    #[napi]
    pub fn app_channel_patch_member_count(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = PatchChannelMemberCountRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        let mut guard = self.runtime.state.write();
        if let Some(existing) = guard.channels.get_channel(req.channel_id).cloned() {
            let mut next = existing;
            let curr = next.member_count.unwrap_or(0);
            next.member_count = Some((curr + req.delta as i64).max(0));
            guard.channels.update_channel(req.channel_id, next);
        }
        Ok(())
    }

    #[napi]
    pub fn app_channel_remove_message(&self, channel_id: i64, message_id: i64) {
        self.runtime
            .state
            .write()
            .channels
            .remove_message(channel_id, message_id);
    }

    #[napi]
    pub fn app_channel_clear_unread(&self, channel_id: i64) {
        let mut guard = self.runtime.state.write();
        guard.channels.clear_channel_unread(channel_id);
        guard.channels.clear_channel_mentions(channel_id);
    }

    // ── UI→Rust signals: current user (self-message rule) + active channel
    //    (unread suppression). Without these the SSOT can't compute unread. ──

    #[napi]
    pub fn app_set_current_user(&self, user_id: Option<i64>) {
        self.runtime.state.write().channels.set_current_user_id(user_id);
    }

    #[napi]
    pub fn app_select_channel(&self, id: Option<i64>) {
        self.runtime.state.write().channels.select_channel(id);
    }

    #[napi]
    pub fn app_set_current_channel(&self, id: Option<i64>) {
        self.runtime.state.write().channels.set_current_channel(id);
    }
}
