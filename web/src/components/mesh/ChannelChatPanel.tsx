"use client";

import { useEffect, useCallback } from "react";
import { useChannelStore } from "@/stores/channel";
import { useMeshStore } from "@/stores/mesh";
import { ChannelHeader } from "./ChannelHeader";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { Loader2 } from "lucide-react";

interface ChannelChatPanelProps {
  channelId: number;
  onClose: () => void;
}

export function ChannelChatPanel({ channelId, onClose }: ChannelChatPanelProps) {
  const {
    currentChannel,
    messages,
    messagesLoading,
    loading,
    fetchChannel,
    fetchMessages,
    sendMessage,
    setCurrentChannel,
  } = useChannelStore();

  const { topology } = useMeshStore();

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

  // Get pod count from topology for this channel
  const channelInfo = topology?.channels.find((c) => c.id === channelId);
  const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;

  // Handle send message
  const handleSendMessage = useCallback(
    async (content: string) => {
      try {
        await sendMessage(channelId, content);
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [channelId, sendMessage]
  );

  // Handle load more messages
  const handleLoadMore = useCallback(() => {
    fetchMessages(channelId, 50, messages.length);
  }, [channelId, messages.length, fetchMessages]);

  // Handle refresh messages
  const handleRefresh = useCallback(() => {
    fetchMessages(channelId);
  }, [channelId, fetchMessages]);

  // Transform messages for MessageList component
  const transformedMessages = messages.map((msg) => ({
    id: msg.id,
    content: msg.content,
    messageType: msg.message_type as "text" | "system" | "code" | "command",
    metadata: msg.metadata,
    createdAt: msg.created_at,
    pod: msg.pod
      ? {
          podKey: msg.pod.pod_key,
          agentType: msg.pod.agent_type ? { name: msg.pod.agent_type.name } : undefined,
        }
      : undefined,
    user: msg.user
      ? {
          id: msg.user.id,
          username: msg.user.username,
          name: msg.user.name,
          avatarUrl: msg.user.avatar_url,
        }
      : undefined,
  }));

  // Loading state
  if (loading && !currentChannel) {
    return (
      <div className="flex flex-col h-full bg-background">
        <div className="flex-shrink-0 border-b border-border px-4 py-3">
          <div className="h-8 w-32 bg-muted animate-pulse rounded" />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-background">
      {/* Header */}
      <ChannelHeader
        name={currentChannel?.name || channelInfo?.name || "Channel"}
        description={currentChannel?.description}
        podCount={podCount}
        onClose={onClose}
        onRefresh={handleRefresh}
        loading={messagesLoading}
      />

      {/* Messages */}
      <MessageList
        messages={transformedMessages}
        loading={messagesLoading}
        hasMore={messages.length >= 50 && messages.length % 50 === 0}
        onLoadMore={handleLoadMore}
      />

      {/* Input */}
      <MessageInput
        onSend={handleSendMessage}
        placeholder="Send a message to this channel..."
      />
    </div>
  );
}

export default ChannelChatPanel;
