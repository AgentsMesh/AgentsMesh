"use client";

import { useState } from "react";
import { useChannelChat } from "@/hooks/useChannelChat";
import { ChannelHeader } from "./ChannelHeader";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { ChannelRightRail } from "./ChannelRightRail";
import { MessageSearchModal } from "./MessageSearchModal";
import { ChannelSettingsModal } from "./ChannelSettingsModal";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";

interface ChannelChatPanelProps {
  channelId: number;
}

export function ChannelChatPanel({ channelId }: ChannelChatPanelProps) {
  const chat = useChannelChat({ channelId });
  const t = useTranslations();
  const isMember = chat.currentChannel?.is_member ?? true;
  const visibility = chat.currentChannel?.visibility ?? "public";
  const isArchived = chat.currentChannel?.is_archived ?? false;

  const [searchOpen, setSearchOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [railOpen, setRailOpen] = useState(true);

  if (chat.channelLoading && !chat.currentChannel) {
    return (
      <div className="flex flex-col h-full bg-background">
        <div className="flex-shrink-0 border-b border-border px-4 py-3">
          <div className="h-8 w-32 bg-muted animate-pulse rounded" />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-background">
      <ChannelHeader
        name={chat.channelName}
        channelId={channelId}
        visibility={visibility}
        isMember={isMember}
        agentCount={chat.agentCount}
        memberCount={chat.currentChannel?.member_count}
        ticket={chat.currentChannel?.ticket}
        repository={chat.currentChannel?.repository}
        onOpenSearch={() => setSearchOpen(true)}
        onToggleRail={() => setRailOpen((v) => !v)}
        railOpen={railOpen}
      />

      <div className="flex min-h-0 flex-1">
        <div className="flex min-w-0 flex-1 flex-col">
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
              {isArchived ? (
                <div className="border-t border-border bg-muted/40 px-4 py-3 text-center text-sm text-muted-foreground">
                  {t("channels.archivedBanner")}
                </div>
              ) : (
                <MessageInput
                  onSend={chat.handleSendMessage}
                  channelId={channelId}
                  channelName={chat.channelName}
                />
              )}
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
              {t("channels.actions.joinToParticipate")}
            </div>
          )}
        </div>

        {railOpen && (
          <ChannelRightRail
            channel={chat.currentChannel ?? null}
            channelId={channelId}
            onPodsChanged={chat.handlePodsChanged}
            onOpenSettings={() => setSettingsOpen(true)}
          />
        )}
      </div>

      <MessageSearchModal
        open={searchOpen}
        onOpenChange={setSearchOpen}
        channelId={channelId}
        channelName={chat.channelName}
      />
      <ChannelSettingsModal
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        channel={chat.currentChannel ?? null}
      />
    </div>
  );
}

export default ChannelChatPanel;
