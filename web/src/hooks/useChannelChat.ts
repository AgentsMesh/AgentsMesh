"use client";

/**
 * Shared hook for channel chat business logic.
 * Eliminates ~80% code duplication between ChannelChatPanel and MobileChannelChat.
 */

import { useEffect, useCallback, useMemo, useRef } from "react";
import { useAuthStore } from "@/stores/auth";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";
import { useMeshStore } from "@/stores/mesh";
import type { TransformedMessage } from "@/components/channel/types";
import type { MentionPayload } from "@/lib/api/channel";

interface UseChannelChatOptions {
  channelId: number;
}

interface UseChannelChatReturn {
  currentChannel: ReturnType<typeof useChannelStore.getState>["currentChannel"];
  channelLoading: boolean;
  messagesLoading: boolean;
  podCount: number;
  channelName: string;
  transformedMessages: TransformedMessage[];
  hasMore: boolean;
  currentUserId: number | undefined;
  handlePodsChanged: () => void;
  handleSendMessage: (content: string, mentions?: MentionPayload[]) => Promise<void>;
  handleEditMessage: (messageId: number, content: string) => Promise<void>;
  handleDeleteMessage: (messageId: number) => Promise<void>;
  handleLoadMore: () => void;
  handleRefresh: () => void;
}

export function useChannelChat({ channelId }: UseChannelChatOptions): UseChannelChatReturn {
  const currentUserId = useAuthStore((s) => s.user?.id);

  const currentChannel = useChannelStore((s) => s.currentChannel);
  const channelLoading = useChannelStore((s) => s.channelLoading);
  const fetchChannel = useChannelStore((s) => s.fetchChannel);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);

  const messages = useChannelMessageStore((s) => s.messages);
  const messagesLoading = useChannelMessageStore((s) => s.messagesLoading);
  const fetchMessages = useChannelMessageStore((s) => s.fetchMessages);
  const sendMessage = useChannelMessageStore((s) => s.sendMessage);
  const editMessage = useChannelMessageStore((s) => s.editMessage);
  const deleteMessage = useChannelMessageStore((s) => s.deleteMessage);
  const markRead = useChannelMessageStore((s) => s.markRead);

  const topology = useMeshStore((s) => s.topology);
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

  // Auto mark-as-read when messages finish loading and channel is visible
  const prevMessagesLoadingRef = useRef(true);
  useEffect(() => {
    // Trigger when loading transitions from true → false (messages just loaded)
    if (prevMessagesLoadingRef.current && !messagesLoading && messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      markRead(channelId, lastMessage.id);
    }
    prevMessagesLoadingRef.current = messagesLoading;
  }, [messagesLoading, messages, channelId, markRead]);

  // Derive pod count and channel name from topology + currentChannel
  const channelInfo = topology?.channels.find((c) => c.id === channelId);
  const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;
  const channelName = currentChannel?.name || channelInfo?.name || "Channel";

  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    fetchChannel(channelId);
  }, [fetchTopology, fetchChannel, channelId]);

  const handleSendMessage = useCallback(
    async (content: string, mentions?: MentionPayload[]) => {
      try {
        await sendMessage(channelId, content, undefined, mentions);
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [channelId, sendMessage]
  );

  const handleEditMessage = useCallback(
    async (messageId: number, content: string) => {
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
    // Use the first (oldest) message ID as cursor for backward pagination
    const oldestId = messages.length > 0 ? messages[0].id : undefined;
    fetchMessages(channelId, 50, oldestId);
  }, [channelId, messages, fetchMessages]);

  const handleRefresh = useCallback(() => {
    fetchMessages(channelId);
  }, [channelId, fetchMessages]);

  // Transform raw store messages into rendering-ready format
  const transformedMessages: TransformedMessage[] = useMemo(
    () =>
      messages.map((msg) => ({
        id: msg.id,
        content: msg.content,
        messageType: msg.message_type as TransformedMessage["messageType"],
        metadata: msg.metadata,
        editedAt: msg.edited_at,
        createdAt: msg.created_at,
        pod: msg.sender_pod_info
          ? {
              podKey: msg.sender_pod_info.pod_key,
              agentType: msg.sender_pod_info.agent_type
                ? { name: msg.sender_pod_info.agent_type.name }
                : undefined,
            }
          : undefined,
        user: msg.sender_user
          ? {
              id: msg.sender_user.id,
              username: msg.sender_user.username,
              name: msg.sender_user.name,
              avatarUrl: msg.sender_user.avatar_url,
            }
          : undefined,
      })),
    [messages]
  );

  const hasMore = messages.length >= 50 && messages.length % 50 === 0;

  return {
    currentChannel,
    channelLoading,
    messagesLoading,
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
