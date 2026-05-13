// proto.channel.v1.ChannelService Connect-RPC client bindings. Procedure
// paths derive from `proto.channel.v1.ChannelService.<Method>` (conventions
// §12). The legacy REST surface was retired; Connect handlers in
// backend/internal/api/connect/channel/ now own the data plane.

use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_channel_v1 as channel_proto;

impl ApiClient {
    pub async fn list_channels_connect(
        &self, req: &channel_proto::ListChannelsRequest,
    ) -> Result<channel_proto::ListChannelsResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/ListChannels", req).await
    }

    pub async fn get_channel_connect(
        &self, req: &channel_proto::GetChannelRequest,
    ) -> Result<channel_proto::Channel, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/GetChannel", req).await
    }

    pub async fn create_channel_connect(
        &self, req: &channel_proto::CreateChannelRequest,
    ) -> Result<channel_proto::Channel, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/CreateChannel", req).await
    }

    pub async fn update_channel_connect(
        &self, req: &channel_proto::UpdateChannelRequest,
    ) -> Result<channel_proto::Channel, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/UpdateChannel", req).await
    }

    pub async fn archive_channel_connect(
        &self, req: &channel_proto::ArchiveChannelRequest,
    ) -> Result<channel_proto::ArchiveChannelResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/ArchiveChannel", req).await
    }

    pub async fn unarchive_channel_connect(
        &self, req: &channel_proto::UnarchiveChannelRequest,
    ) -> Result<channel_proto::UnarchiveChannelResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/UnarchiveChannel", req).await
    }

    pub async fn get_channel_document_connect(
        &self, req: &channel_proto::GetChannelDocumentRequest,
    ) -> Result<channel_proto::GetChannelDocumentResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/GetChannelDocument", req).await
    }

    pub async fn update_channel_document_connect(
        &self, req: &channel_proto::UpdateChannelDocumentRequest,
    ) -> Result<channel_proto::UpdateChannelDocumentResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/UpdateChannelDocument", req).await
    }

    pub async fn list_channel_messages_connect(
        &self, req: &channel_proto::ListChannelMessagesRequest,
    ) -> Result<channel_proto::ListChannelMessagesResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/ListChannelMessages", req).await
    }

    pub async fn search_channel_messages_connect(
        &self, req: &channel_proto::SearchChannelMessagesRequest,
    ) -> Result<channel_proto::SearchChannelMessagesResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/SearchChannelMessages", req).await
    }

    pub async fn send_channel_message_connect(
        &self, req: &channel_proto::SendChannelMessageRequest,
    ) -> Result<channel_proto::ChannelMessage, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/SendChannelMessage", req).await
    }

    pub async fn edit_channel_message_connect(
        &self, req: &channel_proto::EditChannelMessageRequest,
    ) -> Result<channel_proto::ChannelMessage, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/EditChannelMessage", req).await
    }

    pub async fn delete_channel_message_connect(
        &self, req: &channel_proto::DeleteChannelMessageRequest,
    ) -> Result<channel_proto::DeleteChannelMessageResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/DeleteChannelMessage", req).await
    }

    pub async fn mark_channel_read_connect(
        &self, req: &channel_proto::MarkChannelReadRequest,
    ) -> Result<channel_proto::MarkChannelReadResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/MarkChannelRead", req).await
    }

    pub async fn get_channel_unread_counts_connect(
        &self, req: &channel_proto::GetChannelUnreadCountsRequest,
    ) -> Result<channel_proto::GetChannelUnreadCountsResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/GetChannelUnreadCounts", req).await
    }

    pub async fn mute_channel_connect(
        &self, req: &channel_proto::MuteChannelRequest,
    ) -> Result<channel_proto::MuteChannelResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/MuteChannel", req).await
    }

    pub async fn list_channel_members_connect(
        &self, req: &channel_proto::ListChannelMembersRequest,
    ) -> Result<channel_proto::ListChannelMembersResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/ListChannelMembers", req).await
    }

    pub async fn join_channel_connect(
        &self, req: &channel_proto::JoinChannelRequest,
    ) -> Result<channel_proto::JoinChannelResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/JoinChannel", req).await
    }

    pub async fn leave_channel_connect(
        &self, req: &channel_proto::LeaveChannelRequest,
    ) -> Result<channel_proto::LeaveChannelResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/LeaveChannel", req).await
    }

    pub async fn invite_channel_members_connect(
        &self, req: &channel_proto::InviteChannelMembersRequest,
    ) -> Result<channel_proto::InviteChannelMembersResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/InviteChannelMembers", req).await
    }

    pub async fn remove_channel_member_connect(
        &self, req: &channel_proto::RemoveChannelMemberRequest,
    ) -> Result<channel_proto::RemoveChannelMemberResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/RemoveChannelMember", req).await
    }

    pub async fn list_channel_pods_connect(
        &self, req: &channel_proto::ListChannelPodsRequest,
    ) -> Result<channel_proto::ListChannelPodsResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/ListChannelPods", req).await
    }

    pub async fn join_channel_pod_connect(
        &self, req: &channel_proto::JoinChannelPodRequest,
    ) -> Result<channel_proto::JoinChannelPodResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/JoinChannelPod", req).await
    }

    pub async fn leave_channel_pod_connect(
        &self, req: &channel_proto::LeaveChannelPodRequest,
    ) -> Result<channel_proto::LeaveChannelPodResponse, ApiError> {
        connect_call(self, "/proto.channel.v1.ChannelService/LeaveChannelPod", req).await
    }
}
