"use client";

import { useTranslations } from "next-intl";
import { ExternalLink, Bell, BellOff, Pin, PinOff, Circle } from "lucide-react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { openChannelWindow } from "@/lib/windowing";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";

// `children` is the ChannelListItem button (forwardRef + prop-spread), so the
// trigger attaches `asChild` straight onto the focusable <button> — keyboard
// context-menu invocation (Shift+F10) works, unlike a non-focusable wrapper div.
export function ChannelContextMenu({
  channelId,
  isMuted = false,
  isPinned = false,
  children,
}: {
  channelId: number;
  isMuted?: boolean;
  isPinned?: boolean;
  children: React.ReactNode;
}) {
  const t = useTranslations("channels");
  const muteChannel = useChannelMessageStore((s) => s.muteChannel);
  const pinChannel = useChannelMessageStore((s) => s.pinChannel);
  const markUnread = useChannelMessageStore((s) => s.markUnread);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  // is_muted / is_pinned are server-computed Channel fields — refetch the list
  // after a toggle so the row regroups / restyles.
  const handleTogglePin = async () => {
    try {
      await pinChannel(channelId, !isPinned);
      fetchChannels({ includeArchived: true });
    } catch {
      // error already logged by the store action
    }
  };

  const handleToggleMute = async () => {
    try {
      await muteChannel(channelId, !isMuted);
      fetchChannels({ includeArchived: true });
    } catch {
      // error already logged by the store action
    }
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="w-48">
        <ContextMenuItem onClick={() => openChannelWindow(channelId)}>
          <ExternalLink className="mr-2 h-4 w-4" />
          {t("contextMenu.openInNewWindow")}
        </ContextMenuItem>
        <ContextMenuItem onClick={() => { void markUnread(channelId); }}>
          <Circle className="mr-2 h-4 w-4" />
          {t("contextMenu.markUnread")}
        </ContextMenuItem>
        <ContextMenuItem onClick={handleTogglePin}>
          {isPinned ? <PinOff className="mr-2 h-4 w-4" /> : <Pin className="mr-2 h-4 w-4" />}
          {isPinned ? t("contextMenu.unpin") : t("contextMenu.pin")}
        </ContextMenuItem>
        <ContextMenuItem onClick={handleToggleMute}>
          {isMuted ? <Bell className="mr-2 h-4 w-4" /> : <BellOff className="mr-2 h-4 w-4" />}
          {isMuted ? t("contextMenu.unmute") : t("contextMenu.mute")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
