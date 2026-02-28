import type { ChannelInfo, MeshEdge, MeshTopology } from "@/stores/mesh";
import type { ChannelMessage } from "@/lib/api";
import type { MentionedPod } from "@/components/mesh/MessageInput";

/**
 * Common props for tab content components
 */
export interface TabContentProps {
  selectedPodKey: string | null;
  t: (key: string, params?: Record<string, string | number>) => string;
}

/**
 * Props for ChannelsTabContent
 */
export interface ChannelsTabContentProps extends TabContentProps {
  podChannels: ChannelInfo[];
  selectedChannelId: number | null;
  onChannelClick: (channelId: number) => void;
  onBackToList: () => void;
  // Channel detail props
  topology: MeshTopology | null;
  currentChannel: {
    name?: string;
    description?: string;
    document?: string;
    pods?: { pod_key: string }[];
  } | null;
  messages: ChannelMessage[];
  messagesLoading: boolean;
  onSendMessage: (content: string, mentionedPods?: MentionedPod[]) => Promise<void>;
  onLoadMore: () => void;
  onRefresh: () => void;
}

/**
 * Props for ActivityTabContent
 */
export interface ActivityTabContentProps extends TabContentProps {
  incomingBindings: MeshEdge[];
  outgoingBindings: MeshEdge[];
  getPodInfo: (podKey: string) => MeshTopology["nodes"][0] | undefined;
}

/**
 * Transformed message for MessageList component
 */
export interface TransformedMessage {
  id: number;
  content: string;
  messageType: "text" | "system" | "code" | "command";
  metadata?: Record<string, unknown>;
  createdAt: string;
  pod?: {
    podKey: string;
    agentType?: { name: string };
  };
  user?: {
    id: number;
    username: string;
    name?: string;
    avatarUrl?: string;
  };
}
