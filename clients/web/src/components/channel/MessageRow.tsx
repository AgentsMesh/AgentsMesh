"use client";

import { useTranslations } from "next-intl";
import { MessageBubble } from "./MessageBubble";
import { ToolCallCard } from "./ToolCallCard";
import { AttachmentCard } from "./AttachmentCard";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { paletteFor } from "@/lib/avatar-palette";
import { type Pod } from "@/stores/pod";
import { cn } from "@/lib/utils";
import type { TransformedMessage } from "./types";
import type { MessageEditPayload } from "@/lib/viewModels/channelMessage";

export function getSenderName(msg: TransformedMessage, allPods: Pod[]): string {
  if (msg.pod) {
    const storePod = allPods.find((p) => p.pod_key === msg.pod!.podKey);
    return getPodDisplayName(storePod ?? {
      pod_key: msg.pod.podKey, alias: msg.pod.alias,
      agent: msg.pod.agent ? { name: msg.pod.agent.name } : undefined,
    });
  }
  if (msg.user) return msg.user.name || msg.user.username || "Unknown";
  return "Unknown";
}

export function formatMessageTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

interface MessageRowProps {
  message: TransformedMessage;
  allPods: Pod[];
  currentUserId?: number;
  channelId?: number | null;
  isFirstInGroup?: boolean;
  role?: string;
  onEditMessage?: (messageId: number, payload: MessageEditPayload) => Promise<void>;
  onDeleteMessage?: (messageId: number) => Promise<void>;
}

/**
 * One message row. When `isFirstInGroup` is false (a merged consecutive message
 * from the same sender) the 28px avatar slot becomes a hover-reveal timestamp
 * gutter and the name/badge header is omitted, so the body stays column-aligned
 * with the group's first row.
 */
export function MessageRow({
  message,
  allPods,
  currentUserId,
  channelId,
  isFirstInGroup = true,
  role,
  onEditMessage,
  onDeleteMessage,
}: MessageRowProps) {
  const t = useTranslations("channels.members");
  if (message.messageType === "system") {
    return (
      <div data-message-id={message.id} className="flex justify-center py-2">
        <span className="text-[11px] text-muted-foreground">{message.body}</span>
      </div>
    );
  }

  const isPod = !!message.pod;
  const senderName = getSenderName(message, allPods);
  const letter = senderName.charAt(0).toUpperCase() || "?";
  const time = formatMessageTime(message.createdAt);
  const avatarBg = paletteFor(isPod ? (message.pod?.podKey ?? "") : (message.user?.id ?? senderName));
  const isToolCall = message.content?.kind === "tool_call";

  return (
    <div
      data-message-id={message.id}
      className={cn("group/msg flex gap-3 px-6 hover:bg-muted/30", isFirstInGroup ? "py-1.5" : "py-0.5")}
    >
      {isFirstInGroup ? (
        <span
          className={cn(
            "flex h-7 w-7 flex-shrink-0 items-center justify-center text-xs font-semibold text-white",
            avatarBg,
            isPod ? "rounded-md font-mono" : "rounded-full",
          )}
        >
          {letter}
        </span>
      ) : (
        <span className="w-7 shrink-0 pt-0.5 text-right text-[10px] leading-5 text-muted-foreground tabular-nums opacity-0 transition-opacity group-hover/msg:opacity-100 motion-reduce:transition-none">
          {time}
        </span>
      )}

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        {isFirstInGroup && (
          <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <span className="text-[13px] font-semibold text-foreground">{senderName}</span>
            {isPod && message.pod?.agent?.name && (
              <span className="rounded border border-border bg-muted px-1.5 py-[1px] font-mono text-[10px] text-muted-foreground">
                {message.pod.agent.name}
              </span>
            )}
            {!isPod && role === "creator" && (
              <span className="rounded border border-success/30 bg-success/10 px-1.5 py-[1px] text-[10px] font-medium text-success">
                {t("owner")}
              </span>
            )}
            <span>{time}</span>
          </div>
        )}

        {isToolCall ? (
          <ToolCallCard content={message.content!} />
        ) : (
          <>
            <MessageBubble
              message={message}
              isFirstInGroup
              formatTime={formatMessageTime}
              currentUserId={currentUserId}
              channelId={channelId}
              onEdit={onEditMessage}
              onDelete={onDeleteMessage}
            />
            {message.content?.attachment_key && (
              <AttachmentCard url={message.content.attachment_key} />
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default MessageRow;
