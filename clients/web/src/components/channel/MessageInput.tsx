"use client";

import { useState, useRef, type KeyboardEvent } from "react";
import { useTranslations } from "next-intl";
import { clearChannelDraft } from "@/stores/useChannelDraft";
import { MentionDropdown } from "./MentionDropdown";
import { MessageInputToolbar } from "./MessageInputToolbar";
import { useFileAttachment } from "@/hooks/useFileAttachment";
import { useChannelDraftPersistence } from "./useChannelDraftPersistence";
import { useMentionAutocomplete } from "./useMentionAutocomplete";
import type { MessageSendPayload } from "@/lib/viewModels/channelMessage";

interface MessageInputProps {
  onSend: (payload: MessageSendPayload) => void;
  disabled?: boolean;
  placeholder?: string;
  channelId?: number | null;
  channelName?: string;
}

export function MessageInput({ onSend, disabled, placeholder, channelId, channelName }: MessageInputProps) {
  const t = useTranslations();
  const defaultPlaceholder = placeholder
    || (channelName
      ? t("channels.composer.placeholder", { channel: channelName })
      : t("mesh.messageInput.placeholder"));
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const attachment = useFileAttachment();

  useChannelDraftPersistence(channelId, content, setContent);
  const mention = useMentionAutocomplete({ channelId, content, setContent, textareaRef, containerRef });

  const handleSend = () => {
    const trimmed = content.trim();
    if (!trimmed && !attachment.key) return;
    if (disabled) return;

    const mentions = mention.getMentions();
    const payload: MessageSendPayload = { source: trimmed };
    if (Object.keys(mentions).length > 0) payload.mentions = mentions;
    if (attachment.key) payload.attachment_key = attachment.key;

    onSend(payload);
    setContent("");
    if (channelId != null) clearChannelDraft(channelId);
    mention.reset();
    attachment.clear();

    if (textareaRef.current) textareaRef.current.style.height = "auto";
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return;
    if (mention.handleKeyDown(e)) return;
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleInput = () => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
    }
  };

  return (
    <div className="border-t border-border px-4 py-3" ref={containerRef}>
      <div className="relative rounded-lg border border-border bg-background">
        <MentionDropdown
          items={mention.filtered}
          activeIndex={mention.safeActiveIndex}
          onSelect={mention.handleSelect}
          position={mention.position}
          visible={mention.visible}
        />

        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => mention.handleChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onInput={handleInput}
          placeholder={defaultPlaceholder}
          aria-label={defaultPlaceholder}
          disabled={disabled}
          className="block w-full resize-none border-0 bg-transparent px-3 pt-2.5 pb-1 text-[13px] focus:outline-none disabled:opacity-50 min-h-[42px] max-h-[200px]"
          rows={1}
          data-testid="message-input-textarea"
        />

        <MessageInputToolbar
          textareaRef={textareaRef}
          value={content}
          onChange={mention.handleChange}
          onSend={handleSend}
          disabled={disabled}
          attachment={attachment}
          onMention={mention.triggerMention}
        />
      </div>
    </div>
  );
}

export default MessageInput;
