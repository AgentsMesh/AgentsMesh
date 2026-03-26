"use client";

import { useCallback } from "react";
import { Button } from "@/components/ui/button";
import { ChannelHeader } from "@/components/channel/ChannelHeader";
import { ChannelDocument } from "@/components/channel/ChannelDocument";
import { MessageList } from "@/components/channel/MessageList";
import { MessageInput } from "@/components/channel/MessageInput";
import { ChevronLeft } from "lucide-react";
import { useChannelChat } from "@/hooks/useChannelChat";

interface ChannelDetailViewProps {
  channelId: number;
  onBack: () => void;
  onPodsChanged?: () => void;
  t: (key: string, params?: Record<string, string | number>) => string;
}

/**
 * Channel detail view with messages and input.
 * Uses useChannelChat hook internally — no props drilling for message state.
 */
export function ChannelDetailView({
  channelId,
  onBack,
  onPodsChanged,
  t,
}: ChannelDetailViewProps) {
  const chat = useChannelChat({ channelId });
  const { handlePodsChanged: chatPodsChanged } = chat;

  const handlePodsChanged = useCallback(() => {
    chatPodsChanged();
    onPodsChanged?.();
  }, [chatPodsChanged, onPodsChanged]);

  return (
    <div className="flex flex-col h-full">
      {/* Channel Header with back button - softer styling */}
      <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/30">
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0 hover:bg-muted"
          onClick={onBack}
        >
          <ChevronLeft className="w-4 h-4" />
        </Button>
        <div className="flex-1 min-w-0">
          <ChannelHeader
            name={chat.channelName}
            description={chat.currentChannel?.description}
            podCount={chat.podCount}
            channelId={channelId}
            onClose={onBack}
            onRefresh={chat.handleRefresh}
            loading={chat.messagesLoading}
            compact
            onPodsChanged={handlePodsChanged}
          />
        </div>
      </div>

      {/* Document section - collapsible markdown preview */}
      {chat.currentChannel?.document && (
        <ChannelDocument document={chat.currentChannel.document} />
      )}

      {/* Messages */}
      <div className="flex-1 overflow-hidden">
        <MessageList
          messages={chat.transformedMessages}
          loading={chat.messagesLoading}
          loadingMore={chat.loadingMore}
          hasMore={chat.hasMore}
          error={chat.messagesError}
          onLoadMore={chat.handleLoadMore}
          onRetry={chat.handleRefresh}
          currentUserId={chat.currentUserId}
          onEditMessage={chat.handleEditMessage}
          onDeleteMessage={chat.handleDeleteMessage}
        />
      </div>

      {/* Input - softer top border */}
      <div className="flex-shrink-0 bg-muted/20">
        <MessageInput
          onSend={chat.handleSendMessage}
          placeholder={t("ide.bottomPanel.sendMessagePlaceholder")}
          channelId={channelId}
        />
      </div>
    </div>
  );
}

export default ChannelDetailView;
