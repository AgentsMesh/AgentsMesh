"use client";

/**
 * Shared hook for channel chat business logic.
 * Eliminates ~80% code duplication between ChannelChatPanel and MobileChannelChat.
 */

import { useEffect, useCallback, useMemo } from "react";
import { useChannelStore } from "@/stores/channel";
import { useMeshStore } from "@/stores/mesh";
import { podApi } from "@/lib/api/pod";
import {
  extractPromptFromMention,
  buildChannelPrompt,
  type MentionedPod,
} from "@/components/channel/mention";
import type { TransformedMessage } from "@/components/channel/types";

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
  handlePodsChanged: () => void;
  handleSendMessage: (content: string, mentionedPods?: MentionedPod[]) => Promise<void>;
  handleLoadMore: () => void;
  handleRefresh: () => void;
}

export function useChannelChat({ channelId }: UseChannelChatOptions): UseChannelChatReturn {
  const currentChannel = useChannelStore((s) => s.currentChannel);
  const messages = useChannelStore((s) => s.messages);
  const messagesLoading = useChannelStore((s) => s.messagesLoading);
  const channelLoading = useChannelStore((s) => s.channelLoading);
  const fetchChannel = useChannelStore((s) => s.fetchChannel);
  const fetchMessages = useChannelStore((s) => s.fetchMessages);
  const sendMessage = useChannelStore((s) => s.sendMessage);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);

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

  // Derive pod count and channel name from topology + currentChannel
  const channelInfo = topology?.channels.find((c) => c.id === channelId);
  const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;
  const channelName = currentChannel?.name || channelInfo?.name || "Channel";

  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    fetchChannel(channelId);
  }, [fetchTopology, fetchChannel, channelId]);

  const handleSendMessage = useCallback(
    async (content: string, mentionedPods?: MentionedPod[]) => {
      try {
        await sendMessage(channelId, content);

        if (mentionedPods && mentionedPods.length > 0) {
          const rawPrompt = extractPromptFromMention(content, mentionedPods);
          if (rawPrompt) {
            const prompt = buildChannelPrompt(rawPrompt, channelName);
            await Promise.allSettled(
              mentionedPods.map((pod) => podApi.sendPrompt(pod.podKey, prompt))
            );
          }
        }
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [channelId, sendMessage, channelName]
  );

  const handleLoadMore = useCallback(() => {
    fetchMessages(channelId, 50, messages.length);
  }, [channelId, messages.length, fetchMessages]);

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
    handlePodsChanged,
    handleSendMessage,
    handleLoadMore,
    handleRefresh,
  };
}
