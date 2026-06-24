"use client";

import { useTranslations } from "next-intl";
import { ExternalLink } from "lucide-react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { openChannelWindow } from "@/lib/windowing";

// `children` is the ChannelListItem button (forwardRef + prop-spread), so the
// trigger attaches `asChild` straight onto the focusable <button> — keyboard
// context-menu invocation (Shift+F10) works, unlike a non-focusable wrapper div.
export function ChannelContextMenu({
  channelId,
  children,
}: {
  channelId: number;
  children: React.ReactNode;
}) {
  const t = useTranslations("channels");
  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="w-48">
        <ContextMenuItem onClick={() => openChannelWindow(channelId)}>
          <ExternalLink className="mr-2 h-4 w-4" />
          {t("contextMenu.openInNewWindow")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
