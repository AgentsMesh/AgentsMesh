"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Search,
  RefreshCw,
  Plus,
  Archive,
  Loader2,
  MessageSquare,
} from "lucide-react";
import { ChannelListItem } from "./ChannelListItem";
import { CreateChannelDialog } from "@/components/channel";

interface ChannelsSidebarContentProps {
  className?: string;
}

export function ChannelsSidebarContent({ className }: ChannelsSidebarContentProps) {
  const t = useTranslations();
  const currentOrg = useAuthStore((s) => s.currentOrg);

  const channels = useChannelStore((s) => s.channels);
  const loading = useChannelStore((s) => s.loading);
  const selectedChannelId = useChannelStore((s) => s.selectedChannelId);
  const searchQuery = useChannelStore((s) => s.searchQuery);
  const showArchived = useChannelStore((s) => s.showArchived);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);
  const setSelectedChannelId = useChannelStore((s) => s.setSelectedChannelId);
  const setSearchQuery = useChannelStore((s) => s.setSearchQuery);
  const setShowArchived = useChannelStore((s) => s.setShowArchived);

  const unreadCounts = useChannelMessageStore((s) => s.unreadCounts);
  const fetchUnreadCounts = useChannelMessageStore((s) => s.fetchUnreadCounts);

  const [refreshing, setRefreshing] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  // Load channels and unread counts on mount
  useEffect(() => {
    if (currentOrg) {
      fetchChannels({ includeArchived: true });
      fetchUnreadCounts();
    }
  }, [currentOrg, fetchChannels, fetchUnreadCounts]);

  // Filter channels by search query and archived status
  const filteredChannels = useMemo(() => {
    return channels.filter((channel) => {
      if (!showArchived && channel.is_archived) return false;
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const matchesName = channel.name.toLowerCase().includes(query);
        const matchesDesc = channel.description?.toLowerCase().includes(query);
        if (!matchesName && !matchesDesc) return false;
      }
      return true;
    });
  }, [channels, searchQuery, showArchived]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchChannels({ includeArchived: true });
    } finally {
      setRefreshing(false);
    }
  }, [fetchChannels]);

  const handleChannelCreated = useCallback((channelId: number) => {
    setShowCreateDialog(false);
    setSelectedChannelId(channelId);
  }, [setSelectedChannelId]);

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder={t("channels.sidebar.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      {/* Actions bar */}
      <div className="flex items-center justify-between px-2 pb-2">
        <Button
          size="sm"
          variant="outline"
          className="h-7 text-xs gap-1"
          onClick={() => setShowCreateDialog(true)}
        >
          <Plus className="w-3.5 h-3.5" />
          {t("channels.sidebar.createChannel")}
        </Button>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            className={cn("h-7 w-7 p-0", showArchived && "text-primary")}
            onClick={() => setShowArchived(!showArchived)}
            title={showArchived ? t("channels.sidebar.hideArchived") : t("channels.sidebar.showArchived")}
          >
            <Archive className="w-3.5 h-3.5" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0"
            onClick={handleRefresh}
            disabled={refreshing}
            title={t("channels.sidebar.refresh")}
          >
            <RefreshCw className={cn("w-3.5 h-3.5", refreshing && "animate-spin")} />
          </Button>
        </div>
      </div>

      {/* Channel list */}
      <div className="flex-1 overflow-y-auto border-t border-border">
        {loading && channels.length === 0 ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : filteredChannels.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 px-4 text-center">
            <MessageSquare className="w-8 h-8 text-muted-foreground/50 mb-2" />
            <p className="text-sm text-muted-foreground">
              {searchQuery ? t("channels.sidebar.noMatch") : t("channels.sidebar.noChannels")}
            </p>
          </div>
        ) : (
          <div className="py-1">
            {filteredChannels.map((channel) => (
              <ChannelListItem
                key={channel.id}
                channel={channel}
                isSelected={selectedChannelId === channel.id}
                unreadCount={unreadCounts[channel.id] || 0}
                onClick={() => setSelectedChannelId(channel.id)}
              />
            ))}
          </div>
        )}
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
