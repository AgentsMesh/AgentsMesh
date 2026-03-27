"use client";

import { Terminal, Hash, Users, ChevronRight } from "lucide-react";
import { ChannelDetailView } from "./ChannelDetailView";
import type { ChannelsTabContentProps } from "./types";
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
  onPodsChanged,
  t,
}: ChannelsTabContentProps) {
  // If a channel is selected, show channel detail
  if (selectedChannelId) {
    return (
      <ChannelDetailView
        channelId={selectedChannelId}
        onBack={onBackToList}
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
