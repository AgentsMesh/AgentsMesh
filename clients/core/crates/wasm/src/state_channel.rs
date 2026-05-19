use std::collections::HashMap;

use agentsmesh_state::channel_state::{ChannelSortMode, ChannelState};
use agentsmesh_types::{Channel, ChannelMessage, MessagePreview, User};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmChannelState {
    inner: ChannelState,
}

#[wasm_bindgen]
impl WasmChannelState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self { inner: ChannelState::with_storage(crate::new_memory_backend()) }
    }

    pub fn set_current_user_id(&mut self, user_id: Option<i64>) {
        self.inner.set_current_user_id(user_id);
    }

    pub fn set_current_user(&mut self, user_json: &str) {
        match serde_json::from_str::<User>(user_json) {
            Ok(user) => self.inner.set_current_user(Some(user)),
            Err(_) => self.inner.set_current_user(None),
        }
    }

    pub fn channels_json(&self) -> String {
        serde_json::to_string(self.inner.get_channels()).unwrap_or_default()
    }

    pub fn current_channel_json(&self) -> JsValue {
        match self.inner.get_current_channel() {
            Some(c) => JsValue::from_str(
                &serde_json::to_string(c).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn set_channels(&mut self, json: &str) {
        if let Ok(channels) = serde_json::from_str::<Vec<Channel>>(json) {
            self.inner.set_channels(channels);
        }
    }

    pub fn set_current_channel(&mut self, id: Option<i64>) {
        self.inner.set_current_channel(id);
    }

    pub fn get_channel_json(&self, id: i64) -> JsValue {
        match self.inner.get_channel(id) {
            Some(c) => JsValue::from_str(
                &serde_json::to_string(c).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn add_channel(&mut self, json: &str) {
        if let Ok(channel) = serde_json::from_str::<Channel>(json) {
            self.inner.add_channel(channel);
        }
    }

    pub fn update_channel(&mut self, id: i64, json: &str) {
        if let Ok(channel) = serde_json::from_str::<Channel>(json) {
            self.inner.update_channel(id, channel);
        }
    }

    pub fn remove_channel(&mut self, id: i64) {
        self.inner.remove_channel(id);
    }

    pub fn filter_channels_json(&self, query: &str, include_archived: bool) -> String {
        let filtered = self.inner.filter_channels(query, include_archived);
        serde_json::to_string(&filtered).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn select_channel(&mut self, id: Option<i64>) -> JsValue {
        match self.inner.select_channel(id) {
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
        let ids = self.inner.sorted_channel_ids(sort_mode, include_archived);
        serde_json::to_string(&ids).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn get_last_message_json(&self, channel_id: i64) -> JsValue {
        match self.inner.get_last_message(channel_id) {
            Some(preview) => JsValue::from_str(
                &serde_json::to_string(preview).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn set_last_message(&mut self, channel_id: i64, preview_json: &str) {
        if let Ok(preview) = serde_json::from_str::<MessagePreview>(preview_json) {
            self.inner.set_last_message(channel_id, preview);
        }
    }

    pub fn add_message(&mut self, channel_id: i64, message_json: &str) {
        if let Ok(msg) = serde_json::from_str::<ChannelMessage>(message_json) {
            self.inner.add_message(channel_id, msg);
        }
    }

    pub fn on_new_message(&mut self, message_json: &str) -> bool {
        match serde_json::from_str::<ChannelMessage>(message_json) {
            Ok(msg) => self.inner.on_new_message(msg),
            Err(_) => false,
        }
    }

    pub fn update_message(&mut self, channel_id: i64, message_json: &str) {
        if let Ok(msg) = serde_json::from_str::<ChannelMessage>(message_json) {
            self.inner.update_message(channel_id, msg);
        }
    }

    pub fn remove_message(&mut self, channel_id: i64, message_id: i64) {
        self.inner.remove_message(channel_id, message_id);
    }

    pub fn get_messages_json(&self, channel_id: i64) -> JsValue {
        match self.inner.get_messages(channel_id) {
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

    pub fn set_messages(
        &mut self,
        channel_id: i64,
        messages_json: &str,
        has_more: bool,
    ) {
        if let Ok(messages) =
            serde_json::from_str::<Vec<ChannelMessage>>(messages_json)
        {
            self.inner.set_messages(channel_id, messages, has_more);
        }
    }

    pub fn prepend_messages(
        &mut self,
        channel_id: i64,
        messages_json: &str,
        has_more: bool,
    ) {
        if let Ok(messages) =
            serde_json::from_str::<Vec<ChannelMessage>>(messages_json)
        {
            self.inner.prepend_messages(channel_id, messages, has_more);
        }
    }

    pub fn set_unread_counts(&mut self, json: &str) {
        if let Ok(counts) = serde_json::from_str::<HashMap<i64, u32>>(json) {
            self.inner.set_unread_counts(counts);
        }
    }

    pub fn increment_unread(&mut self, channel_id: i64) {
        self.inner.increment_unread(channel_id);
    }

    pub fn clear_channel_unread(&mut self, channel_id: i64) {
        self.inner.clear_channel_unread(channel_id);
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 {
        self.inner.get_unread_count(channel_id)
    }

    pub fn total_unread_count(&self) -> u32 {
        self.inner.total_unread_count()
    }

    pub fn unread_counts_json(&self) -> String {
        serde_json::to_string(self.inner.get_all_unread_counts()).unwrap_or_else(|_| "{}".to_string())
    }

    pub fn increment_mention(&mut self, channel_id: i64) {
        self.inner.increment_mention(channel_id);
    }

    pub fn clear_channel_mentions(&mut self, channel_id: i64) {
        self.inner.clear_channel_mentions(channel_id);
    }

    pub fn get_mention_count(&self, channel_id: i64) -> u32 {
        self.inner.get_mention_count(channel_id)
    }

    pub fn total_mention_count(&self) -> u32 {
        self.inner.total_mention_count()
    }

    pub fn set_mention_counts(&mut self, json: &str) {
        if let Ok(counts) = serde_json::from_str::<HashMap<i64, u32>>(json) {
            self.inner.set_mention_counts(counts);
        }
    }

    pub fn mention_counts_json(&self) -> String {
        serde_json::to_string(self.inner.get_all_mention_counts()).unwrap_or_else(|_| "{}".to_string())
    }
}
