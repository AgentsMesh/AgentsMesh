"use client";

import { cn } from "@/lib/utils";
import type { Channel } from "@/stores/channel";
import { Hash, Archive } from "lucide-react";

interface ChannelListItemProps {
  channel: Channel;
  isSelected: boolean;
  unreadCount?: number;
  onClick: () => void;
}

/**
 * Individual channel item in the sidebar list.
 * Shows active pod indicator (green dot) when channel has running pods.
 */
export function ChannelListItem({ channel, isSelected, unreadCount = 0, onClick }: ChannelListItemProps) {
  const runningPodCount =
    channel.pods?.filter((p) => p.status === "running" || p.status === "initializing").length ?? 0;
  const hasActivePods = runningPodCount > 0;

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-2 cursor-pointer rounded-md mx-1 transition-colors",
        "hover:bg-muted/50",
        isSelected && "bg-muted"
      )}
      onClick={onClick}
    >
      <div className="relative shrink-0">
        <Hash className="w-4 h-4 text-muted-foreground" />
        {/* Green dot for active pods */}
        {hasActivePods && (
          <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-green-500 rounded-full ring-1 ring-background" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium truncate">{channel.name}</span>
          {channel.is_archived && (
            <Archive className="w-3 h-3 text-muted-foreground shrink-0" />
          )}
        </div>
        {channel.description && (
          <p className="text-xs text-muted-foreground truncate mt-0.5">
            {channel.description}
          </p>
        )}
      </div>
      {/* Unread badge */}
      {unreadCount > 0 && !isSelected && (
        <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold rounded-full bg-primary text-primary-foreground shrink-0">
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}
      {channel.pods && channel.pods.length > 0 && (
        <span
          className={cn(
            "text-xs px-1.5 py-0.5 rounded-full shrink-0",
            hasActivePods
              ? "text-green-700 dark:text-green-400 bg-green-500/10"
              : "text-muted-foreground bg-muted/50"
          )}
        >
          {channel.pods.length}
        </span>
      )}
    </div>
  );
}

export default ChannelListItem;
