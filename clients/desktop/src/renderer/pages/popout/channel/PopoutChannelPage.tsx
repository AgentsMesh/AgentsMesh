import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useCurrentUser } from "@/stores/auth";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { ChannelChatPanel } from "@/components/channel";

// Chromeless single-channel window (no DashboardShell). Renders ChannelChatPanel
// directly off the :id route param — it self-loads via useChannelChat. View
// state is renderer-local; the channel data cache syncs across windows through
// the main-process realtime mirror.
export function PopoutChannelPage() {
  const { id } = useParams<{ id: string }>();
  const user = useCurrentUser();
  const channelId = Number(id);

  // Static title to match the web sibling: this popout's renderer-local channel
  // cache is empty (no channel-list fetch here), so deriving channel.name never
  // resolved anyway.
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
