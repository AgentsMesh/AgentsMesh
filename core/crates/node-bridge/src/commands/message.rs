use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn message_send_message(&self, json: String, pod_key: Option<String>) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.send_message(&json, pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn message_get_messages(&self, unread_only: Option<bool>, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_messages(unread_only, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn message_get_unread_count(&self) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_unread_count().await.map_err(err)
    }

    #[napi]
    pub async fn message_get_message(&self, id: i64) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_message(id).await.map_err(err)
    }

    #[napi]
    pub async fn message_mark_read(&self, json: String) -> napi::Result<()> {
        let svc = self.message.lock().await;
            svc.mark_read(&json).await.map_err(err)
    }

    #[napi]
    pub async fn message_mark_all_read(&self) -> napi::Result<()> {
        let svc = self.message.lock().await;
            svc.mark_all_read().await.map_err(err)
    }

    #[napi]
    pub async fn message_get_conversation(&self, correlation_id: String, limit: Option<u32>) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_conversation(&correlation_id, limit).await.map_err(err)
    }

    #[napi]
    pub async fn message_get_sent_messages(&self, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_sent_messages(limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn message_get_dead_letters(&self, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.message.lock().await;
            svc.get_dead_letters(limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn message_replay_dead_letter(&self, entry_id: i64) -> napi::Result<()> {
        let svc = self.message.lock().await;
            svc.replay_dead_letter(entry_id).await.map_err(err)
    }

}
