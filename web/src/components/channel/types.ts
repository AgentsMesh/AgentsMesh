import type { MessageContent, MessageMentions } from "@/lib/api/channel-message-types";

export interface TransformedMessage {
  id: number;
  body: string;
  content?: MessageContent;
  messageType: string;
  mentions?: MessageMentions;
  editedAt?: string;
  createdAt: string;
  pod?: {
    podKey: string;
    alias?: string;
    agent?: { name: string };
  };
  user?: {
    id: number;
    username: string;
    name?: string;
    avatarUrl?: string;
  };
}
