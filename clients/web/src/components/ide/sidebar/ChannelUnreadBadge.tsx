"use client";

import { useTranslations } from "next-intl";
import { BellOff } from "lucide-react";

interface ChannelUnreadBadgeProps {
  unreadCount: number;
  mentionCount?: number;
  manuallyUnread?: boolean;
  isMuted?: boolean;
  isSelected?: boolean;
}

/**
 * Sidebar unread indicator. Priority (Feishu-style): a channel that @-mentions
 * the user shows a red count even when muted; a muted channel shows a gray dot;
 * a manually-marked-unread channel with no real count shows a red dot; otherwise
 * a red count. The slot keeps a fixed min width so toggling never reflows the row.
 */
export function ChannelUnreadBadge({
  unreadCount,
  mentionCount = 0,
  manuallyUnread = false,
  isMuted = false,
  isSelected = false,
}: ChannelUnreadBadgeProps) {
  const t = useTranslations("channels.sidebar");
  const active = !isSelected;
  const isMention = mentionCount > 0;
  const signal = active && (unreadCount > 0 || manuallyUnread);
  // Gate on unreadCount > 0 so a stranded mention (mention>0, unread cleared)
  // never paints a red "0"; mention still pierces mute via the isMention term.
  const showRedBadge = unreadCount > 0 && (isMention || (signal && !isMuted));
  const showGrayDot = signal && isMuted && !isMention;
  const showRedDot = signal && unreadCount === 0 && !isMuted && !isMention;
  const dotAria = unreadCount > 0 ? t("unreadBadge", { count: unreadCount }) : t("markedUnread");

  return (
    <span className="flex min-w-[22px] shrink-0 items-center justify-end gap-1">
      {isMuted && <BellOff className="h-3 w-3 text-muted-foreground/50" aria-label={t("muted")} />}
      {showRedBadge && (
        <span
          data-testid="channel-unread-badge"
          aria-label={t("unreadBadge", { count: unreadCount })}
          className="flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium leading-none text-destructive-foreground tabular-nums"
        >
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}
      {showGrayDot && (
        <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" aria-label={dotAria} />
      )}
      {showRedDot && (
        <span className="h-1.5 w-1.5 rounded-full bg-destructive" aria-label={dotAria} />
      )}
    </span>
  );
}

export default ChannelUnreadBadge;
