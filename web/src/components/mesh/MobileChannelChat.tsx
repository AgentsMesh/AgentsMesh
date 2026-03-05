"use client";

import { useEffect, useCallback } from "react";
import { useChannelStore } from "@/stores/channel";
import { useMeshStore } from "@/stores/mesh";
import { podApi } from "@/lib/api/pod";
import { MessageList } from "./MessageList";
import { MessageInput, extractPromptFromMention, buildChannelPrompt } from "./MessageInput";
import type { MentionedPod } from "./MessageInput";
import { ChannelDocument } from "./ChannelDocument";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Radio, RefreshCw, Loader2 } from "lucide-react";
import { ChannelPodManager } from "./ChannelPodManager";
import { cn } from "@/lib/utils";

interface MobileChannelChatProps {
  channelId: number;
  onClose: () => void;
}

/**
 * MobileChannelChat - Full-screen chat panel for mobile devices
 */
export function MobileChannelChat({ channelId, onClose }: MobileChannelChatProps) {
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

  // Get pod count from topology
  const channelInfo = topology?.channels.find((c) => c.id === channelId);
  const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;

  // Resolve channel name for prompt context (extracted for React Compiler compatibility)
  const channelName = currentChannel?.name || channelInfo?.name || "Channel";

  // Refresh topology and channel data when pod membership changes
  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    fetchChannel(channelId);
  }, [fetchTopology, fetchChannel, channelId]);

  // Handle send message — also forward prompt to @mentioned pods
  const handleSendMessage = useCallback(
    async (content: string, mentionedPods?: MentionedPod[]) => {
      try {
        await sendMessage(channelId, content);

        // Forward prompt to each mentioned pod's terminal
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

  // Handle load more messages
  const handleLoadMore = useCallback(() => {
    fetchMessages(channelId, 50, messages.length);
  }, [channelId, messages.length, fetchMessages]);

  // Handle refresh messages
  const handleRefresh = useCallback(() => {
    fetchMessages(channelId);
  }, [channelId, fetchMessages]);

  // Transform messages for MessageList component
  // Note: Backend returns sender_pod_info and sender_user (GORM json tags)
  const transformedMessages = messages.map((msg) => ({
    id: msg.id,
    content: msg.content,
    messageType: msg.message_type as "text" | "system" | "code" | "command",
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
  }));

  // Loading state
  if (channelLoading && !currentChannel) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-background">
        <div className="flex-shrink-0 border-b border-border px-4 py-3 flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onClose}>
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div className="h-6 w-32 bg-muted animate-pulse rounded" />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      {/* Header with back button */}
      <div className="flex-shrink-0 border-b border-border">
        <div className="flex items-center justify-between px-2 py-2">
          <div className="flex items-center gap-2 min-w-0">
            <Button variant="ghost" size="icon" className="h-9 w-9 flex-shrink-0" onClick={onClose}>
              <ArrowLeft className="w-5 h-5" />
            </Button>
            <div className="w-8 h-8 rounded-lg bg-blue-500/10 flex items-center justify-center flex-shrink-0">
              <Radio className="w-4 h-4 text-blue-500 dark:text-blue-400" />
            </div>
            <div className="min-w-0">
              <h3 className="font-semibold text-sm truncate">#{channelName}</h3>
              {currentChannel?.description && (
                <p className="text-xs text-muted-foreground truncate">
                  {currentChannel.description}
                </p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2 mr-2 flex-shrink-0">
            {/* Pod manager popover */}
            <ChannelPodManager
              channelId={channelId}
              podCount={podCount}
              onPodsChanged={handlePodsChanged}
            />

            {/* Refresh button */}
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={handleRefresh}
              disabled={messagesLoading}
            >
              <RefreshCw className={cn("w-4 h-4", messagesLoading && "animate-spin")} />
            </Button>
          </div>
        </div>
      </div>

      {/* Document section - collapsible markdown preview */}
      {currentChannel?.document && (
        <ChannelDocument document={currentChannel.document} />
      )}

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
        placeholder="Send a message..."
        channelId={channelId}
      />
    </div>
  );
}

export default MobileChannelChat;
