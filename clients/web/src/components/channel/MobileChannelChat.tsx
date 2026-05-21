"use client";

import { useChannelChat } from "@/hooks/useChannelChat";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { ChannelDocument } from "./ChannelDocument";
import { ChannelPodManager } from "./ChannelPodManager";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Radio, RefreshCw, Loader2, Lock, LogIn } from "lucide-react";
import { cn } from "@/lib/utils";
import { useChannelStore } from "@/stores/channel";
import { useTranslations } from "next-intl";
import { useState } from "react";

interface MobileChannelChatProps {
  channelId: number;
  onClose: () => void;
}

export function MobileChannelChat({ channelId, onClose }: MobileChannelChatProps) {
  const chat = useChannelChat({ channelId });
  const t = useTranslations();
  const joinUserChannel = useChannelStore((s) => s.joinUserChannel);
  const [joining, setJoining] = useState(false);

  const isMember = chat.currentChannel?.is_member ?? true;
  const isPrivate = chat.currentChannel?.visibility === "private";

  const handleJoin = async () => {
    setJoining(true);
    try { await joinUserChannel(channelId); } finally { setJoining(false); }
  };

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

  const IconComponent = isPrivate ? Lock : Radio;
  const iconColor = isPrivate ? "text-amber-500" : "text-blue-500 dark:text-blue-400";
  const iconBg = isPrivate ? "bg-amber-500/10" : "bg-blue-500/10";

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      <div className="flex-shrink-0 border-b border-border">
        <div className="flex items-center justify-between px-2 py-2">
          <div className="flex items-center gap-2 min-w-0">
            <Button variant="ghost" size="icon" className="h-9 w-9 flex-shrink-0" onClick={onClose}>
              <ArrowLeft className="w-5 h-5" />
            </Button>
            <div className={cn("w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0", iconBg)}>
              <IconComponent className={cn("w-4 h-4", iconColor)} />
            </div>
            <div className="min-w-0">
              <h3 className="font-semibold text-sm truncate">#{chat.channelName}</h3>
              {chat.currentChannel?.description && (
                <p className="text-xs text-muted-foreground truncate">{chat.currentChannel.description}</p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2 mr-2 flex-shrink-0">
            {!isMember && !isPrivate && (
              <Button size="sm" onClick={handleJoin} disabled={joining}>
                <LogIn className="w-3.5 h-3.5 mr-1.5" />
                {t("channels.actions.join")}
              </Button>
            )}
            {isMember && (
              <ChannelPodManager
                channelId={channelId}
                podCount={chat.agentCount}
                onPodsChanged={chat.handlePodsChanged}
              />
            )}
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={chat.handleRefresh} disabled={chat.messagesLoading}>
              <RefreshCw className={cn("w-4 h-4", chat.messagesLoading && "animate-spin")} />
            </Button>
          </div>
        </div>
      </div>

      {chat.currentChannel?.document && (
        <ChannelDocument document={chat.currentChannel.document} />
      )}

      {isMember ? (
        <>
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
          <MessageInput
            onSend={chat.handleSendMessage}
            placeholder="Send a message..."
            channelId={channelId}
          />
        </>
      ) : (
        <div className="flex-1 flex items-center justify-center text-muted-foreground text-sm">
          {t("channels.actions.joinToParticipate")}
        </div>
      )}
    </div>
  );
}

export default MobileChannelChat;
