"use client";

import { useEffect, useCallback, useMemo, useRef } from "react";
import { useCurrentUser, useAuthStore } from "@/stores/auth";
import { useChannelStore, useChannelMessageStore, useCurrentChannel } from "@/stores/channel";
import { EMPTY_CACHE, LOAD_MORE_MESSAGE_LIMIT, useChannelMessages } from "@/stores/channelMessageStore";
import { useMeshStore, useTopology } from "@/stores/mesh";
import { transformMessage } from "@/components/channel/transformMessage";
import type { TransformedMessage } from "@/components/channel/types";
import type { MessageSendPayload, MessageEditPayload } from "@/lib/api/channel-message-types";

interface UseChannelChatOptions {
  channelId: number;
}

interface UseChannelChatReturn {
  currentChannel: ReturnType<typeof useCurrentChannel>;
  channelLoading: boolean;
  messagesLoading: boolean;
  loadingMore: boolean;
  messagesError: string | null;
  agentCount: number;
  channelName: string;
  transformedMessages: TransformedMessage[];
  hasMore: boolean;
  currentUserId: number | undefined;
  handlePodsChanged: () => void;
  handleSendMessage: (payload: MessageSendPayload) => Promise<void>;
  handleEditMessage: (messageId: number, payload: MessageEditPayload) => Promise<void>;
  handleDeleteMessage: (messageId: number) => Promise<void>;
  handleLoadMore: () => void;
  handleRefresh: () => void;
}

export function useChannelChat({ channelId }: UseChannelChatOptions): UseChannelChatReturn {
  const currentUserId = useCurrentUser()?.id;

  // WASM `useCurrentChannel` reflects fetchChannel writes; JS `currentChannel`
  // only updates via setCurrentChannel and stays null on select→fetch.
  const currentChannel = useCurrentChannel();
  const channelLoading = useChannelStore((s) => s.channelLoading);
  const fetchChannel = useChannelStore((s) => s.fetchChannel);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);

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

  useEffect(() => {
    if (channelId) {
      fetchChannel(channelId);
      fetchMessages(channelId);
    }
    return () => {
      setCurrentChannel(null);
    };
  }, [channelId, fetchChannel, fetchMessages, setCurrentChannel]);

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

  const channelInfo = topology?.channels.find((c: { id: number }) => c.id === channelId);
  const agentCount = currentChannel?.agent_count ?? channelInfo?.pod_keys.length ?? 0;
  const channelName = currentChannel?.name || channelInfo?.name || "Channel";

  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    fetchChannel(channelId);
  }, [fetchTopology, fetchChannel, channelId]);

  const handleSendMessage = useCallback(
    async (payload: MessageSendPayload) => {
      try {
        await sendMessage(channelId, payload);
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [channelId, sendMessage]
  );

  const handleEditMessage = useCallback(
    async (messageId: number, payload: MessageEditPayload) => {
      await editMessage(channelId, messageId, payload);
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
    if (loadingMore || !hasMore || messages.length === 0) return;
    fetchMessages(channelId, LOAD_MORE_MESSAGE_LIMIT, messages[0].id);
  }, [channelId, messages, fetchMessages, loadingMore, hasMore]);

  const handleRefresh = useCallback(() => {
    fetchMessages(channelId);
  }, [channelId, fetchMessages]);

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
    agentCount,
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
