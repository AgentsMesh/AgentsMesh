import { request, orgPath } from "./base";
import type { MessageContent, MessageMentions } from "./channel-message-types";

export type { MessageContent, MessageMentions } from "./channel-message-types";
export type { InlineElement, Block } from "./channel-message-types";

// Channel types
export interface ChannelData {
  id: number;
  organization_id: number;
  name: string;
  description?: string;
  document?: string;
  repository_id?: number;
  ticket_id?: number;
  ticket_slug?: string;
  created_by_pod?: string;
  created_by_user_id?: number;
  visibility: "public" | "private";
  is_archived: boolean;
  is_member: boolean;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface ChannelMessage {
  id: number;
  channel_id: number;
  sender_pod?: string;
  sender_user_id?: number;
  message_type: string;
  body: string;
  content?: MessageContent;
  mentions?: MessageMentions;
  reply_to?: number;
  edited_at?: string;
  is_deleted?: boolean;
  created_at: string;
  sender_pod_info?: {
    pod_key: string;
    alias?: string;
    agent?: { name: string };
    ticket?: { slug?: string; title?: string };
    loop?: { name?: string };
    title?: string;
  };
  sender_user?: {
    id: number;
    username: string;
    name?: string;
    avatar_url?: string;
  };
}

// Channels API
export const channelApi = {
  list: (filters?: {
    repository_id?: number;
    ticket_slug?: string;
    include_archived?: boolean;
  }) => {
    const params = new URLSearchParams();
    if (filters?.repository_id) params.append("repository_id", String(filters.repository_id));
    if (filters?.ticket_slug) params.append("ticket_slug", filters.ticket_slug);
    if (filters?.include_archived) params.append("include_archived", "true");
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ channels: ChannelData[]; total: number }>(`${orgPath("/channels")}${query}`);
  },

  get: (id: number) =>
    request<{ channel: ChannelData }>(`${orgPath("/channels")}/${id}`),

  create: (data: {
    name: string;
    description?: string;
    document?: string;
    repository_id?: number;
    ticket_slug?: string;
    visibility?: "public" | "private";
    member_ids?: number[];
  }) =>
    request<{ channel: ChannelData }>(orgPath("/channels"), {
      method: "POST",
      body: data,
    }),

  update: (id: number, data: { name?: string; description?: string; document?: string }) =>
    request<{ channel: ChannelData }>(`${orgPath("/channels")}/${id}`, {
      method: "PUT",
      body: data,
    }),

  archive: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/archive`, { method: "POST" }),

  unarchive: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/unarchive`, { method: "POST" }),

  getMessages: (id: number, limit?: number, beforeId?: number) => {
    const params = new URLSearchParams();
    if (limit) params.append("limit", String(limit));
    if (beforeId) params.append("before_id", String(beforeId));
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ messages: ChannelMessage[]; has_more: boolean }>(`${orgPath("/channels")}/${id}/messages${query}`);
  },

  sendMessage: (id: number, content: MessageContent, podKey?: string) =>
    request<{ message: ChannelMessage }>(`${orgPath("/channels")}/${id}/messages`, {
      method: "POST",
      body: { content, ...(podKey ? { pod_key: podKey } : {}) },
    }),

  getPods: (id: number) =>
    request<{
      pods: Array<{ id: number; pod_key: string; alias?: string; status: string; agent_status: string }>;
      total: number;
    }>(`${orgPath("/channels")}/${id}/pods`),

  joinPod: (id: number, podKey: string) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/pods`, { method: "POST", body: { pod_key: podKey } }),

  leavePod: (id: number, podKey: string) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/pods/${podKey}`, { method: "DELETE" }),

  markRead: (id: number, messageId: number) =>
    request<{ status: string }>(`${orgPath("/channels")}/${id}/read`, { method: "POST", body: { message_id: messageId } }),

  getUnreadCounts: () =>
    request<{ unread: Record<string, number> }>(orgPath("/channels/unread")),

  mute: (id: number, muted: boolean) =>
    request<{ status: string }>(`${orgPath("/channels")}/${id}/mute`, { method: "POST", body: { muted } }),

  editMessage: (channelId: number, messageId: number, content: MessageContent) =>
    request<{ message: ChannelMessage }>(`${orgPath("/channels")}/${channelId}/messages/${messageId}`, {
      method: "PUT",
      body: { content },
    }),

  deleteMessage: (channelId: number, messageId: number) =>
    request<{ status: string }>(`${orgPath("/channels")}/${channelId}/messages/${messageId}`, { method: "DELETE" }),

  joinChannel: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/join`, { method: "POST" }),

  leaveChannel: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/leave`, { method: "POST" }),

  inviteMembers: (id: number, userIds: number[]) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/members`, { method: "POST", body: { user_ids: userIds } }),

  removeMember: (id: number, userId: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/members/${userId}`, { method: "DELETE" }),

  listMembers: (id: number) =>
    request<{ members: Array<{ channel_id: number; user_id: number; role: string; is_muted: boolean; joined_at: string }>; total: number }>(
      `${orgPath("/channels")}/${id}/members`
    ),
};
