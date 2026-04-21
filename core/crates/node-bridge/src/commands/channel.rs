use napi_derive::napi;
use crate::AppState;

#[napi]
impl AppState {
    #[napi]
    pub async fn channel_channels_json(&self) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.channels_json())
    }

    #[napi]
    pub async fn channel_current_channel_json(&self) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.current_channel_json().unwrap_or_default())
    }

    #[napi]
    pub async fn channel_get_channel_json(&self, id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.get_channel_json(id).unwrap_or_default())
    }

    #[napi]
    pub async fn channel_filter_channels_json(&self, query: String, include_archived: bool) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.filter_channels_json(&query, include_archived))
    }

    #[napi]
    pub async fn channel_get_messages_json(&self, channel_id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.get_messages_json(channel_id).unwrap_or_default())
    }

    #[napi]
    pub async fn channel_get_unread_count(&self, channel_id: i64) -> napi::Result<u32> {
        let svc = self.channel.lock().await;
            Ok(svc.get_unread_count(channel_id))
    }

    #[napi]
    pub async fn channel_total_unread_count(&self) -> napi::Result<u32> {
        let svc = self.channel.lock().await;
            Ok(svc.total_unread_count())
    }

    #[napi]
    pub async fn channel_unread_counts_json(&self) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.unread_counts_json())
    }

    #[napi]
    pub async fn channel_get_mention_count(&self, channel_id: i64) -> napi::Result<u32> {
        let svc = self.channel.lock().await;
            Ok(svc.get_mention_count(channel_id))
    }

    #[napi]
    pub async fn channel_total_mention_count(&self) -> napi::Result<u32> {
        let svc = self.channel.lock().await;
            Ok(svc.total_mention_count())
    }

    #[napi]
    pub async fn channel_mention_counts_json(&self) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.mention_counts_json())
    }

    #[napi]
    pub async fn channel_sorted_channel_ids_json(&self, mode: String, include_archived: bool) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.sorted_channel_ids_json(&mode, include_archived))
    }

    #[napi]
    pub async fn channel_get_last_message_json(&self, channel_id: i64) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.get_last_message_json(channel_id).unwrap_or_default())
    }

    #[napi]
    pub async fn channel_set_channels(&self, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_channels(&json);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_current_channel(&self, id: Option<i64>) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_current_channel(id);
            Ok(())
    }

    #[napi]
    pub async fn channel_select_channel(&self, id: Option<i64>) -> napi::Result<String> {
        let svc = self.channel.lock().await;
            Ok(svc.select_channel(id).unwrap_or_default())
    }

    #[napi]
    pub async fn channel_add_channel_local(&self, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.add_channel_local(&json);
            Ok(())
    }

    #[napi]
    pub async fn channel_update_channel_local(&self, id: i64, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.update_channel_local(id, &json);
            Ok(())
    }

    #[napi]
    pub async fn channel_remove_channel_local(&self, id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.remove_channel_local(id);
            Ok(())
    }

}
