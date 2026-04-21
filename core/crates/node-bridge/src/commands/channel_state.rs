use napi_derive::napi;
use crate::AppState;

#[napi]
impl AppState {
    #[napi]
    pub async fn channel_set_current_user(&self, user_json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_current_user(&user_json);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_current_user_id(&self, user_id: Option<i64>) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_current_user_id(user_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_messages(&self, channel_id: i64, json: String, has_more: bool) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_messages(channel_id, &json, has_more);
            Ok(())
    }

    #[napi]
    pub async fn channel_prepend_messages(&self, channel_id: i64, json: String, has_more: bool) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.prepend_messages(channel_id, &json, has_more);
            Ok(())
    }

    #[napi]
    pub async fn channel_add_message(&self, channel_id: i64, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.add_message(channel_id, &json);
            Ok(())
    }

    #[napi]
    pub async fn channel_on_new_message(&self, json: String) -> napi::Result<bool> {
        let svc = self.channel.lock().await;
            Ok(svc.on_new_message(&json))
    }

    #[napi]
    pub async fn channel_update_message_local(&self, channel_id: i64, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.update_message_local(channel_id, &json);
            Ok(())
    }

    #[napi]
    pub async fn channel_remove_message_local(&self, channel_id: i64, message_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.remove_message_local(channel_id, message_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_unread_counts(&self, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_unread_counts(&json);
            Ok(())
    }

    #[napi]
    pub async fn channel_increment_unread(&self, channel_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.increment_unread(channel_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_clear_channel_unread(&self, channel_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.clear_channel_unread(channel_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_mention_counts(&self, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_mention_counts(&json);
            Ok(())
    }

    #[napi]
    pub async fn channel_increment_mention(&self, channel_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.increment_mention(channel_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_clear_channel_mentions(&self, channel_id: i64) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.clear_channel_mentions(channel_id);
            Ok(())
    }

    #[napi]
    pub async fn channel_set_last_message(&self, channel_id: i64, json: String) -> napi::Result<()> {
        let svc = self.channel.lock().await;
            svc.set_last_message(channel_id, &json);
            Ok(())
    }

}
