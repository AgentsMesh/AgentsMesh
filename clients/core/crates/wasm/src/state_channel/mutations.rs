// Fetch→state + channel/message mutations for WasmChannelState. Split from
// the parent module for the file-size limit; a child module so it reaches the
// parent's private `state` field and `decode_err` via `super`.
use agentsmesh_state::channel_state::ChannelSortMode;
use agentsmesh_state::channel_types::ChannelMessage;
use agentsmesh_types::proto_channel_v1::{
    Channel, ListChannelMembersResponse, ListChannelMessagesResponse, ListChannelPodsResponse,
    ListChannelsResponse,
};
use agentsmesh_types::proto_channel_state_v1::{
    ApplyChannelMessageEditedEventRequest, ApplyIncomingChannelMessageRequest,
    InsertChannelMessageRequest, InsertChannelRequest, PatchChannelMemberCountRequest,
};
use prost::Message;
use wasm_bindgen::prelude::*;

use super::{decode_err, WasmChannelState};

#[wasm_bindgen]
impl WasmChannelState {
    // Fetch→state: decode wire ListChannelsResponse + fold into state. Replaces
    // the TS channelFromProto + channelToProtoChannel chain — web fetch now
    // hands raw wire bytes straight to Rust.
    pub fn apply_fetched_channels(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListChannelsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_channels(resp.items);
        Ok(())
    }

    // Fetch→state: decode wire ListChannelMessagesResponse + fold into state.
    // Replaces TS messageFromProto + channelMessageToProto + replace_cached_channel_messages.
    pub fn apply_fetched_messages(&self, channel_id: i64, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListChannelMessagesResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_messages(channel_id, resp.items, resp.has_more);
        Ok(())
    }

    // Fetch→state for pagination load-more (prepend older messages).
    pub fn apply_fetched_messages_prepend(&self, channel_id: i64, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListChannelMessagesResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_messages_prepend(channel_id, resp.items, resp.has_more);
        Ok(())
    }

    // Fetch→state: decode wire ListChannelMembersResponse + fold into state.
    pub fn apply_fetched_members(&self, channel_id: i64, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListChannelMembersResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_members(channel_id, resp.items);
        Ok(())
    }

    // Fetch→state: decode wire ListChannelPodsResponse + fold into state.
    pub fn apply_fetched_pods(&self, channel_id: i64, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListChannelPodsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_pods(channel_id, resp.items);
        Ok(())
    }

    pub fn insert_channel(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertChannelRequest::decode(req_bytes).map_err(decode_err)?;
        let channel = req.channel.ok_or_else(|| JsValue::from_str("missing channel"))?;
        let id = channel.id;
        let mut guard = self.state.write();
        if guard.channels.get_channel(id).is_some() {
            guard.channels.update_channel(id, channel);
        } else {
            guard.channels.add_channel(channel);
        }
        Ok(())
    }

    // Fetch→state (B): decode the wire GetChannel response (proto.channel.v1
    // Channel) + upsert via the wire→state converter — no TS channelFromProto +
    // channelToProtoChannel round-trip.
    pub fn apply_fetched_channel(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let channel = Channel::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().channels.apply_fetched_channel(channel);
        Ok(())
    }

    pub fn remove_channel(&self, id: i64) {
        self.state.write().channels.remove_channel(id);
    }

    pub fn patch_channel_member_count(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchChannelMemberCountRequest::decode(req_bytes).map_err(decode_err)?;
        let mut guard = self.state.write();
        if let Some(existing) = guard.channels.get_channel(req.channel_id).cloned() {
            let mut next = existing;
            let curr = next.member_count.unwrap_or(0);
            let new = (curr + req.delta as i64).max(0);
            next.member_count = Some(new);
            guard.channels.update_channel(req.channel_id, next);
        }
        Ok(())
    }

    pub fn filter_channels_json(&self, query: &str, include_archived: bool) -> String {
        let guard = self.state.read();
        let filtered = guard.channels.filter_channels(query, include_archived);
        serde_json::to_string(&filtered).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn select_channel(&self, id: Option<i64>) -> JsValue {
        let mut guard = self.state.write();
        match guard.channels.select_channel(id) {
            Some(c) => JsValue::from_str(
                &serde_json::to_string(c).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn sorted_channel_ids_json(&self, mode: &str, include_archived: bool) -> String {
        let sort_mode = match mode {
            "unread_first" => ChannelSortMode::UnreadFirst,
            "name" => ChannelSortMode::Name,
            _ => ChannelSortMode::LastMessage,
        };
        let ids = self.state.read().channels.sorted_channel_ids(sort_mode, include_archived);
        serde_json::to_string(&ids).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn get_last_message_json(&self, channel_id: i64) -> JsValue {
        match self.state.read().channels.get_last_message(channel_id) {
            Some(preview) => JsValue::from_str(
                &serde_json::to_string(preview).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn apply_incoming_channel_message(&self, req_bytes: &[u8]) -> Result<bool, JsValue> {
        let req = ApplyIncomingChannelMessageRequest::decode(req_bytes).map_err(decode_err)?;
        let msg = req.message.ok_or_else(|| JsValue::from_str("missing message"))?;
        Ok(self.state.write().channels.on_new_message(msg))
    }

    pub fn insert_channel_message(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertChannelMessageRequest::decode(req_bytes).map_err(decode_err)?;
        let msg = req.message.ok_or_else(|| JsValue::from_str("missing message"))?;
        self.state.write().channels.add_message(req.channel_id, msg);
        Ok(())
    }

    pub fn apply_channel_message_edited_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyChannelMessageEditedEventRequest::decode(req_bytes).map_err(decode_err)?;
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
        self.state.write().channels.update_message(req.channel_id, patch);
        Ok(())
    }

    pub fn remove_message(&self, channel_id: i64, message_id: i64) {
        self.state.write().channels.remove_message(channel_id, message_id);
    }

    pub fn get_messages_json(&self, channel_id: i64) -> JsValue {
        match self.state.read().channels.get_messages(channel_id) {
            Some(cache) => {
                let val = serde_json::json!({
                    "messages": cache.messages,
                    "has_more": cache.has_more,
                });
                JsValue::from_str(
                    &serde_json::to_string(&val).unwrap_or_default(),
                )
            }
            None => JsValue::NULL,
        }
    }
}
