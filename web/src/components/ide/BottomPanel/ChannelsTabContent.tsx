"use client";

import { Terminal, Hash, Users, ChevronRight } from "lucide-react";
import { ChannelDetailView } from "./ChannelDetailView";
import type { ChannelsTabContentProps, TransformedMessage } from "./types";
import type { ChannelInfo } from "@/stores/mesh";

/**
 * Channels tab content - shows channel list or channel detail
 */
export function ChannelsTabContent({
  selectedPodKey,
  podChannels,
  selectedChannelId,
  onChannelClick,
  onBackToList,
  topology,
  currentChannel,
  messages,
  messagesLoading,
  onSendMessage,
  onLoadMore,
  onRefresh,
  onPodsChanged,
  t,
}: ChannelsTabContentProps) {
  // Transform messages for MessageList component
  const transformedMessages: TransformedMessage[] = messages.map((msg) => ({
    id: msg.id,
    content: msg.content,
    messageType: msg.message_type as "text" | "system" | "code" | "command",
    metadata: msg.metadata,
    createdAt: msg.created_at,
    pod: msg.sender_pod_info
      ? {
          podKey: msg.sender_pod_info.pod_key,
          agentType: msg.sender_pod_info.agent_type
            ? { name: msg.sender_pod_info.agent_type.name }
            : undefined,
        }
      : undefined,
    user: msg.sender_user
      ? {
          id: msg.sender_user.id,
          username: msg.sender_user.username,
          name: msg.sender_user.name,
          avatarUrl: msg.sender_user.avatar_url,
        }
      : undefined,
  }));

  // If a channel is selected, show channel detail
  if (selectedChannelId) {
    return (
      <ChannelDetailView
        channelId={selectedChannelId}
        topology={topology}
        currentChannel={currentChannel}
        messages={transformedMessages}
        messagesLoading={messagesLoading}
        onBack={onBackToList}
        onSendMessage={onSendMessage}
        onLoadMore={onLoadMore}
        onRefresh={onRefresh}
        onPodsChanged={onPodsChanged}
        t={t}
      />
    );
  }

  // No pod selected
  if (!selectedPodKey) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
        <Terminal className="w-4 h-4 mr-2" />
        <span>{t("ide.bottomPanel.selectPodFirst")}</span>
      </div>
    );
  }

  // No channels for pod
  if (podChannels.length === 0) {
    return (
      <div className="text-xs text-muted-foreground">
        <p>{t("ide.bottomPanel.noChannels")}</p>
      </div>
    );
  }

  // Channel list
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground mb-2">
        {t("ide.bottomPanel.podChannels", { count: podChannels.length })}
      </p>
      <div className="space-y-1">
        {podChannels.map((channel: ChannelInfo) => (
          <button
            key={channel.id}
            className="w-full flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 hover:bg-muted transition-colors cursor-pointer text-left"
            onClick={() => onChannelClick(channel.id)}
          >
            <Hash className="w-3.5 h-3.5 text-muted-foreground" />
            <span className="text-xs font-medium flex-1">{channel.name}</span>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Users className="w-3 h-3" />
              <span>{t("ide.bottomPanel.members")}: {channel.pod_keys.length}</span>
            </div>
            <ChevronRight className="w-3 h-3 text-muted-foreground" />
          </button>
        ))}
      </div>
    </div>
  );
}

export default ChannelsTabContent;
