import { request, orgPath } from "./base";

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
  is_archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface ChannelMessage {
  id: number;
  channel_id: number;
  sender_pod?: string;
  sender_user_id?: number;
  message_type: "text" | "system" | "code" | "command";
  content: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  // Backend returns these field names (from GORM json tags)
  sender_pod_info?: {
    pod_key: string;
    agent_type?: {
      name: string;
    };
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
  // List channels with optional filters
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

  // Get a single channel
  get: (id: number) =>
    request<{ channel: ChannelData }>(`${orgPath("/channels")}/${id}`),

  // Create a new channel
  create: (data: {
    name: string;
    description?: string;
    document?: string;
    repository_id?: number;
    ticket_slug?: string;
  }) =>
    request<{ channel: ChannelData }>(orgPath("/channels"), {
      method: "POST",
      body: data,
    }),

  // Update a channel
  update: (id: number, data: { name?: string; description?: string; document?: string }) =>
    request<{ channel: ChannelData }>(`${orgPath("/channels")}/${id}`, {
      method: "PUT",
      body: data,
    }),

  // Archive a channel
  archive: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/archive`, {
      method: "POST",
    }),

  // Unarchive a channel
  unarchive: (id: number) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/unarchive`, {
      method: "POST",
    }),

  // Get messages in a channel
  getMessages: (id: number, limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit) params.append("limit", String(limit));
    if (offset) params.append("offset", String(offset));
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ messages: ChannelMessage[] }>(`${orgPath("/channels")}/${id}/messages${query}`);
  },

  // Send a message to a channel
  sendMessage: (id: number, content: string, podKey?: string, messageType?: string) =>
    request<{ message: ChannelMessage }>(`${orgPath("/channels")}/${id}/messages`, {
      method: "POST",
      body: { content, pod_key: podKey, message_type: messageType || "text" },
    }),

  // Get pods joined to a channel
  getPods: (id: number) =>
    request<{
      pods: Array<{
        id: number;
        pod_key: string;
        status: string;
        agent_status: string;
      }>;
      total: number;
    }>(`${orgPath("/channels")}/${id}/pods`),

  // Join a pod to a channel
  joinPod: (id: number, podKey: string) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/pods`, {
      method: "POST",
      body: { pod_key: podKey },
    }),

  // Remove a pod from a channel
  leavePod: (id: number, podKey: string) =>
    request<{ message: string }>(`${orgPath("/channels")}/${id}/pods/${podKey}`, {
      method: "DELETE",
    }),
};
