use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use prost::Message as _;

pub struct ChannelService {
    client: Arc<ApiClient>,
}

impl ChannelService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    /// Crate-local accessor used by channel_connect.rs to forward to the
    /// underlying api-client `*_connect` methods. The channel cache itself
    /// is the shared `AppState.channels` (dispatch-hook SSOT), reached via
    /// the wasm/napi `app_channel*` surface — this service is networking-only.
    pub(crate) fn client(&self) -> &ApiClient { &self.client }
}

// =============================================================================
// Connect-RPC bridge methods. Binary in (prost-encoded), binary out — same wire
// the wasm/node-bridge layers speak. Each method decodes the request, calls the
// `*_connect` method on the shared ApiClient, and re-encodes the response.
// =============================================================================

use agentsmesh_types::proto_channel_v1 as channel_proto;

use crate::wire;

macro_rules! connect_bridge {
    ($name:ident, $req:ident, $client_call:ident) => {
        pub async fn $name(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
            let req = channel_proto::$req::decode(request_bytes)
                .map_err(|e| format!("decode {}: {e}", stringify!($req)))?;
            tracing::debug!(target: "channel", rpc = stringify!($req));
            let resp = self.client().$client_call(&req).await.map_err(wire)?;
            Ok(resp.encode_to_vec())
        }
    };
}

impl ChannelService {
    connect_bridge!(list_channels_connect, ListChannelsRequest, list_channels_connect);
    connect_bridge!(get_channel_connect, GetChannelRequest, get_channel_connect);
    connect_bridge!(create_channel_connect, CreateChannelRequest, create_channel_connect);
    connect_bridge!(update_channel_connect, UpdateChannelRequest, update_channel_connect);
    connect_bridge!(archive_channel_connect, ArchiveChannelRequest, archive_channel_connect);
    connect_bridge!(unarchive_channel_connect, UnarchiveChannelRequest, unarchive_channel_connect);
    connect_bridge!(get_channel_document_connect, GetChannelDocumentRequest, get_channel_document_connect);
    connect_bridge!(update_channel_document_connect, UpdateChannelDocumentRequest, update_channel_document_connect);
    connect_bridge!(list_channel_messages_connect, ListChannelMessagesRequest, list_channel_messages_connect);
    connect_bridge!(search_channel_messages_connect, SearchChannelMessagesRequest, search_channel_messages_connect);
    connect_bridge!(send_channel_message_connect, SendChannelMessageRequest, send_channel_message_connect);
    connect_bridge!(edit_channel_message_connect, EditChannelMessageRequest, edit_channel_message_connect);
    connect_bridge!(delete_channel_message_connect, DeleteChannelMessageRequest, delete_channel_message_connect);
    connect_bridge!(mark_channel_read_connect, MarkChannelReadRequest, mark_channel_read_connect);
    connect_bridge!(get_channel_unread_counts_connect, GetChannelUnreadCountsRequest, get_channel_unread_counts_connect);
    connect_bridge!(mute_channel_connect, MuteChannelRequest, mute_channel_connect);
    connect_bridge!(list_channel_members_connect, ListChannelMembersRequest, list_channel_members_connect);
    connect_bridge!(join_channel_connect, JoinChannelRequest, join_channel_connect);
    connect_bridge!(leave_channel_connect, LeaveChannelRequest, leave_channel_connect);
    connect_bridge!(invite_channel_members_connect, InviteChannelMembersRequest, invite_channel_members_connect);
    connect_bridge!(remove_channel_member_connect, RemoveChannelMemberRequest, remove_channel_member_connect);
    connect_bridge!(list_channel_pods_connect, ListChannelPodsRequest, list_channel_pods_connect);
    connect_bridge!(join_channel_pod_connect, JoinChannelPodRequest, join_channel_pod_connect);
    connect_bridge!(leave_channel_pod_connect, LeaveChannelPodRequest, leave_channel_pod_connect);
}
