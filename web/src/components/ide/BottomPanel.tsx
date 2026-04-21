"use client";

import React, { useRef, useEffect, useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import { useIDEStore, type BottomPanelTab } from "@/stores/ide";
import { useChannelStore } from "@/stores/channel";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronUp, X, MessageSquare, Activity, Bot, GitPullRequest, Info } from "lucide-react";
import { AutopilotPanelContent } from "@/components/autopilot";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useBottomPanelData } from "./useBottomPanelData";
import { ChannelsTabContent, ActivityTabContent, DeliveryTabContent, InfoTabContent } from "./BottomPanel/index";

const TAB_ICONS: Record<BottomPanelTab, React.ReactNode> = {
  channels: <MessageSquare className="w-3.5 h-3.5" />,
  activity: <Activity className="w-3.5 h-3.5" />,
  autopilot: <Bot className="w-3.5 h-3.5" />,
  delivery: <GitPullRequest className="w-3.5 h-3.5" />,
  info: <Info className="w-3.5 h-3.5" />,
};
const TAB_IDS: BottomPanelTab[] = ["channels", "activity", "autopilot", "delivery", "info"];

export function BottomPanel({ className }: { className?: string }) {
  const t = useTranslations();
  const bottomPanelOpen = useIDEStore((s) => s.bottomPanelOpen);
  const bottomPanelHeight = useIDEStore((s) => s.bottomPanelHeight);
  const bottomPanelTab = useIDEStore((s) => s.bottomPanelTab);
  const setBottomPanelOpen = useIDEStore((s) => s.setBottomPanelOpen);
  const setBottomPanelHeight = useIDEStore((s) => s.setBottomPanelHeight);
  const setBottomPanelTab = useIDEStore((s) => s.setBottomPanelTab);
  const toggleBottomPanel = useIDEStore((s) => s.toggleBottomPanel);
  const orgSlug = useCurrentOrg()?.slug || "";

  const {
    selectedPodKey, currentPod, activeAutopilot, topology, fetchTopology,
    podChannels, incomingBindings, outgoingBindings, getPodInfo,
  } = useBottomPanelData();

  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);
  const resizeRef = useRef<HTMLDivElement>(null);
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => { if (!topology) fetchTopology(); }, [topology, fetchTopology]);

  // Resize handler
  useEffect(() => {
    if (!isResizing) return;
    const onMove = (e: MouseEvent) => {
      setBottomPanelHeight(Math.min(Math.max(window.innerHeight - e.clientY, 100), window.innerHeight * 0.6));
    };
    const onUp = () => setIsResizing(false);
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
    return () => { document.removeEventListener("mousemove", onMove); document.removeEventListener("mouseup", onUp); };
  }, [isResizing, setBottomPanelHeight]);

  const handleChannelClick = useCallback((id: number) => setSelectedChannelId(id), []);
  const handleBackToChannelList = useCallback(() => { setSelectedChannelId(null); setCurrentChannel(null); }, [setCurrentChannel]);
  const handlePodsChanged = useCallback(() => fetchTopology(), [fetchTopology]);
  const handleTabClick = useCallback((tabId: BottomPanelTab, shouldOpen = false) => {
    setBottomPanelTab(tabId);
    if (shouldOpen) setBottomPanelOpen(true);
    if (tabId !== "channels") setSelectedChannelId(null);
  }, [setBottomPanelTab, setBottomPanelOpen]);

  const renderTabButtons = (collapsed = false) => (
    <>
      {TAB_IDS.map((tabId) => (
        <button key={tabId} className={cn(
          "flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors relative",
          bottomPanelTab === tabId ? (collapsed ? "text-foreground" : "text-foreground bg-muted") : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
          tabId === "autopilot" && activeAutopilot && bottomPanelTab !== tabId && "text-green-500"
        )} onClick={() => handleTabClick(tabId, collapsed)}>
          {TAB_ICONS[tabId]}
          <span>{t(`ide.bottomPanel.${tabId}`)}</span>
          {tabId === "autopilot" && activeAutopilot && (
            <span className="relative flex h-2 w-2 ml-1">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
            </span>
          )}
        </button>
      ))}
    </>
  );

  if (!bottomPanelOpen) {
    return (
      <div className={cn("h-8 bg-background border-t border-border flex items-center px-2 gap-2", className)}>
        {renderTabButtons(true)}
        <div className="flex-1" />
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={toggleBottomPanel}><ChevronUp className="w-4 h-4" /></Button>
      </div>
    );
  }

  return (
    <div className={cn("bg-background border-t border-border flex flex-col", className)} style={{ height: bottomPanelHeight }}>
      <div ref={resizeRef} className={cn("h-1 cursor-row-resize hover:bg-primary/50 transition-colors", isResizing && "bg-primary/50")} onMouseDown={() => setIsResizing(true)} />
      <div className="h-8 flex items-center px-2 gap-2 border-b border-border">
        {renderTabButtons()}
        <div className="flex-1" />
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={toggleBottomPanel}><ChevronDown className="w-4 h-4" /></Button>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={() => setBottomPanelOpen(false)}><X className="w-4 h-4" /></Button>
      </div>
      <div className="flex-1 overflow-auto p-2">
        {bottomPanelTab === "channels" && <ChannelsTabContent selectedPodKey={selectedPodKey} podChannels={podChannels} selectedChannelId={selectedChannelId} onChannelClick={handleChannelClick} onBackToList={handleBackToChannelList} onPodsChanged={handlePodsChanged} t={t} />}
        {bottomPanelTab === "activity" && <ActivityTabContent selectedPodKey={selectedPodKey} incomingBindings={incomingBindings} outgoingBindings={outgoingBindings} getPodInfo={getPodInfo} t={t} />}
        {bottomPanelTab === "autopilot" && <AutopilotPanelContent podKey={selectedPodKey} />}
        {bottomPanelTab === "delivery" && <DeliveryTabContent selectedPodKey={selectedPodKey} pod={currentPod} t={t} />}
        {bottomPanelTab === "info" && <InfoTabContent selectedPodKey={selectedPodKey} pod={currentPod} orgSlug={orgSlug} t={t} />}
      </div>
    </div>
  );
}

export default BottomPanel;
