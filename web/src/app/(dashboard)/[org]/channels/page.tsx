"use client";

import { useCallback } from "react";
import { useTranslations } from "next-intl";
import { useChannelStore } from "@/stores/channel";
import { ChannelChatPanel, MobileChannelChat } from "@/components/channel";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { ChannelsSidebarContent } from "@/components/ide/sidebar/ChannelsSidebarContent";
import { MessageSquare } from "lucide-react";

export default function ChannelsPage() {
  const t = useTranslations();
  const { isMobile } = useBreakpoint();

  const selectedChannelId = useChannelStore((s) => s.selectedChannelId);
  const setSelectedChannelId = useChannelStore((s) => s.setSelectedChannelId);

  const handleClose = useCallback(() => {
    setSelectedChannelId(null);
  }, [setSelectedChannelId]);

  // Mobile: show channel list when none selected, full-screen chat when selected
  if (isMobile) {
    if (!selectedChannelId) {
      return <ChannelsSidebarContent className="h-full" />;
    }
    return (
      <MobileChannelChat
        channelId={selectedChannelId}
        onClose={handleClose}
      />
    );
  }

  // Desktop: empty state when no channel selected
  if (!selectedChannelId) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center px-8">
        <MessageSquare className="w-12 h-12 text-muted-foreground/30 mb-4" />
        <h2 className="text-lg font-medium text-foreground mb-1">
          {t("channels.emptyState")}
        </h2>
        <p className="text-sm text-muted-foreground max-w-md">
          {t("channels.emptyStateHint")}
        </p>
      </div>
    );
  }

  // Desktop: chat panel
  return (
    <div className="h-full w-full">
      <ChannelChatPanel
        channelId={selectedChannelId}
        onClose={handleClose}
      />
    </div>
  );
}
