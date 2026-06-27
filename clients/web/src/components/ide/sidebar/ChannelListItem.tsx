"use client";

import { forwardRef, type ComponentPropsWithoutRef } from "react";
import { cn } from "@/lib/utils";
import type { Channel, ChannelLastMessage } from "@/stores/channel";
import { formatRelativeShort } from "@/lib/format-relative-time";
import { useTranslations } from "next-intl";
import { Lock } from "lucide-react";
import { ChannelUnreadBadge } from "./ChannelUnreadBadge";
import { previewLabel } from "./channel-preview-label";
import { useChannelHasDraft } from "@/stores/useChannelDraft";

// Extends button props so a ContextMenuTrigger `asChild` can forward its ref +
// onContextMenu onto the focusable <button> directly (keyboard Shift+F10 works).
interface ChannelListItemProps extends Omit<ComponentPropsWithoutRef<"button">, "onClick"> {
  channel: Channel;
  isSelected: boolean;
  unreadCount?: number;
  mentionCount?: number;
  manuallyUnread?: boolean;
  isMuted?: boolean;
  lastMessage?: ChannelLastMessage | null;
  onClick?: () => void;
}

/**
 * Channel row in the sidebar list — matches design/desktop/pages/channels.pastel
 * `channel_row`: hash + name + last message preview + short time + unread badge.
 * A red `@me` prefix marks unread @-mentions (highest-priority signal). Private
 * channels use the lock icon in place of #.
 */
export const ChannelListItem = forwardRef<HTMLButtonElement, ChannelListItemProps>(function ChannelListItem(
  {
    channel,
    isSelected,
    unreadCount = 0,
    mentionCount = 0,
    manuallyUnread = false,
    isMuted = false,
    lastMessage,
    onClick,
    className,
    ...rest
  },
  ref,
) {
  const t = useTranslations("channels.sidebar");
  const hasDraft = useChannelHasDraft(channel.id);
  const isPrivate = channel.visibility === "private";
  const previewBody = lastMessage ? previewLabel(lastMessage, t) : channel.description ?? "";
  const preview = lastMessage?.sender_name
    ? `${lastMessage.sender_name}: ${previewBody}`
    : previewBody;
  const time = formatRelativeShort(lastMessage?.timestamp ?? channel.updated_at);
  const showMention = mentionCount > 0 && !isSelected;

  return (
    <button
      ref={ref}
      type="button"
      onClick={onClick}
      data-testid="channel-list-item"
      data-channel-id={String(channel.id)}
      className={cn(
        "group flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left transition-colors",
        isSelected ? "bg-muted" : "hover:bg-muted/50",
        className,
      )}
      {...rest}
    >
      <span
        className={cn(
          "shrink-0 font-mono text-[14px]",
          isSelected ? "font-semibold text-foreground" : "text-muted-foreground/70",
        )}
      >
        {isPrivate ? <Lock className="h-3.5 w-3.5" /> : "#"}
      </span>

      <span className="min-w-0 flex-1 flex flex-col gap-0.5">
        <span className="flex items-center justify-between gap-2">
          <span
            className={cn(
              "truncate text-[13px]",
              isSelected ? "font-semibold text-foreground" : "text-foreground",
            )}
          >
            {channel.name}
          </span>
          {time && (
            <span className="shrink-0 text-[10px] text-muted-foreground/70">{time}</span>
          )}
        </span>
        {(preview || showMention || hasDraft) && (
          <span className="flex min-w-0 items-center truncate text-[11px] text-muted-foreground/70">
            {showMention ? (
              <span className="mr-1 shrink-0 font-medium text-destructive">{t("atMe")}</span>
            ) : hasDraft ? (
              <span className="mr-1 shrink-0 font-medium text-destructive">{t("draftTag")}</span>
            ) : null}
            <span className="truncate">{preview}</span>
          </span>
        )}
      </span>

      <ChannelUnreadBadge
        unreadCount={unreadCount}
        mentionCount={mentionCount}
        manuallyUnread={manuallyUnread}
        isMuted={isMuted}
        isSelected={isSelected}
      />
    </button>
  );
});

export default ChannelListItem;
