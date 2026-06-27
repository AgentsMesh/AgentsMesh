export interface Channel {
  id: number; organization_id?: number; name: string; description?: string; document?: string;
  is_archived: boolean;
  visibility?: "public" | "private";
  is_member?: boolean;
  is_muted?: boolean;
  is_pinned?: boolean;
  member_count: number;
  agent_count?: number;
  created_at?: string; updated_at?: string;
  repository?: { id: number; name: string };
  ticket?: { id: number; slug: string; title: string };
  pods?: Array<{ pod_key: string; alias?: string; status: string; agent?: { name: string } }>;
}

export interface ChannelLastMessage {
  sender_name: string;
  content_preview: string;
  message_type?: string;
  timestamp: string;
}

export interface ChannelMember {
  channel_id: number;
  user_id: number;
  role: string;
  is_muted: boolean;
  joined_at: string;
}
