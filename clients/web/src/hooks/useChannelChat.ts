"use client";

import { useEffect, useCallback, useMemo, useState } from "react";
import { useCurrentUser } from "@/stores/auth";
import { useChannelStore, useChannelMessageStore, useCurrentChannel, useChannelMembers } from "@/stores/channel";
import { EMPTY_CACHE, LOAD_MORE_MESSAGE_LIMIT, useChannelMessages } from "@/stores/channelMessageStore";
import { useMeshStore, useTopology } from "@/stores/mesh";
import { transformMessage } from "@/components/channel/transformMessage";
import { useChannelEntryAnchor } from "@/components/channel/useChannelEntryAnchor";
import { useChannelEntryMarkRead } from "./useChannelEntryMarkRead";
import type { TransformedMessage } from "@/components/channel/types";
import type { MessageSendPayload, MessageEditPayload } from "@/lib/viewModels/channelMessage";

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
  firstUnreadId: number | null;
  entryAnchorResolved: boolean;
  roleByUserId: Map<number, string>;
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
  const fetchUnreadCounts = useChannelMessageStore((s) => s.fetchUnreadCounts);

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
  // Tagged with channelId: `attempted` only counts for the channel it was set
  // on, so switching away derives false without a reset effect. (A→B→A re-entry
  // reuses a stale attempted=true, but only matters if the cursor is unknown
  // again on return, which mark-read on the prior visit makes vanishingly rare.)
  const [unreadSummaryState, setUnreadSummaryState] = useState({
    channelId: null as number | null,
    attempted: false,
  });
  const unreadSummaryAttempted =
    unreadSummaryState.channelId === channelId && unreadSummaryState.attempted;

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

  const entryAnchor = useChannelEntryAnchor(channelId, transformedMessages, unreadSummaryAttempted);
  const firstUnreadId = entryAnchor.firstUnreadId;

  useEffect(() => {
    if (!entryAnchor.needsUnreadSummary || unreadSummaryAttempted) return;
    let cancelled = false;
    fetchUnreadCounts().finally(() => {
      if (!cancelled) setUnreadSummaryState({ channelId, attempted: true });
    });
    return () => {
      cancelled = true;
    };
  }, [channelId, entryAnchor.needsUnreadSummary, fetchUnreadCounts, unreadSummaryAttempted]);

  useChannelEntryMarkRead({
    channelId,
    lastMessageId,
    entryAnchorResolved: entryAnchor.resolved,
    markRead,
  });

  const members = useChannelMembers(channelId);
  const roleByUserId = useMemo(
    () => new Map(members.map((m) => [m.user_id, m.role])),
    [members],
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
    firstUnreadId,
    entryAnchorResolved: entryAnchor.resolved,
    roleByUserId,
    currentUserId,
    handlePodsChanged,
    handleSendMessage,
    handleEditMessage,
    handleDeleteMessage,
    handleLoadMore,
    handleRefresh,
  };
}
