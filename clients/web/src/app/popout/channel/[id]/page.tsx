"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useCurrentUser } from "@/stores/auth";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { ChannelChatPanel } from "@/components/channel";

// Popout single-channel window (browser popout on web; native window on
// desktop). Mirrors the popout terminal page: AuthBootstrap (popout/layout)
// rehydrates the session, RealtimeProvider drives live messages, and
// ChannelChatPanel self-loads from the :id route param via useChannelChat.
export default function PopoutChannelPage() {
  const { id } = useParams<{ id: string }>();
  const user = useCurrentUser();
  const channelId = Number(id);

  useEffect(() => {
    document.title = "Channel";
  }, []);

  if (!user || !Number.isFinite(channelId) || channelId <= 0) return null;

  return (
    <RealtimeProvider>
      <div className="h-screen w-screen bg-background">
        <ChannelChatPanel channelId={channelId} />
      </div>
    </RealtimeProvider>
  );
}
