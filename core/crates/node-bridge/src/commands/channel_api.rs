use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn channel_fetch_channels(&self, include_archived: Option<bool>) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.fetch_channels(include_archived).await.map_err(err)
    }

    #[napi]
    pub async fn channel_fetch_channel(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.fetch_channel(id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_create_channel(&self, request_json: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
        svc.create_channel(&request_json).await.map_err(err)
    }

    #[napi]
    pub async fn channel_update_channel(&self, id: i64, request_json: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.update_channel(id, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn channel_archive_channel(&self, id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.archive_channel(id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_unarchive_channel(&self, id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.unarchive_channel(id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_join_channel(&self, channel_id: i64, pod_key: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.join_channel(channel_id, &pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn channel_leave_channel(&self, channel_id: i64, pod_key: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.leave_channel(channel_id, &pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn channel_fetch_messages(&self, channel_id: i64, limit: Option<u32>, before_id: Option<i64>) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.fetch_messages(channel_id, limit, before_id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_send_message(&self, channel_id: i64, request_json: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.send_message(channel_id, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn channel_edit_message(&self, channel_id: i64, message_id: i64, content: String) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.edit_message(channel_id, message_id, &content).await.map_err(err)
    }

    #[napi]
    pub async fn channel_delete_message(&self, channel_id: i64, message_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.delete_message(channel_id, message_id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_fetch_unread_counts(&self) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.fetch_unread_counts().await.map_err(err)
    }

    #[napi]
    pub async fn channel_mark_read(&self, channel_id: i64, message_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.mark_read(channel_id, message_id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_mute_channel(&self, channel_id: i64, muted: bool) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.mute_channel(channel_id, muted).await.map_err(err)
    }

    #[napi]
    pub async fn channel_get_channel_pods(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            svc.get_channel_pods(id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_channel_pods_json(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
        Ok(svc.channel_pods_json(id))
    }

    #[napi]
    pub async fn channel_fetch_channel_members(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
        svc.fetch_channel_members(id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_invite_channel_members(&self, id: i64, user_ids_json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
        svc.invite_channel_members(id, &user_ids_json).await.map_err(err)
    }

    #[napi]
    pub async fn channel_remove_channel_member(&self, id: i64, user_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
        svc.remove_channel_member(id, user_id).await.map_err(err)
    }

    #[napi]
    pub async fn channel_channel_members_json(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
        Ok(svc.channel_members_json(id))
    }

    #[napi]
    pub async fn channel_search_channel_messages(&self, id: i64, q: String, limit: Option<u32>) -> napi::Result<String> {
        let svc = self.channel.lock().await;
        svc.search_channel_messages(id, &q, limit).await.map_err(err)
    }

}
