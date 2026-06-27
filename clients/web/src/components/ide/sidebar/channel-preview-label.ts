import type { ChannelLastMessage } from "@/stores/channel";

/**
 * Sidebar preview text for a channel's last message. Non-text message types
 * render a localized placeholder ("[Attachment]" / "[附件]" …) — so a sender's
 * image/file/command never shows up as an empty or raw-body preview. Plain text
 * falls through to the truncated content preview produced by the Rust core.
 */
export function previewLabel(
  m: ChannelLastMessage,
  t: (key: string) => string,
): string {
  switch (m.message_type) {
    case "attachment":
      return t("typeAttachment");
    case "code":
      return t("typeCode");
    case "command":
      return t("typeCommand");
    default:
      return m.content_preview ?? "";
  }
}
