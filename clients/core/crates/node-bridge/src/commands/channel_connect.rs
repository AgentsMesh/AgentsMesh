use napi_derive::napi;
use crate::{AppState, err};

// Channel Connect-RPC NAPI bridge. Mirrors auth_connect.rs — each method
// takes proto-encoded bytes, forwards to `services::ChannelService::*_connect`,
// returns proto-encoded bytes. The TS-side ElectronChannelService uses these
// to drive the channel domain identically to the WASM renderer, eliminating
// the legacy JSON-IPC shim path that broke when channel migration removed
// `service::ChannelService::send_message` and friends.

#[napi]
impl AppState {
    #[napi]
    pub async fn channel_list_channels_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.list_channels_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_get_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.get_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_create_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.create_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_update_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.update_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_archive_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.archive_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_unarchive_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.unarchive_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_list_channel_messages_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.list_channel_messages_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_send_channel_message_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.send_channel_message_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_edit_channel_message_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.edit_channel_message_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_delete_channel_message_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.delete_channel_message_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_mark_channel_read_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.mark_channel_read_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_get_channel_unread_counts_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.get_channel_unread_counts_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_list_channel_members_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.list_channel_members_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_mute_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.mute_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_pin_channel_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.pin_channel_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_mark_channel_unread_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.mark_channel_unread_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn channel_get_message_read_by_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.channel.lock().await;
        svc.get_message_read_by_connect(&request).await.map_err(err)
    }
}
