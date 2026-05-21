use std::sync::Arc;
use std::sync::RwLock;

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::channel_state::ChannelState;
use agentsmesh_types::{
    Channel, ChannelMessage,
    CreateChannelRequest, UpdateChannelRequest,
    SendChannelMessageRequest, EditChannelMessageRequest,
    JoinChannelPodRequest, MuteChannelRequest,
};

pub struct ChannelService {
    client: Arc<ApiClient>,
    state: RwLock<ChannelState>,
}

impl ChannelService {
    pub fn new(client: Arc<ApiClient>, state: ChannelState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub fn channels_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_channels()).unwrap_or_default()
    }

    pub fn current_channel_json(&self) -> Option<String> {
        self.state.read().unwrap().get_current_channel()
            .map(|c| serde_json::to_string(c).unwrap_or_default())
    }

    pub fn get_channel_json(&self, id: i64) -> Option<String> {
        self.state.read().unwrap().get_channel(id)
            .map(|c| serde_json::to_string(c).unwrap_or_default())
    }

    pub fn filter_channels_json(&self, query: &str, include_archived: bool) -> String {
        let binding = self.state.read().unwrap();
        let filtered = binding.filter_channels(query, include_archived);
        serde_json::to_string(&filtered).unwrap_or_default()
    }

    pub fn get_messages_json(&self, channel_id: i64) -> Option<String> {
        self.state.read().unwrap().get_messages(channel_id).map(|cache| {
            let result = serde_json::json!({
                "messages": cache.messages,
                "has_more": cache.has_more,
            });
            serde_json::to_string(&result).unwrap_or_default()
        })
    }

    pub fn get_unread_count(&self, channel_id: i64) -> u32 {
        self.state.read().unwrap().get_unread_count(channel_id)
    }

    pub fn total_unread_count(&self) -> u32 {
        self.state.read().unwrap().total_unread_count()
    }

    pub fn unread_counts_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_all_unread_counts()).unwrap_or_default()
    }

    pub fn get_mention_count(&self, channel_id: i64) -> u32 {
        self.state.read().unwrap().get_mention_count(channel_id)
    }

    pub fn total_mention_count(&self) -> u32 {
        self.state.read().unwrap().total_mention_count()
    }

    pub fn mention_counts_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_all_mention_counts()).unwrap_or_default()
    }

    pub fn sorted_channel_ids_json(&self, mode: &str, include_archived: bool) -> String {
        use agentsmesh_state::channel_state::ChannelSortMode;
        let sort_mode = match mode {
            "unread_first" => ChannelSortMode::UnreadFirst,
            "name" => ChannelSortMode::Name,
            _ => ChannelSortMode::LastMessage,
        };
        let ids = self.state.read().unwrap().sorted_channel_ids(sort_mode, include_archived);
        serde_json::to_string(&ids).unwrap_or_default()
    }

    pub fn get_last_message_json(&self, channel_id: i64) -> Option<String> {
        self.state.read().unwrap().get_last_message(channel_id)
            .map(|p| serde_json::to_string(p).unwrap_or_default())
    }

    pub fn set_channels(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<Channel>>(json) {
            self.state.write().unwrap().set_channels(v);
        }
    }

    pub fn set_current_channel(&self, id: Option<i64>) {
        self.state.write().unwrap().set_current_channel(id);
    }

    pub fn select_channel(&self, id: Option<i64>) -> Option<String> {
        self.state.write().unwrap().select_channel(id)
            .map(|c| serde_json::to_string(c).unwrap_or_default())
    }

    pub fn add_channel_local(&self, json: &str) {
        if let Ok(c) = serde_json::from_str::<Channel>(json) {
            self.state.write().unwrap().add_channel(c);
        }
    }

    pub fn update_channel_local(&self, id: i64, json: &str) {
        if let Ok(c) = serde_json::from_str::<Channel>(json) {
            self.state.write().unwrap().update_channel(id, c);
        }
    }

    pub fn remove_channel_local(&self, id: i64) {
        self.state.write().unwrap().remove_channel(id);
    }

    pub fn set_current_user(&self, user_json: &str) {
        if let Ok(u) = serde_json::from_str(user_json) {
            self.state.write().unwrap().set_current_user(Some(u));
        }
    }

    pub fn set_current_user_id(&self, user_id: Option<i64>) {
        self.state.write().unwrap().set_current_user_id(user_id);
    }

    pub fn set_messages(&self, channel_id: i64, json: &str, has_more: bool) {
        if let Ok(msgs) = serde_json::from_str::<Vec<ChannelMessage>>(json) {
            self.state.write().unwrap().set_messages(channel_id, msgs, has_more);
        }
    }

    pub fn prepend_messages(&self, channel_id: i64, json: &str, has_more: bool) {
        if let Ok(msgs) = serde_json::from_str::<Vec<ChannelMessage>>(json) {
            self.state.write().unwrap().prepend_messages(channel_id, msgs, has_more);
        }
    }

    pub fn add_message(&self, channel_id: i64, json: &str) {
        if let Ok(msg) = serde_json::from_str::<ChannelMessage>(json) {
            self.state.write().unwrap().add_message(channel_id, msg);
        }
    }

    pub fn on_new_message(&self, json: &str) -> bool {
        match serde_json::from_str::<ChannelMessage>(json) {
            Ok(msg) => self.state.write().unwrap().on_new_message(msg),
            Err(_) => false,
        }
    }

    pub fn update_message_local(&self, channel_id: i64, json: &str) {
        if let Ok(msg) = serde_json::from_str::<ChannelMessage>(json) {
            self.state.write().unwrap().update_message(channel_id, msg);
        }
    }

    pub fn remove_message_local(&self, channel_id: i64, message_id: i64) {
        self.state.write().unwrap().remove_message(channel_id, message_id);
    }

    pub fn set_unread_counts(&self, json: &str) {
        if let Ok(counts) = serde_json::from_str(json) {
            self.state.write().unwrap().set_unread_counts(counts);
        }
    }

    pub fn increment_unread(&self, channel_id: i64) {
        self.state.write().unwrap().increment_unread(channel_id);
    }

    pub fn clear_channel_unread(&self, channel_id: i64) {
        self.state.write().unwrap().clear_channel_unread(channel_id);
    }

    pub fn set_mention_counts(&self, json: &str) {
        if let Ok(counts) = serde_json::from_str(json) {
            self.state.write().unwrap().set_mention_counts(counts);
        }
    }

    pub fn increment_mention(&self, channel_id: i64) {
        self.state.write().unwrap().increment_mention(channel_id);
    }

    pub fn clear_channel_mentions(&self, channel_id: i64) {
        self.state.write().unwrap().clear_channel_mentions(channel_id);
    }

    pub fn set_last_message(&self, channel_id: i64, json: &str) {
        if let Ok(p) = serde_json::from_str(json) {
            self.state.write().unwrap().set_last_message(channel_id, p);
        }
    }

    pub async fn fetch_channels(&self, include_archived: Option<bool>) -> Result<String, String> {
        let resp = self.client
            .list_channels(include_archived)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_channels(resp.channels.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn fetch_channel(&self, id: i64) -> Result<String, String> {
        let ch: Channel = self.client
            .get_channel(id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().update_channel(id, ch.clone());
        serde_json::to_string(&ch).map_err(crate::wire)
    }

    pub async fn create_channel(&self, request_json: &str) -> Result<String, String> {
        let req: CreateChannelRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let ch: Channel = self.client
            .create_channel(&req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().add_channel(ch.clone());
        serde_json::to_string(&ch).map_err(crate::wire)
    }

    pub async fn update_channel(&self, id: i64, request_json: &str) -> Result<String, String> {
        let req: UpdateChannelRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let ch: Channel = self.client
            .update_channel(id, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().update_channel(id, ch.clone());
        serde_json::to_string(&ch).map_err(crate::wire)
    }

    pub async fn archive_channel(&self, id: i64) -> Result<(), String> {
        self.client.archive_channel(id).await.map_err(crate::wire)?;
        if let Some(ch) = self.state.read().unwrap().get_channel(id).cloned() {
            let mut updated = ch;
            updated.is_archived = true;
            self.state.write().unwrap().update_channel(id, updated);
        }
        Ok(())
    }

    pub async fn unarchive_channel(&self, id: i64) -> Result<(), String> {
        self.client.unarchive_channel(id).await.map_err(crate::wire)?;
        if let Some(ch) = self.state.read().unwrap().get_channel(id).cloned() {
            let mut updated = ch;
            updated.is_archived = false;
            self.state.write().unwrap().update_channel(id, updated);
        }
        Ok(())
    }

    pub async fn join_channel(&self, channel_id: i64, pod_key: &str) -> Result<String, String> {
        let req = JoinChannelPodRequest { pod_key: pod_key.to_string() };
        self.client.join_channel_pod(channel_id, &req).await.map_err(crate::wire)?;
        let ch: Channel = self.client
            .get_channel(channel_id)
            .await.map_err(crate::wire)?;
        let pods_resp = self.client
            .get_channel_pods(channel_id)
            .await.map_err(crate::wire)?;
        {
            let mut state = self.state.write().unwrap();
            state.update_channel(channel_id, ch.clone());
            state.set_channel_pods(channel_id, pods_resp.pods);
        }
        serde_json::to_string(&ch).map_err(crate::wire)
    }

    pub async fn leave_channel(&self, channel_id: i64, pod_key: &str) -> Result<String, String> {
        self.client.leave_channel_pod(channel_id, pod_key).await.map_err(crate::wire)?;
        let ch: Channel = self.client
            .get_channel(channel_id)
            .await.map_err(crate::wire)?;
        let pods_resp = self.client
            .get_channel_pods(channel_id)
            .await.map_err(crate::wire)?;
        {
            let mut state = self.state.write().unwrap();
            state.update_channel(channel_id, ch.clone());
            state.set_channel_pods(channel_id, pods_resp.pods);
        }
        serde_json::to_string(&ch).map_err(crate::wire)
    }

    pub async fn fetch_messages(
        &self, channel_id: i64, limit: Option<u32>, before_id: Option<i64>,
    ) -> Result<String, String> {
        let resp = self.client
            .get_channel_messages(channel_id, limit, before_id)
            .await.map_err(crate::wire)?;
        if before_id.is_some() {
            self.state.write().unwrap().prepend_messages(
                channel_id, resp.messages.clone(), false,
            );
        } else {
            self.state.write().unwrap().set_messages(
                channel_id, resp.messages.clone(), false,
            );
        }
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn send_message(&self, channel_id: i64, request_json: &str) -> Result<String, String> {
        let value: serde_json::Value = serde_json::from_str(request_json).map_err(crate::wire)?;
        let req = if request_has_new_shape(&value) {
            serde_json::from_value::<SendChannelMessageRequest>(value).map_err(crate::wire)?
        } else {
            SendChannelMessageRequest { content: Some(value), ..Default::default() }
        };
        let msg: ChannelMessage = self.client
            .send_channel_message(channel_id, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().on_new_message(msg.clone());
        serde_json::to_string(&msg).map_err(crate::wire)
    }

    /// Accepts either an `EditChannelMessageRequest` or a bare
    /// `MessageContent` AST from legacy callers — bare AST gets rewrapped
    /// into `{content: <ast>}` (shape detected structurally).
    pub async fn edit_message(
        &self, channel_id: i64, message_id: i64, request_json: &str,
    ) -> Result<String, String> {
        let value: serde_json::Value = serde_json::from_str(request_json)
            .map_err(|e| format!("invalid edit request JSON: {e}"))?;
        let req = if request_has_new_shape(&value) {
            serde_json::from_value::<EditChannelMessageRequest>(value)
                .map_err(|e| format!("invalid edit request JSON: {e}"))?
        } else {
            EditChannelMessageRequest { content: Some(value), ..Default::default() }
        };
        let msg: ChannelMessage = self.client
            .edit_channel_message(channel_id, message_id, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().update_message(channel_id, msg.clone());
        serde_json::to_string(&msg).map_err(crate::wire)
    }

    pub async fn delete_message(&self, channel_id: i64, message_id: i64) -> Result<(), String> {
        self.client
            .delete_channel_message(channel_id, message_id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().remove_message(channel_id, message_id);
        Ok(())
    }

    pub async fn fetch_unread_counts(&self) -> Result<String, String> {
        let resp = self.client
            .get_channel_unread_counts()
            .await.map_err(crate::wire)?;
        let counts: std::collections::HashMap<i64, u32> = resp.unread
            .into_iter()
            .filter_map(|(k, v)| k.parse::<i64>().ok().map(|id| (id, v)))
            .collect();
        self.state.write().unwrap().set_unread_counts(counts);
        serde_json::to_string(self.state.read().unwrap().get_all_unread_counts()).map_err(crate::wire)
    }

    pub async fn mark_read(&self, channel_id: i64, message_id: i64) -> Result<(), String> {
        self.client
            .mark_channel_read(channel_id, message_id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().clear_channel_unread(channel_id);
        Ok(())
    }

    pub async fn mute_channel(&self, channel_id: i64, muted: bool) -> Result<(), String> {
        let req = MuteChannelRequest { muted };
        self.client.mute_channel(channel_id, &req).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn get_channel_pods(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .get_channel_pods(id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_channel_pods(id, resp.pods.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub fn channel_pods_json(&self, id: i64) -> String {
        let pods = self.state.read().unwrap().get_channel_pods(id);
        serde_json::to_string(&pods).unwrap_or_else(|_| "[]".into())
    }

    pub async fn fetch_channel_members(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .list_channel_members(id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_channel_members(id, resp.members.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn invite_channel_members(&self, id: i64, user_ids_json: &str) -> Result<(), String> {
        let user_ids: Vec<i64> = serde_json::from_str(user_ids_json).map_err(crate::wire)?;
        let req = agentsmesh_types::InviteChannelMembersRequest { user_ids };
        self.client
            .invite_channel_members(id, &req)
            .await.map_err(crate::wire)?;
        let fresh = self.client
            .list_channel_members(id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_channel_members(id, fresh.members);
        Ok(())
    }

    pub async fn remove_channel_member(&self, id: i64, user_id: i64) -> Result<(), String> {
        self.client
            .remove_channel_member(id, user_id)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().remove_channel_member(id, user_id);
        Ok(())
    }

    pub fn channel_members_json(&self, id: i64) -> String {
        let members = self.state.read().unwrap().get_channel_members(id);
        serde_json::to_string(&members).unwrap_or_else(|_| "[]".into())
    }

    pub async fn search_channel_messages(&self, id: i64, q: &str, limit: Option<u32>) -> Result<String, String> {
        let resp = self.client
            .search_channel_messages(id, q, limit)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }
}

fn request_has_new_shape(v: &serde_json::Value) -> bool {
    let Some(obj) = v.as_object() else { return false };
    obj.contains_key("source")
        || obj.contains_key("content")
        || obj.contains_key("mentions")
        || obj.contains_key("attachment_key")
        || obj.contains_key("pod_key")
        || obj.contains_key("reply_to")
}
