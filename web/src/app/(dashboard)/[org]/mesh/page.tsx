"use client";

import { useEffect } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { MeshTopology } from "@/components/mesh";
import { ChannelChatPanel, MobileChannelChat } from "@/components/mesh";
import { useMeshStore } from "@/stores/mesh";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { useTranslations } from "next-intl";

export default function MeshPage() {
  const t = useTranslations();
  const {
    topology,
    selectedChannel,
    loading,
    error,
    fetchTopology,
    selectChannel,
    clearError,
  } = useMeshStore();

  const { isMobile } = useBreakpoint();

  useEffect(() => {
    fetchTopology();
  }, [fetchTopology]);

  const activePodCount = topology?.nodes.filter(
    (n) => n.status === "running" || n.status === "initializing"
  ).length || 0;

  const activeChannelCount = topology?.channels.filter((c) => !c.is_archived).length || 0;

  // Handle closing the chat panel
  const handleCloseChat = () => {
    selectChannel(null);
  };

  const showDesktopChat = selectedChannel && !isMobile;

  // Topology content shared between resizable and non-resizable layouts
  const topologyContent = (
    <div className="flex-1 flex flex-col min-w-0 h-full">
      {/* Header */}
      <div className="px-6 py-4 border-b border-border flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-foreground">{t("mesh.page.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("mesh.page.subtitle")}
          </p>
        </div>
        <div className="flex items-center gap-4">
          {/* Stats */}
          <div className="flex items-center gap-6 text-sm">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-green-500" />
              <span className="text-muted-foreground">
                {t("mesh.page.activePods", { count: activePodCount })}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-blue-500 dark:text-blue-400">#</span>
              <span className="text-muted-foreground">
                {t("mesh.page.channelsCount", { count: activeChannelCount })}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
              <span className="text-muted-foreground">
                {t("mesh.page.bindingsCount", { count: topology?.edges.length || 0 })}
              </span>
            </div>
          </div>

          {/* Actions */}
          <Button variant="outline" size="sm" onClick={() => fetchTopology()}>
            <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {t("mesh.page.refresh")}
          </Button>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="mx-6 mt-4 p-4 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center justify-between">
          <div className="flex items-center gap-2 text-destructive">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="text-sm">{error}</span>
          </div>
          <Button variant="ghost" size="sm" onClick={clearError}>
            {t("mesh.page.dismiss")}
          </Button>
        </div>
      )}

      {/* Topology Visualization */}
      <div className="flex-1 relative">
        {loading && !topology && (
          <div className="absolute inset-0 bg-background/50 z-10">
            <CenteredSpinner />
          </div>
        )}
        <MeshTopology />

        {/* Loading indicator for polling */}
        {loading && topology && (
          <div className="absolute top-4 right-4 flex items-center gap-2 px-3 py-1.5 bg-background/80 border border-border rounded-full text-xs text-muted-foreground">
            <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
            {t("mesh.page.updating")}
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="px-6 py-3 border-t border-border">
        <div className="flex items-center gap-6 text-xs text-muted-foreground flex-wrap">
          <span className="font-medium">{t("mesh.legend.title")}:</span>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded bg-green-500" />
            <span>{t("mesh.legend.running")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded bg-yellow-500" />
            <span>{t("mesh.legend.initializing")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded bg-gray-500" />
            <span>{t("mesh.legend.terminated")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded bg-red-500" />
            <span>{t("mesh.legend.failed")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded bg-blue-500" />
            <span>{t("mesh.legend.channel")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-6 border-t-2 border-green-500" />
            <span>{t("mesh.legend.activeBinding")}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-6 border-t-2 border-yellow-500 border-dashed" />
            <span>{t("mesh.legend.pendingBinding")}</span>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div className="flex h-full">
      {showDesktopChat ? (
        /* Resizable layout when chat panel is open on desktop */
        <Group
          orientation="horizontal"
          className="h-full hidden md:flex"
        >
          <Panel defaultSize={65} minSize={40}>
            {topologyContent}
          </Panel>
          <Separator
            className="group relative flex items-center justify-center w-1 bg-transparent hover:bg-primary transition-colors cursor-col-resize"
          >
            <div className="absolute z-10 w-3 h-full -left-1" />
          </Separator>
          <Panel defaultSize={35} minSize={20} maxSize={50}>
            <ChannelChatPanel
              channelId={selectedChannel}
              onClose={handleCloseChat}
            />
          </Panel>
        </Group>
      ) : (
        /* Full-width topology when no chat panel */
        topologyContent
      )}

      {/* Mobile: Full-screen chat overlay */}
      {selectedChannel && isMobile && (
        <MobileChannelChat
          channelId={selectedChannel}
          onClose={handleCloseChat}
        />
      )}
    </div>
  );
}
