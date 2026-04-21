use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::ChannelService;
use agentsmesh_state::channel_state::ChannelState;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmChannelService(pub(crate) ChannelService);

#[wasm_bindgen]
impl WasmChannelService {
    pub(crate) fn new(client: Arc<ApiClient>, state: ChannelState) -> Self {
        Self(ChannelService::new(client, state))
    }

    pub fn channels_json(&self) -> String { self.0.channels_json() }

    pub fn current_channel_json(&self) -> JsValue {
        match self.0.current_channel_json() {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn get_channel_json(&self, id: i64) -> JsValue {
        match self.0.get_channel_json(id) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn filter_channels_json(&self, query: &str, include_archived: bool) -> String {
        self.0.filter_channels_json(query, include_archived)
    }

    pub fn get_messages_json(&self, channel_id: i64) -> JsValue {
        match self.0.get_messages_json(channel_id) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 { self.0.get_unread_count(channel_id) }
    pub fn total_unread_count(&self) -> u32 { self.0.total_unread_count() }
    pub fn unread_counts_json(&self) -> String { self.0.unread_counts_json() }
    pub fn get_mention_count(&self, channel_id: i64) -> u32 { self.0.get_mention_count(channel_id) }
    pub fn total_mention_count(&self) -> u32 { self.0.total_mention_count() }
    pub fn mention_counts_json(&self) -> String { self.0.mention_counts_json() }

    pub fn sorted_channel_ids_json(&self, mode: &str, include_archived: bool) -> String {
        self.0.sorted_channel_ids_json(mode, include_archived)
    }

    pub fn get_last_message_json(&self, channel_id: i64) -> JsValue {
        match self.0.get_last_message_json(channel_id) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn set_channels(&self, json: &str) { self.0.set_channels(json); }
    pub fn set_current_channel(&self, id: Option<i64>) { self.0.set_current_channel(id); }

    pub fn select_channel(&self, id: Option<i64>) -> JsValue {
        match self.0.select_channel(id) {
            Some(s) => JsValue::from_str(&s),
            None => JsValue::NULL,
        }
    }

    pub fn add_channel_local(&self, json: &str) { self.0.add_channel_local(json); }

    pub fn update_channel_local(&self, id: i64, json: &str) {
        self.0.update_channel_local(id, json);
    }

    pub fn remove_channel_local(&self, id: i64) { self.0.remove_channel_local(id); }

    pub fn set_current_user(&self, user_json: &str) { self.0.set_current_user(user_json); }
    pub fn set_current_user_id(&self, user_id: Option<i64>) { self.0.set_current_user_id(user_id); }

    pub fn set_messages(&self, channel_id: i64, json: &str, has_more: bool) {
        self.0.set_messages(channel_id, json, has_more);
    }

    pub fn prepend_messages(&self, channel_id: i64, json: &str, has_more: bool) {
        self.0.prepend_messages(channel_id, json, has_more);
    }

    pub fn add_message(&self, channel_id: i64, json: &str) { self.0.add_message(channel_id, json); }
    pub fn on_new_message(&self, json: &str) -> bool { self.0.on_new_message(json) }

    pub fn update_message_local(&self, channel_id: i64, json: &str) {
        self.0.update_message_local(channel_id, json);
    }

    pub fn remove_message_local(&self, channel_id: i64, message_id: i64) {
        self.0.remove_message_local(channel_id, message_id);
    }

    pub fn set_unread_counts(&self, json: &str) { self.0.set_unread_counts(json); }
    pub fn increment_unread(&self, channel_id: i64) { self.0.increment_unread(channel_id); }
    pub fn clear_channel_unread(&self, channel_id: i64) { self.0.clear_channel_unread(channel_id); }
    pub fn set_mention_counts(&self, json: &str) { self.0.set_mention_counts(json); }
    pub fn increment_mention(&self, channel_id: i64) { self.0.increment_mention(channel_id); }

    pub fn clear_channel_mentions(&self, channel_id: i64) {
        self.0.clear_channel_mentions(channel_id);
    }

    pub fn set_last_message(&self, channel_id: i64, json: &str) {
        self.0.set_last_message(channel_id, json);
    }

    pub async fn fetch_channels(&self, include_archived: Option<bool>) -> Result<String, String> {
        self.0.fetch_channels(include_archived).await
    }

    pub async fn fetch_channel(&self, id: i64) -> Result<String, String> {
        self.0.fetch_channel(id).await
    }

    pub async fn create_channel(&self, request_json: &str) -> Result<String, String> {
        self.0.create_channel(request_json).await
    }

    pub async fn update_channel(&self, id: i64, request_json: &str) -> Result<String, String> {
        self.0.update_channel(id, request_json).await
    }

    pub async fn archive_channel(&self, id: i64) -> Result<(), String> {
        self.0.archive_channel(id).await
    }

    pub async fn unarchive_channel(&self, id: i64) -> Result<(), String> {
        self.0.unarchive_channel(id).await
    }

    pub async fn join_channel(&self, channel_id: i64, pod_key: &str) -> Result<String, String> {
        self.0.join_channel(channel_id, pod_key).await
    }

    pub async fn leave_channel(&self, channel_id: i64, pod_key: &str) -> Result<String, String> {
        self.0.leave_channel(channel_id, pod_key).await
    }

    pub async fn fetch_messages(
        &self, channel_id: i64, limit: Option<u32>, before_id: Option<i64>,
    ) -> Result<String, String> {
        self.0.fetch_messages(channel_id, limit, before_id).await
    }

    pub async fn send_message(&self, channel_id: i64, request_json: &str) -> Result<String, String> {
        self.0.send_message(channel_id, request_json).await
    }

    pub async fn edit_message(
        &self, channel_id: i64, message_id: i64, content: &str,
    ) -> Result<String, String> {
        self.0.edit_message(channel_id, message_id, content).await
    }

    pub async fn delete_message(&self, channel_id: i64, message_id: i64) -> Result<(), String> {
        self.0.delete_message(channel_id, message_id).await
    }

    pub async fn fetch_unread_counts(&self) -> Result<String, String> {
        self.0.fetch_unread_counts().await
    }

    pub async fn mark_read(&self, channel_id: i64, message_id: i64) -> Result<(), String> {
        self.0.mark_read(channel_id, message_id).await
    }

    pub async fn mute_channel(&self, channel_id: i64, muted: bool) -> Result<(), String> {
        self.0.mute_channel(channel_id, muted).await
    }

    pub async fn get_channel_pods(&self, id: i64) -> Result<String, String> {
        self.0.get_channel_pods(id).await
    }

    pub fn channel_pods_json(&self, id: i64) -> String {
        self.0.channel_pods_json(id)
    }

    pub async fn fetch_channel_members(&self, id: i64) -> Result<String, String> {
        self.0.fetch_channel_members(id).await
    }

    pub async fn invite_channel_members(&self, id: i64, user_ids_json: String) -> Result<(), String> {
        self.0.invite_channel_members(id, &user_ids_json).await
    }

    pub async fn remove_channel_member(&self, id: i64, user_id: i64) -> Result<(), String> {
        self.0.remove_channel_member(id, user_id).await
    }

    pub fn channel_members_json(&self, id: i64) -> String {
        self.0.channel_members_json(id)
    }

    pub async fn search_channel_messages(&self, id: i64, q: &str, limit: Option<u32>) -> Result<String, String> {
        self.0.search_channel_messages(id, q, limit).await
    }
}
