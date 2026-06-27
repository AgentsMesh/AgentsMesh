// Fetch→state + insert/edit mutations for the desktop main-process channel
// state. Split from app_channel.rs for the file-size limit; operates on the
// same `AppState` SSOT the EventBus dispatch mutates (see app_channel.rs).
use napi_derive::napi;
use std::collections::HashMap;

use agentsmesh_state::channel_types::ChannelMessage;
use agentsmesh_types::proto_channel_v1::{
    Channel, ListChannelMembersResponse, ListChannelMessagesResponse, ListChannelPodsResponse,
    ListChannelsResponse,
};
use agentsmesh_types::proto_channel_state_v1::{
    ApplyChannelMessageEditedEventRequest, InsertChannelMessageRequest, InsertChannelRequest,
    ReplaceChannelUnreadCountsRequest,
};
use prost::Message as _;

use crate::AppState;

fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
}

#[napi]
impl AppState {
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
        let mentions: HashMap<i64, u32> = req.mentions.into_iter().collect();
        let last_read: HashMap<i64, i64> = req.last_read.into_iter().collect();
        let manually_unread: HashMap<i64, bool> = req.manually_unread.into_iter().collect();
        let mut guard = self.runtime.state.write();
        guard.channels.set_unread_counts(counts);
        guard.channels.set_mention_counts(mentions);
        guard.channels.set_last_read(last_read);
        guard.channels.set_manually_unread(manually_unread);
        Ok(())
    }
}
