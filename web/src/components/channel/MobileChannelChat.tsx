"use client";

import { useChannelChat } from "@/hooks/useChannelChat";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { ChannelDocument } from "./ChannelDocument";
import { ChannelPodManager } from "./ChannelPodManager";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Radio, RefreshCw, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface MobileChannelChatProps {
  channelId: number;
  onClose: () => void;
}

/**
 * Full-screen chat panel for mobile devices.
 * Uses the same useChannelChat hook as ChannelChatPanel,
 * but with mobile-specific layout (fixed inset-0, back button).
 */
export function MobileChannelChat({ channelId, onClose }: MobileChannelChatProps) {
  const chat = useChannelChat({ channelId });

  // Loading skeleton
  if (chat.channelLoading && !chat.currentChannel) {
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
              <h3 className="font-semibold text-sm truncate">#{chat.channelName}</h3>
              {chat.currentChannel?.description && (
                <p className="text-xs text-muted-foreground truncate">
                  {chat.currentChannel.description}
                </p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2 mr-2 flex-shrink-0">
            <ChannelPodManager
              channelId={channelId}
              podCount={chat.podCount}
              onPodsChanged={chat.handlePodsChanged}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={chat.handleRefresh}
              disabled={chat.messagesLoading}
            >
              <RefreshCw className={cn("w-4 h-4", chat.messagesLoading && "animate-spin")} />
            </Button>
          </div>
        </div>
      </div>

      {chat.currentChannel?.document && (
        <ChannelDocument document={chat.currentChannel.document} />
      )}

      <MessageList
        messages={chat.transformedMessages}
        loading={chat.messagesLoading}
        hasMore={chat.hasMore}
        onLoadMore={chat.handleLoadMore}
        currentUserId={chat.currentUserId}
        onEditMessage={chat.handleEditMessage}
        onDeleteMessage={chat.handleDeleteMessage}
      />

      <MessageInput
        onSend={chat.handleSendMessage}
        placeholder="Send a message..."
        channelId={channelId}
      />
    </div>
  );
}

export default MobileChannelChat;
