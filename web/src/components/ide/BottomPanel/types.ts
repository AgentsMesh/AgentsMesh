import type { ChannelInfo, MeshEdge, MeshTopology } from "@/stores/mesh";

// Re-export from the canonical source to avoid duplication
export type { TransformedMessage } from "@/components/channel/types";

export interface TabContentProps {
  selectedPodKey: string | null;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export interface ChannelsTabContentProps extends TabContentProps {
  podChannels: ChannelInfo[];
  selectedChannelId: number | null;
  onChannelClick: (channelId: number) => void;
  onBackToList: () => void;
  onPodsChanged?: () => void;
}

export interface ActivityTabContentProps extends TabContentProps {
  incomingBindings: MeshEdge[];
  outgoingBindings: MeshEdge[];
  getPodInfo: (podKey: string) => MeshTopology["nodes"][0] | undefined;
}
