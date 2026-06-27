"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import {
  useChannelStore,
  useChannels,
  useChannelMessageStore,
  useUnreadCounts,
  useMentionCounts,
  useManuallyUnread,
} from "@/stores/channel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Search, Loader2, MessageSquare, RefreshCw } from "lucide-react";
import { ChannelListItem } from "./ChannelListItem";
import { ChannelContextMenu } from "./ChannelContextMenu";
import { useChannelGroups, type GroupedRow } from "./useChannelGroups";

interface ChannelsSidebarContentProps {
  className?: string;
}

function SectionLabel({ children, count }: { children: string; count?: number }) {
  return (
    <div className="flex items-baseline justify-between px-4 pt-3 pb-1.5">
      <span className="text-[11px] font-medium text-muted-foreground">
        {children}
      </span>
      {typeof count === "number" && count > 0 && (
        <span className="font-mono text-[10px] text-muted-foreground">{count}</span>
      )}
    </div>
  );
}

export function ChannelsSidebarContent({ className }: ChannelsSidebarContentProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();

  const channels = useChannels();
  const loading = useChannelStore((s) => s.loading);
  const selectedChannelId = useChannelStore((s) => s.selectedChannelId);
  const searchQuery = useChannelStore((s) => s.searchQuery);
  const showArchived = useChannelStore((s) => s.showArchived);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);
  const setSelectedChannelId = useChannelStore((s) => s.setSelectedChannelId);
  const setSearchQuery = useChannelStore((s) => s.setSearchQuery);
  const setShowArchived = useChannelStore((s) => s.setShowArchived);
  const _tick = useChannelStore((s) => s._tick);

  const unreadCounts = useUnreadCounts();
  const mentionCounts = useMentionCounts();
  const manuallyUnread = useManuallyUnread();
  const fetchUnreadCounts = useChannelMessageStore((s) => s.fetchUnreadCounts);

  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    if (currentOrg) {
      fetchChannels({ includeArchived: true });
      fetchUnreadCounts();
    }
  }, [currentOrg, fetchChannels, fetchUnreadCounts]);

  const visible = useMemo(() => {
    return channels.filter((channel) => {
      if (!showArchived && channel.is_archived) return false;
      if (!searchQuery && !channel.is_member) return false;
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const matchesName = channel.name.toLowerCase().includes(query);
        const matchesDesc = channel.description?.toLowerCase().includes(query);
        if (!matchesName && !matchesDesc) return false;
      }
      return true;
    });
  }, [channels, searchQuery, showArchived]);

  const grouped = useChannelGroups(visible, _tick);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchChannels({ includeArchived: true });
    } finally {
      setRefreshing(false);
    }
  }, [fetchChannels]);

  const renderGroup = (label: string, rows: GroupedRow[]) => {
    if (rows.length === 0) return null;
    return (
      <>
        <SectionLabel count={rows.length}>{label}</SectionLabel>
        <div className="flex flex-col gap-0.5 px-2">
          {rows.map(({ channel, lastMsg }) => (
            <ChannelContextMenu
              key={channel.id}
              channelId={channel.id}
              isMuted={channel.is_muted}
              isPinned={channel.is_pinned}
            >
              <ChannelListItem
                channel={channel}
                isSelected={selectedChannelId === channel.id}
                unreadCount={unreadCounts[channel.id] || 0}
                mentionCount={mentionCounts[channel.id] || 0}
                manuallyUnread={manuallyUnread[channel.id] ?? false}
                isMuted={channel.is_muted}
                lastMessage={lastMsg}
                onClick={() => setSelectedChannelId(channel.id)}
              />
            </ChannelContextMenu>
          ))}
        </div>
      </>
    );
  };

  return (
    <div className={cn("flex h-full flex-col", className)}>
      <div className="flex flex-col gap-2 px-3 pb-2 pt-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("channels.sidebar.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 pl-8 text-[13px]"
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && channels.length === 0 ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : visible.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-4 py-8 text-center">
            <MessageSquare className="mb-2 h-8 w-8 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              {searchQuery
                ? t("channels.sidebar.noMatch")
                : t("channels.sidebar.noChannels")}
            </p>
          </div>
        ) : (
          <div className="pb-3">
            {renderGroup(t("channels.sidebar.groupPinned"), grouped.pinned)}
            {renderGroup(t("channels.sidebar.groupActive"), grouped.active)}
            {renderGroup(t("channels.sidebar.groupLinked"), grouped.linked)}
            {renderGroup(t("channels.sidebar.groupQuiet"), grouped.quiet)}
          </div>
        )}
      </div>

      <div className="flex items-center justify-between border-t border-border/60 px-3 py-2.5 text-[12px]">
        <button
          type="button"
          onClick={() => setShowArchived(!showArchived)}
          className="text-primary hover:underline"
        >
          {showArchived
            ? t("channels.sidebar.hideArchived")
            : t("channels.sidebar.showArchived")}
        </button>
        <Button
          size="sm"
          variant="ghost"
          className="h-6 w-6 p-0 text-muted-foreground"
          onClick={handleRefresh}
          disabled={refreshing}
          title={t("channels.sidebar.refresh")}
        >
          <RefreshCw className={cn("h-3.5 w-3.5", refreshing && "animate-spin")} />
        </Button>
      </div>
    </div>
  );
}

export default ChannelsSidebarContent;
