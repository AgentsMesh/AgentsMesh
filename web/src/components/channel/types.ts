/**
 * Shared message types for channel components.
 * Extracted from the inline `Message` interface in MessageList.tsx
 * to enable reuse across MessageList, MessageBubble, and useChannelChat hook.
 */

/** Backend-normalized message for rendering */
export interface TransformedMessage {
  id: number;
  content: string;
  messageType: "text" | "system" | "code" | "command";
  metadata?: Record<string, unknown>;
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
