// Facade re-export of the channel Connect-RPC adapter. Business code imports
// from here (or from the `@/lib/api` barrel) so the wire-shape layer stays
// internal to the facade boundary. Tests mock this path.
//
// The wire layer was split into 3 SRP-aligned files (CRUD / messages /
// members) — this facade unifies them so callers see one surface.

export {
  channelFromProto,
  messageFromProto,
  listChannels,
  listChannelsRaw,
  getChannel,
  getChannelRaw,
  createChannel,
  updateChannel,
  archiveChannel,
  unarchiveChannel,
  getChannelDocument,
  updateChannelDocument,
  type ChannelData,
  type ChannelMessage,
} from "../connect/channelConnect";

export {
  listChannelMessages,
  listChannelMessagesRaw,
  searchChannelMessages,
  sendChannelMessage,
  editChannelMessage,
  deleteChannelMessage,
  markChannelRead,
  getChannelUnreadCounts,
  muteChannel,
  type SendChannelMessagePayload,
} from "../connect/channelMessageConnect";

export {
  listChannelMembers,
  listChannelMembersRaw,
  joinChannel,
  leaveChannel,
  inviteChannelMembers,
  removeChannelMember,
  listChannelPods,
  listChannelPodsRaw,
  joinChannelPod,
  leaveChannelPod,
  type ChannelMemberData,
  type ChannelPodSummary,
} from "../connect/channelMembersConnect";
