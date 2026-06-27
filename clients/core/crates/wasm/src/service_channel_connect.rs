// Connect-RPC bridge methods for WasmChannelService. Binary in, binary out
// (conventions §2.5).
//
// TS encodes the request via @bufbuild/protobuf .toBinary(), passes the
// Uint8Array in, receives a Uint8Array back, decodes via .fromBinary().
// No JSON intermediate; conventions §2.5 forbids it on the client.
//
// Split from service_channel.rs to honor the 200-line/file limit. Both
// `impl` blocks attach to WasmChannelService; wasm-bindgen handles multiple
// impl blocks as long as each is annotated.

use wasm_bindgen::prelude::*;

use crate::service_channel::WasmChannelService;

#[wasm_bindgen]
impl WasmChannelService {
    #[wasm_bindgen(js_name = listChannelsConnect)]
    pub async fn list_channels_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_channels_connect(request).await
    }

    #[wasm_bindgen(js_name = getChannelConnect)]
    pub async fn get_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = createChannelConnect)]
    pub async fn create_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.create_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = updateChannelConnect)]
    pub async fn update_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = archiveChannelConnect)]
    pub async fn archive_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.archive_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = unarchiveChannelConnect)]
    pub async fn unarchive_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.unarchive_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = getChannelDocumentConnect)]
    pub async fn get_channel_document_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_channel_document_connect(request).await
    }

    #[wasm_bindgen(js_name = updateChannelDocumentConnect)]
    pub async fn update_channel_document_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_channel_document_connect(request).await
    }

    #[wasm_bindgen(js_name = listChannelMessagesConnect)]
    pub async fn list_channel_messages_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_channel_messages_connect(request).await
    }

    #[wasm_bindgen(js_name = searchChannelMessagesConnect)]
    pub async fn search_channel_messages_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.search_channel_messages_connect(request).await
    }

    #[wasm_bindgen(js_name = sendChannelMessageConnect)]
    pub async fn send_channel_message_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.send_channel_message_connect(request).await
    }

    #[wasm_bindgen(js_name = editChannelMessageConnect)]
    pub async fn edit_channel_message_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.edit_channel_message_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteChannelMessageConnect)]
    pub async fn delete_channel_message_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.delete_channel_message_connect(request).await
    }

    #[wasm_bindgen(js_name = markChannelReadConnect)]
    pub async fn mark_channel_read_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.mark_channel_read_connect(request).await
    }

    #[wasm_bindgen(js_name = getChannelUnreadCountsConnect)]
    pub async fn get_channel_unread_counts_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_channel_unread_counts_connect(request).await
    }

    #[wasm_bindgen(js_name = muteChannelConnect)]
    pub async fn mute_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.mute_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = pinChannelConnect)]
    pub async fn pin_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.pin_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = markChannelUnreadConnect)]
    pub async fn mark_channel_unread_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.mark_channel_unread_connect(request).await
    }

    #[wasm_bindgen(js_name = getMessageReadByConnect)]
    pub async fn get_message_read_by_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_message_read_by_connect(request).await
    }

    #[wasm_bindgen(js_name = listChannelMembersConnect)]
    pub async fn list_channel_members_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_channel_members_connect(request).await
    }

    #[wasm_bindgen(js_name = joinChannelConnect)]
    pub async fn join_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.join_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = leaveChannelConnect)]
    pub async fn leave_channel_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.leave_channel_connect(request).await
    }

    #[wasm_bindgen(js_name = inviteChannelMembersConnect)]
    pub async fn invite_channel_members_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.invite_channel_members_connect(request).await
    }

    #[wasm_bindgen(js_name = removeChannelMemberConnect)]
    pub async fn remove_channel_member_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.remove_channel_member_connect(request).await
    }

    #[wasm_bindgen(js_name = listChannelPodsConnect)]
    pub async fn list_channel_pods_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_channel_pods_connect(request).await
    }

    #[wasm_bindgen(js_name = joinChannelPodConnect)]
    pub async fn join_channel_pod_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.join_channel_pod_connect(request).await
    }

    #[wasm_bindgen(js_name = leaveChannelPodConnect)]
    pub async fn leave_channel_pod_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.leave_channel_pod_connect(request).await
    }
}
