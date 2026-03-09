"use client";

import { useChannelChat } from "@/hooks/useChannelChat";
import { ChannelHeader } from "./ChannelHeader";
import { ChannelDocument } from "./ChannelDocument";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { Loader2 } from "lucide-react";

interface ChannelChatPanelProps {
  channelId: number;
  onClose?: () => void;
}

export function ChannelChatPanel({ channelId, onClose }: ChannelChatPanelProps) {
  const chat = useChannelChat({ channelId });

  // Loading skeleton
  if (chat.channelLoading && !chat.currentChannel) {
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
      <ChannelHeader
        name={chat.channelName}
        description={chat.currentChannel?.description}
        podCount={chat.podCount}
        channelId={channelId}
        onClose={onClose}
        onRefresh={chat.handleRefresh}
        loading={chat.messagesLoading}
        onPodsChanged={chat.handlePodsChanged}
      />

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
        placeholder="Send a message to this channel..."
        channelId={channelId}
      />
    </div>
  );
}

export default ChannelChatPanel;
