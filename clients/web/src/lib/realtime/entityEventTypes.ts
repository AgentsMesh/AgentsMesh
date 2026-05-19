export interface PodStatusChangedData {
  pod_key: string;
  status: string;
  previous_status?: string;
  agent_status?: string;
  error_code?: string;
  error_message?: string;
}

export interface PodCreatedData {
  pod_key: string;
  status: string;
  agent_status?: string;
  runner_id: number;
  ticket_id?: number;
  ticket_slug?: string;
  created_by_id: number;
}

export interface RunnerStatusData {
  runner_id: number;
  node_id: string;
  status: string;
  current_pods?: number;
  last_heartbeat?: string;
}

export interface TicketStatusChangedData {
  slug: string;
  status: string;
  previous_status?: string;
}

export interface PodTitleChangedData {
  pod_key: string;
  title: string;
}

export interface PodAliasChangedData {
  pod_key: string;
  alias: string | null;
}

export interface PodInitProgressData {
  pod_key: string;
  phase: string; // pending, cloning, preparing, starting_pod, ready
  progress: number; // 0-100
  message: string; // Human-readable progress message
}

export interface PodPerpetualChangedData {
  pod_key: string;
  perpetual: boolean;
}

import type { MessageContent, MessageMentions } from "@/lib/api/channel-message-types";

export interface ChannelMessageData {
  id: number;
  channel_id: number;
  sender_pod?: string;
  sender_user_id?: number;
  sender_name?: string;
  sender_pod_info?: {
    pod_key: string;
    alias?: string;
    agent?: { name: string };
  };
  message_type: string;
  body: string;
  content?: MessageContent;
  mentions?: MessageMentions;
  reply_to?: number;
  created_at: string;
}

export interface ChannelMessageEditedData {
  id: number;
  channel_id: number;
  body: string;
  content?: MessageContent;
  mentions?: MessageMentions;
  edited_at: string;
}

export interface ChannelMessageDeletedData {
  id: number;
  channel_id: number;
}

export interface ChannelMemberChangedData {
  channel_id: number;
  user_id: number;
  role?: string;
}
