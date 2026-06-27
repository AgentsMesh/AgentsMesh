use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_channel_state_v1::{
    InsertChannelRequest, ReplaceCachedChannelMessagesRequest, ReplaceCachedChannelsRequest,
    ReplaceChannelMembersRequest, ReplaceChannelPodsRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

mod mutations;
mod readstate;

/// View into `AppState.channels` exposed to JavaScript. See `state_pod.rs`
/// for the shared-state pattern rationale.
#[wasm_bindgen]
pub struct WasmChannelState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmChannelState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmChannelState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::with_storage(crate::new_memory_backend()))),
        }
    }

    pub fn set_current_user_id(&self, user_id: Option<i64>) {
        self.state.write().channels.set_current_user_id(user_id);
    }

    pub fn channels_json(&self) -> String {
        serde_json::to_string(self.state.read().channels.get_channels()).unwrap_or_default()
    }

    // Read side, zero-JSON: prost-encode state channels (reuse the list wrapper).
    // Web/desktop decode via fromBinary + channelToCache. Replaces channels_json.
    pub fn channels_bytes(&self) -> Vec<u8> {
        let channels = self.state.read().channels.get_channels().to_vec();
        ReplaceCachedChannelsRequest { channels }.encode_to_vec()
    }

    // Read side, zero-JSON for the remaining facets. Reuse the *Request wrappers
    // (web/desktop decode via fromBinary + the same xToCache as the mutators).
    pub fn get_messages_bytes(&self, channel_id: i64) -> Vec<u8> {
        match self.state.read().channels.get_messages(channel_id) {
            Some(cache) => ReplaceCachedChannelMessagesRequest {
                channel_id,
                messages: cache.messages.clone(),
                has_more: cache.has_more,
            }
            .encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn channel_members_bytes(&self, channel_id: i64) -> Vec<u8> {
        let members = self.state.read().channels.get_channel_members(channel_id);
        ReplaceChannelMembersRequest { channel_id, members }.encode_to_vec()
    }

    pub fn channel_pods_bytes(&self, channel_id: i64) -> Vec<u8> {
        let pods = self.state.read().channels.get_channel_pods(channel_id);
        ReplaceChannelPodsRequest { channel_id, pods }.encode_to_vec()
    }

    // Single-channel + preview read side (zero-JSON). Single channel reuses the
    // InsertChannelRequest wrapper; preview encodes MessagePreview directly.
    pub fn get_channel_bytes(&self, id: i64) -> Vec<u8> {
        match self.state.read().channels.get_channel(id) {
            Some(c) => InsertChannelRequest { channel: Some(c.clone()) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn current_channel_bytes(&self) -> Vec<u8> {
        match self.state.read().channels.get_current_channel() {
            Some(c) => InsertChannelRequest { channel: Some(c.clone()) }.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn get_last_message_bytes(&self, channel_id: i64) -> Vec<u8> {
        match self.state.read().channels.get_last_message(channel_id) {
            Some(preview) => preview.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn current_channel_json(&self) -> JsValue {
        match self.state.read().channels.get_current_channel() {
            Some(c) => JsValue::from_str(
                &serde_json::to_string(c).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn set_current_channel(&self, id: Option<i64>) {
        self.state.write().channels.set_current_channel(id);
    }

    pub fn get_channel_json(&self, id: i64) -> JsValue {
        match self.state.read().channels.get_channel(id) {
            Some(c) => JsValue::from_str(
                &serde_json::to_string(c).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

}
