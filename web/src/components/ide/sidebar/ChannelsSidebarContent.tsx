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
  getLastMessage,
  type Channel,
  type ChannelLastMessage,
} from "@/stores/channel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Search, Plus, Loader2, MessageSquare, RefreshCw } from "lucide-react";
import { ChannelListItem } from "./ChannelListItem";
import { CreateChannelDialog } from "@/components/channel";

interface ChannelsSidebarContentProps {
  className?: string;
}

/**
 * Activity-weighted group the channel belongs to. Mirrors the design's
 * "Active / Linked / Quiet" sections:
 *   - Active: has any message in the last 24h
 *   - Linked: tied to a ticket or repository (and not Active)
 *   - Quiet:  everything else
 */
type ChannelGroup = "active" | "linked" | "quiet";
const DAY_MS = 24 * 60 * 60 * 1000;

function classifyChannel(
  channel: Channel,
  lastMsg: ChannelLastMessage | null,
  now: number,
): ChannelGroup {
  const ts = lastMsg?.timestamp ?? channel.updated_at;
  if (ts) {
    const t = new Date(ts).getTime();
    if (!Number.isNaN(t) && now - t < DAY_MS) return "active";
  }
  if (channel.ticket || channel.repository) return "linked";
  return "quiet";
}

function SectionLabel({ children, count }: { children: string; count?: number }) {
  return (
    <div className="flex items-baseline justify-between px-4 pt-3 pb-1.5">
      <span className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
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
  const fetchUnreadCounts = useChannelMessageStore((s) => s.fetchUnreadCounts);

  const [showCreateDialog, setShowCreateDialog] = useState(false);
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

  const grouped = useMemo(() => {
    const now = Date.now();
    const rows = visible.map((ch) => {
      const lastMsg = getLastMessage(ch.id);
      return { channel: ch, lastMsg, group: classifyChannel(ch, lastMsg, now) };
    });
    rows.sort((a, b) => {
      const ta = a.lastMsg?.timestamp ?? a.channel.updated_at ?? "";
      const tb = b.lastMsg?.timestamp ?? b.channel.updated_at ?? "";
      return tb.localeCompare(ta);
    });
    return {
      active: rows.filter((r) => r.group === "active"),
      linked: rows.filter((r) => r.group === "linked"),
      quiet: rows.filter((r) => r.group === "quiet"),
    };
    // `_tick` is the WASM store invalidator — re-derive on any channel event.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, _tick]);

  const handleChannelCreated = useCallback(
    (channelId: number) => {
      setShowCreateDialog(false);
      setSelectedChannelId(channelId);
    },
    [setSelectedChannelId],
  );

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchChannels({ includeArchived: true });
    } finally {
      setRefreshing(false);
    }
  }, [fetchChannels]);

  const renderGroup = (label: string, rows: typeof grouped.active) => {
    if (rows.length === 0) return null;
    return (
      <>
        <SectionLabel count={rows.length}>{label}</SectionLabel>
        <div className="flex flex-col gap-0.5 px-2">
          {rows.map(({ channel, lastMsg }) => (
            <ChannelListItem
              key={channel.id}
              channel={channel}
              isSelected={selectedChannelId === channel.id}
              unreadCount={unreadCounts[channel.id] || 0}
              lastMessage={lastMsg}
              onClick={() => setSelectedChannelId(channel.id)}
            />
          ))}
        </div>
      </>
    );
  };

  return (
    <div className={cn("flex h-full flex-col", className)}>
      {/* Search + CTA */}
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
        <Button
          size="sm"
          onClick={() => setShowCreateDialog(true)}
          className="h-8 w-full gap-1.5 text-[13px]"
        >
          <Plus className="h-3.5 w-3.5" />
          {t("channels.sidebar.createChannel")}
        </Button>
      </div>

      {/* Groups */}
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
            {renderGroup(t("channels.sidebar.groupActive"), grouped.active)}
            {renderGroup(t("channels.sidebar.groupLinked"), grouped.linked)}
            {renderGroup(t("channels.sidebar.groupQuiet"), grouped.quiet)}
          </div>
        )}
      </div>

      {/* Footer: archive toggle + refresh */}
      <div className="flex items-center justify-between border-t border-border px-3 py-2.5 text-[12px]">
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

      <CreateChannelDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onCreated={handleChannelCreated}
      />
    </div>
  );
}

export default ChannelsSidebarContent;
