use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_channels(
        &self,
        include_archived: Option<bool>,
    ) -> Result<ChannelListResponse, ApiError> {
        let mut path = self.org_path("/channels");
        if let Some(archived) = include_archived {
            path = format!("{path}?include_archived={archived}");
        }
        self.get(&path).await
    }

    pub async fn get_channel(&self, id: i64) -> Result<Channel, ApiError> {
        self.get_resource(&self.org_path(&format!("/channels/{id}")), "channel").await
    }

    pub async fn create_channel(
        &self,
        data: &CreateChannelRequest,
    ) -> Result<Channel, ApiError> {
        self.post_resource(&self.org_path("/channels"), data, "channel").await
    }

    pub async fn update_channel(
        &self,
        id: i64,
        data: &UpdateChannelRequest,
    ) -> Result<Channel, ApiError> {
        self.put_resource(&self.org_path(&format!("/channels/{id}")), data, "channel").await
    }

    pub async fn archive_channel(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/channels/{id}/archive")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn unarchive_channel(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/channels/{id}/unarchive")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn get_channel_messages(
        &self,
        id: i64,
        limit: Option<u32>,
        before_id: Option<i64>,
    ) -> Result<ChannelMessageListResponse, ApiError> {
        let mut path = self.org_path(&format!("/channels/{id}/messages"));
        let mut params = Vec::new();
        if let Some(l) = limit {
            params.push(format!("limit={l}"));
        }
        if let Some(b) = before_id {
            params.push(format!("before_id={b}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn send_channel_message(
        &self,
        id: i64,
        data: &SendChannelMessageRequest,
    ) -> Result<ChannelMessage, ApiError> {
        self.post_resource(&self.org_path(&format!("/channels/{id}/messages")), data, "message").await
    }

    pub async fn get_channel_pods(&self, id: i64) -> Result<PodListResponse, ApiError> {
        self.get(&self.org_path(&format!("/channels/{id}/pods")))
            .await
    }

    pub async fn join_channel_pod(
        &self,
        id: i64,
        data: &JoinChannelPodRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(&self.org_path(&format!("/channels/{id}/pods")), data)
            .await
    }

    pub async fn leave_channel_pod(
        &self,
        id: i64,
        pod_key: &str,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/channels/{id}/pods/{pod_key}")))
            .await
    }

    pub async fn mark_channel_read(
        &self,
        id: i64,
        message_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/channels/{id}/read")),
            &serde_json::json!({ "message_id": message_id }),
        )
        .await
    }

    pub async fn get_channel_unread_counts(&self) -> Result<ChannelUnreadResponse, ApiError> {
        self.get(&self.org_path("/channels/unread")).await
    }

    pub async fn edit_channel_message(
        &self,
        channel_id: i64,
        message_id: i64,
        data: &EditChannelMessageRequest,
    ) -> Result<ChannelMessage, ApiError> {
        self.put_resource(
            &self.org_path(&format!("/channels/{channel_id}/messages/{message_id}")),
            data, "message",
        ).await
    }

    pub async fn delete_channel_message(
        &self,
        channel_id: i64,
        message_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!(
            "/channels/{channel_id}/messages/{message_id}"
        )))
        .await
    }

    pub async fn mute_channel(
        &self,
        id: i64,
        data: &MuteChannelRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(&self.org_path(&format!("/channels/{id}/mute")), data)
            .await
    }

    pub async fn list_channel_members(
        &self,
        id: i64,
    ) -> Result<ChannelMemberListResponse, ApiError> {
        self.get(&self.org_path(&format!("/channels/{id}/members")))
            .await
    }

    pub async fn invite_channel_members(
        &self,
        id: i64,
        data: &InviteChannelMembersRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(&self.org_path(&format!("/channels/{id}/members")), data)
            .await
    }

    pub async fn remove_channel_member(
        &self,
        id: i64,
        user_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/channels/{id}/members/{user_id}")))
            .await
    }

    pub async fn search_channel_messages(
        &self,
        id: i64,
        q: &str,
        limit: Option<u32>,
    ) -> Result<serde_json::Value, ApiError> {
        let lim = limit.unwrap_or(20);
        let path = self.org_path(&format!(
            "/channels/{id}/messages/search?q={}&limit={lim}",
            urlencoding::encode(q)
        ));
        self.get(&path).await
    }
}
