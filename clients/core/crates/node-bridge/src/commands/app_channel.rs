use napi_derive::napi;

use agentsmesh_types::proto_channel_state_v1::PatchChannelMemberCountRequest;
use prost::Message as _;

use crate::AppState;

// Channel state surface over the shared `runtime.state` (the dispatch-hook
// SSOT). The legacy `channel_*` commands operate on the per-service
// `ChannelService` cache; on desktop that cache is disjoint from the realtime
// dispatch target, so the renderer's realtime mirror starves. These `app_*`
// commands read/write the SAME `AppState` the EventBus dispatch mutates, so a
// post-dispatch snapshot read reflects realtime + fetched baseline together.
fn decode_err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(format!("decode: {e}"))
}

#[napi]
impl AppState {
    // ── Snapshot reads (main pushes these to the renderer after dispatch) ──

    #[napi]
    pub fn app_channels_json(&self) -> String {
        serde_json::to_string(self.runtime.state.read().channels.get_channels()).unwrap_or_default()
    }

    #[napi]
    pub fn app_channel_messages_json(&self, channel_id: i64) -> String {
        match self.runtime.state.read().channels.get_messages(channel_id) {
            Some(cache) => serde_json::to_string(&serde_json::json!({
                "messages": cache.messages,
                "has_more": cache.has_more,
            }))
            .unwrap_or_default(),
            None => String::new(),
        }
    }

    #[napi]
    pub fn app_channel_unread_counts_json(&self) -> String {
        serde_json::to_string(&self.runtime.state.read().channels.get_all_unread_counts())
            .unwrap_or_else(|_| "{}".to_string())
    }

    #[napi]
    pub fn app_channel_mention_counts_json(&self) -> String {
        serde_json::to_string(&self.runtime.state.read().channels.get_all_mention_counts())
            .unwrap_or_else(|_| "{}".to_string())
    }

    #[napi]
    pub fn app_channel_get_last_read_id(&self, channel_id: i64) -> f64 {
        self.runtime
            .state
            .read()
            .channels
            .get_last_read(channel_id)
            .unwrap_or(0) as f64
    }

    #[napi]
    pub fn app_channel_advance_last_read(&self, channel_id: i64, message_id: i64) {
        self.runtime.state.write().channels.advance_last_read(channel_id, message_id);
    }

    #[napi]
    pub fn app_channel_patch_member_count(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = PatchChannelMemberCountRequest::decode(&req_bytes[..]).map_err(decode_err)?;
        let mut guard = self.runtime.state.write();
        if let Some(existing) = guard.channels.get_channel(req.channel_id).cloned() {
            let mut next = existing;
            let curr = next.member_count.unwrap_or(0);
            next.member_count = Some((curr + req.delta as i64).max(0));
            guard.channels.update_channel(req.channel_id, next);
        }
        Ok(())
    }

    #[napi]
    pub fn app_channel_remove_message(&self, channel_id: i64, message_id: i64) {
        self.runtime
            .state
            .write()
            .channels
            .remove_message(channel_id, message_id);
    }

    #[napi]
    pub fn app_channel_clear_unread(&self, channel_id: i64) {
        let mut guard = self.runtime.state.write();
        guard.channels.clear_channel_unread(channel_id);
        guard.channels.clear_channel_mentions(channel_id);
    }

    // ── UI→Rust signals: current user (self-message rule) + active channel
    //    (unread suppression). Without these the SSOT can't compute unread. ──

    #[napi]
    pub fn app_set_current_user(&self, user_id: Option<i64>) {
        self.runtime.state.write().channels.set_current_user_id(user_id);
    }

    #[napi]
    pub fn app_select_channel(&self, id: Option<i64>) {
        self.runtime.state.write().channels.select_channel(id);
    }

    #[napi]
    pub fn app_set_current_channel(&self, id: Option<i64>) {
        self.runtime.state.write().channels.set_current_channel(id);
    }
}
