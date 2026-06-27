"use client";

import { useTranslations } from "next-intl";

/**
 * Feishu-style "new messages" separator rendered before the first unread
 * message. `data-unread-anchor` is the scroll target the message list jumps to
 * on channel entry.
 */
export function UnreadDivider() {
  const t = useTranslations("channels.messages");
  return (
    <div
      role="separator"
      data-unread-anchor
      aria-label={t("newMessagesDivider")}
      className="flex scroll-mt-4 items-center gap-2 px-6 py-2"
    >
      <span className="h-px flex-1 bg-destructive/30" />
      <span className="rounded-full border border-destructive/30 bg-destructive/10 px-2 py-0.5 text-[11px] font-medium text-destructive">
        {t("newMessagesDivider")}
      </span>
      <span className="h-px flex-1 bg-destructive/30" />
    </div>
  );
}

export default UnreadDivider;
