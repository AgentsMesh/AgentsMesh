"use client";

/**
 * Shared hook for channel chat business logic.
 * Single source of truth for all channel message consumers:
 * ChannelChatPanel, MobileChannelChat, ChannelDetailView (BottomPanel).
 */

import { useEffect, useCallback, useMemo, useRef } from "react";
import { useCurrentUser, useAuthStore } from "@/stores/auth";
import { useChannelStore, useChannelMessageStore, useCurrentChannel } from "@/stores/channel";
import { EMPTY_CACHE, LOAD_MORE_MESSAGE_LIMIT, useChannelMessages } from "@/stores/channelMessageStore";
import { useMeshStore, useTopology } from "@/stores/mesh";
import { transformMessage } from "@/components/channel/transformMessage";
import type { TransformedMessage } from "@/components/channel/types";
import type { MessageContent } from "@/lib/api/channel-message-types";

interface UseChannelChatOptions {
  channelId: number;
}

interface UseChannelChatReturn {
  currentChannel: ReturnType<typeof useCurrentChannel>;
  channelLoading: boolean;
  messagesLoading: boolean;
  loadingMore: boolean;
  messagesError: string | null;
  podCount: number;
  channelName: string;
  transformedMessages: TransformedMessage[];
  hasMore: boolean;
  currentUserId: number | undefined;
  handlePodsChanged: () => void;
  handleSendMessage: (content: MessageContent) => Promise<void>;
  handleEditMessage: (messageId: number, content: MessageContent) => Promise<void>;
  handleDeleteMessage: (messageId: number) => Promise<void>;
  handleLoadMore: () => void;
  handleRefresh: () => void;
}

export function useChannelChat({ channelId }: UseChannelChatOptions): UseChannelChatReturn {
  const currentUserId = useCurrentUser()?.id;

  // Prefer WASM-derived current channel so detail loads through fetchChannel
  // immediately surface to the header. The JS-side `currentChannel` is only
  // updated by setCurrentChannel() and stays null on the select→fetch path.
  const currentChannel = useCurrentChannel();
  const channelLoading = useChannelStore((s) => s.channelLoading);
  const fetchChannel = useChannelStore((s) => s.fetchChannel);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);

  // Per-channel UI state (loading/error) — Rust is SSOT for messages/hasMore.
  const channelCache = useChannelMessageStore(
    (s) => s.cache[channelId] ?? EMPTY_CACHE
  );
  const { loading: messagesLoading, loadingMore, error: messagesError } = channelCache;
  const { messages, hasMore } = useChannelMessages(channelId);

  const fetchMessages = useChannelMessageStore((s) => s.fetchMessages);
  const sendMessage = useChannelMessageStore((s) => s.sendMessage);
  const editMessage = useChannelMessageStore((s) => s.editMessage);
  const deleteMessage = useChannelMessageStore((s) => s.deleteMessage);
  const markRead = useChannelMessageStore((s) => s.markRead);

  const topology = useTopology();
  const fetchTopology = useMeshStore((s) => s.fetchTopology);

  // Load channel and messages when channelId changes
  useEffect(() => {
    if (channelId) {
      fetchChannel(channelId);
      fetchMessages(channelId);
    }
    return () => {
      setCurrentChannel(null);
    };
  }, [channelId, fetchChannel, fetchMessages, setCurrentChannel]);

  // Auto mark-as-read: debounced to avoid excessive API calls when messages stream in
  const lastMessageId = messages.length > 0 ? messages[messages.length - 1].id : null;
  const prevLastMsgIdRef = useRef<number | null>(null);
  const markReadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastMessageIdRef = useRef(lastMessageId);
  lastMessageIdRef.current = lastMessageId;

  useEffect(() => {
    if (lastMessageId !== null && lastMessageId !== prevLastMsgIdRef.current) {
      prevLastMsgIdRef.current = lastMessageId;
      if (markReadTimerRef.current) clearTimeout(markReadTimerRef.current);
      markReadTimerRef.current = setTimeout(() => {
        markRead(channelId, lastMessageId);
      }, 300);
    }
    return () => {
      if (markReadTimerRef.current) clearTimeout(markReadTimerRef.current);
    };
  }, [lastMessageId, channelId, markRead]);

  // Flush pending markRead on unmount to prevent debounce loss
  useEffect(() => {
    return () => {
      if (markReadTimerRef.current) {
        clearTimeout(markReadTimerRef.current);
        if (lastMessageIdRef.current !== null) {
          markRead(channelId, lastMessageIdRef.current);
        }
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channelId]);

  // Derive pod count and channel name from topology + currentChannel
  const channelInfo = topology?.channels.find((c: { id: number }) => c.id === channelId);
  const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;
  const channelName = currentChannel?.name || channelInfo?.name || "Channel";

  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    fetchChannel(channelId);
  }, [fetchTopology, fetchChannel, channelId]);

  const handleSendMessage = useCallback(
    async (content: MessageContent) => {
      try {
        await sendMessage(channelId, content);
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [channelId, sendMessage]
  );

  const handleEditMessage = useCallback(
    async (messageId: number, content: MessageContent) => {
      await editMessage(channelId, messageId, content);
    },
    [channelId, editMessage]
  );

  const handleDeleteMessage = useCallback(
    async (messageId: number) => {
      await deleteMessage(channelId, messageId);
    },
    [channelId, deleteMessage]
  );

  const handleLoadMore = useCallback(() => {
    // Guard: prevent concurrent requests and unnecessary calls
    if (loadingMore || !hasMore || messages.length === 0) return;
    fetchMessages(channelId, LOAD_MORE_MESSAGE_LIMIT, messages[0].id);
  }, [channelId, messages, fetchMessages, loadingMore, hasMore]);

  const handleRefresh = useCallback(() => {
    fetchMessages(channelId);
  }, [channelId, fetchMessages]);

  // Transform raw store messages into rendering-ready format (single implementation)
  const transformedMessages: TransformedMessage[] = useMemo(
    () => messages.map(transformMessage),
    [messages]
  );

  return {
    currentChannel,
    channelLoading,
    messagesLoading,
    loadingMore,
    messagesError,
    podCount,
    channelName,
    transformedMessages,
    hasMore,
    currentUserId,
    handlePodsChanged,
    handleSendMessage,
    handleEditMessage,
    handleDeleteMessage,
    handleLoadMore,
    handleRefresh,
  };
}
