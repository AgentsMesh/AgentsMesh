"use client";

import React, { useRef, useEffect, useState, useMemo, useCallback } from "react";
import { cn } from "@/lib/utils";
import { useIDEStore, type BottomPanelTab } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { useMeshStore, type MeshEdge } from "@/stores/mesh";
import { useChannelStore } from "@/stores/channel";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronUp,
  X,
  MessageSquare,
  Activity,
  Bot,
  GitPullRequest,
  Info,
} from "lucide-react";
import { AutopilotPanelContent } from "@/components/autopilot";
import { useAutopilotStore } from "@/stores/autopilot";
import { usePodStore } from "@/stores/pod";
import { useAuthStore } from "@/stores/auth";
import { podApi } from "@/lib/api/pod";
import { extractPromptFromMention, buildChannelPrompt } from "@/components/mesh/MessageInput";
import type { MentionedPod } from "@/components/mesh/MessageInput";
import { ChannelsTabContent, ActivityTabContent, DeliveryTabContent, InfoTabContent } from "./BottomPanel/index";

interface BottomPanelProps {
  className?: string;
}

const TAB_ICONS: Record<BottomPanelTab, React.ReactNode> = {
  channels: <MessageSquare className="w-3.5 h-3.5" />,
  activity: <Activity className="w-3.5 h-3.5" />,
  autopilot: <Bot className="w-3.5 h-3.5" />,
  delivery: <GitPullRequest className="w-3.5 h-3.5" />,
  info: <Info className="w-3.5 h-3.5" />,
};

const TAB_IDS: BottomPanelTab[] = ["channels", "activity", "autopilot", "delivery", "info"];

export function BottomPanel({ className }: BottomPanelProps) {
  const t = useTranslations();
  const bottomPanelOpen = useIDEStore((s) => s.bottomPanelOpen);
  const bottomPanelHeight = useIDEStore((s) => s.bottomPanelHeight);
  const bottomPanelTab = useIDEStore((s) => s.bottomPanelTab);
  const setBottomPanelOpen = useIDEStore((s) => s.setBottomPanelOpen);
  const setBottomPanelHeight = useIDEStore((s) => s.setBottomPanelHeight);
  const setBottomPanelTab = useIDEStore((s) => s.setBottomPanelTab);
  const toggleBottomPanel = useIDEStore((s) => s.toggleBottomPanel);

  const currentOrg = useAuthStore((s) => s.currentOrg);
  const orgSlug = currentOrg?.slug || "";

  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const topology = useMeshStore((s) => s.topology);
  const fetchTopology = useMeshStore((s) => s.fetchTopology);

  // Channel detail state
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const currentChannel = useChannelStore((s) => s.currentChannel);
  const messages = useChannelStore((s) => s.messages);
  const messagesLoading = useChannelStore((s) => s.messagesLoading);
  const fetchChannel = useChannelStore((s) => s.fetchChannel);
  const fetchMessages = useChannelStore((s) => s.fetchMessages);
  const sendMessage = useChannelStore((s) => s.sendMessage);
  const setCurrentChannel = useChannelStore((s) => s.setCurrentChannel);

  const resizeRef = useRef<HTMLDivElement>(null);
  const [isResizing, setIsResizing] = useState(false);

  // Fetch topology data if not available
  useEffect(() => {
    if (!topology) {
      fetchTopology();
    }
  }, [topology, fetchTopology]);

  // Load channel and messages when selectedChannelId changes
  useEffect(() => {
    if (selectedChannelId) {
      fetchChannel(selectedChannelId);
      fetchMessages(selectedChannelId);
    }

    return () => {
      if (selectedChannelId) {
        setCurrentChannel(null);
      }
    };
  }, [selectedChannelId, fetchChannel, fetchMessages, setCurrentChannel]);

  // Get the currently selected pod's podKey
  const selectedPodKey = useMemo(() => {
    if (!activePane) return null;
    const pane = panes.find((p) => p.id === activePane);
    return pane?.podKey ?? null;
  }, [activePane, panes]);

  // Get current pod data for Delivery tab — reactive selector
  const currentPod = usePodStore((s) =>
    selectedPodKey ? s.pods.find((p) => p.pod_key === selectedPodKey) ?? null : null
  );

  // Reactive autopilot selector — re-renders when controller for this pod changes
  const activePhases = ["initializing", "running", "paused", "user_takeover", "waiting_approval"];
  const activeAutopilot = useAutopilotStore((s) =>
    selectedPodKey
      ? s.autopilotControllers.find((c) => c.pod_key === selectedPodKey && activePhases.includes(c.phase))
      : undefined
  );

  // Get channels and bindings for selected pod — compute directly from topology
  // (imperative store methods like getChannelsForNode have stable refs that
  // won't trigger useMemo re-computation when topology updates)
  const podChannels = useMemo(() => {
    if (!selectedPodKey || !topology) return [];
    return topology.channels.filter((c) => c.pod_keys.includes(selectedPodKey));
  }, [selectedPodKey, topology]);

  const podEdges = useMemo(() => {
    if (!selectedPodKey || !topology) return [];
    return topology.edges.filter(
      (e) => e.source === selectedPodKey || e.target === selectedPodKey
    );
  }, [selectedPodKey, topology]);

  // Separate incoming and outgoing bindings
  const { incomingBindings, outgoingBindings } = useMemo(() => {
    const incoming: MeshEdge[] = [];
    const outgoing: MeshEdge[] = [];

    podEdges.forEach((edge) => {
      if (edge.target === selectedPodKey) {
        incoming.push(edge);
      } else if (edge.source === selectedPodKey) {
        outgoing.push(edge);
      }
    });

    return { incomingBindings: incoming, outgoingBindings: outgoing };
  }, [podEdges, selectedPodKey]);

  // Get pod info from topology
  const getPodInfo = useCallback((podKey: string) => {
    return topology?.nodes.find((n) => n.pod_key === podKey);
  }, [topology]);

  // Handle panel resize
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      const windowHeight = window.innerHeight;
      const newHeight = Math.min(
        Math.max(windowHeight - e.clientY, 100),
        windowHeight * 0.6
      );
      setBottomPanelHeight(newHeight);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    }

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, setBottomPanelHeight]);

  // Channel handlers
  const handleChannelClick = useCallback((channelId: number) => {
    setSelectedChannelId(channelId);
  }, []);

  const handleBackToChannelList = useCallback(() => {
    setSelectedChannelId(null);
    setCurrentChannel(null);
  }, [setCurrentChannel]);

  // Resolve channel name for prompt context (extracted for React Compiler compatibility)
  const selectedChannelName = currentChannel?.name || "unknown";

  const handleSendMessage = useCallback(
    async (content: string, mentionedPods?: MentionedPod[]) => {
      if (!selectedChannelId) return;
      try {
        await sendMessage(selectedChannelId, content);

        // Forward prompt to each mentioned pod's terminal
        if (mentionedPods && mentionedPods.length > 0) {
          const rawPrompt = extractPromptFromMention(content, mentionedPods);
          if (rawPrompt) {
            const prompt = buildChannelPrompt(rawPrompt, selectedChannelName);
            await Promise.allSettled(
              mentionedPods.map((pod) => podApi.sendPrompt(pod.podKey, prompt))
            );
          }
        }
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [selectedChannelId, sendMessage, selectedChannelName]
  );

  const handleLoadMoreMessages = useCallback(() => {
    if (!selectedChannelId) return;
    fetchMessages(selectedChannelId, 50, messages.length);
  }, [selectedChannelId, messages.length, fetchMessages]);

  const handleRefreshMessages = useCallback(() => {
    if (!selectedChannelId) return;
    fetchMessages(selectedChannelId);
  }, [selectedChannelId, fetchMessages]);

  // Refresh topology and channel data when pod membership changes
  const handlePodsChanged = useCallback(() => {
    fetchTopology();
    if (selectedChannelId) {
      fetchChannel(selectedChannelId);
    }
  }, [fetchTopology, fetchChannel, selectedChannelId]);

  // Tab switch handler
  const handleTabClick = useCallback((tabId: BottomPanelTab, shouldOpen = false) => {
    setBottomPanelTab(tabId);
    if (shouldOpen) {
      setBottomPanelOpen(true);
    }
    // Reset channel selection when switching tabs
    if (tabId !== "channels") {
      setSelectedChannelId(null);
    }
  }, [setBottomPanelTab, setBottomPanelOpen]);

  // Render tab buttons (shared between collapsed and expanded states)
  const renderTabButtons = (collapsed = false) => (
    <>
      {TAB_IDS.map((tabId) => (
        <button
          key={tabId}
          className={cn(
            "flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors relative",
            bottomPanelTab === tabId
              ? collapsed ? "text-foreground" : "text-foreground bg-muted"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
            tabId === "autopilot" && activeAutopilot && bottomPanelTab !== tabId && "text-green-500"
          )}
          onClick={() => handleTabClick(tabId, collapsed)}
        >
          {TAB_ICONS[tabId]}
          <span>{t(`ide.bottomPanel.${tabId}`)}</span>
          {tabId === "autopilot" && activeAutopilot && (
            <span className="relative flex h-2 w-2 ml-1">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
            </span>
          )}
        </button>
      ))}
    </>
  );

  // Collapsed state
  if (!bottomPanelOpen) {
    return (
      <div
        className={cn(
          "h-8 bg-background border-t border-border flex items-center px-2 gap-2",
          className
        )}
      >
        {renderTabButtons(true)}
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0"
          onClick={toggleBottomPanel}
        >
          <ChevronUp className="w-4 h-4" />
        </Button>
      </div>
    );
  }

  // Expanded state
  return (
    <div
      className={cn("bg-background border-t border-border flex flex-col", className)}
      style={{ height: bottomPanelHeight }}
    >
      {/* Resize handle */}
      <div
        ref={resizeRef}
        className={cn(
          "h-1 cursor-row-resize hover:bg-primary/50 transition-colors",
          isResizing && "bg-primary/50"
        )}
        onMouseDown={() => setIsResizing(true)}
      />

      {/* Tab bar */}
      <div className="h-8 flex items-center px-2 gap-2 border-b border-border">
        {renderTabButtons()}
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0"
          onClick={toggleBottomPanel}
        >
          <ChevronDown className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0"
          onClick={() => setBottomPanelOpen(false)}
        >
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* Content area */}
      <div className="flex-1 overflow-auto p-2">
        {bottomPanelTab === "channels" && (
          <ChannelsTabContent
            selectedPodKey={selectedPodKey}
            podChannels={podChannels}
            selectedChannelId={selectedChannelId}
            onChannelClick={handleChannelClick}
            onBackToList={handleBackToChannelList}
            topology={topology}
            currentChannel={currentChannel}
            messages={messages}
            messagesLoading={messagesLoading}
            onSendMessage={handleSendMessage}
            onLoadMore={handleLoadMoreMessages}
            onRefresh={handleRefreshMessages}
            onPodsChanged={handlePodsChanged}
            t={t}
          />
        )}
        {bottomPanelTab === "activity" && (
          <ActivityTabContent
            selectedPodKey={selectedPodKey}
            incomingBindings={incomingBindings}
            outgoingBindings={outgoingBindings}
            getPodInfo={getPodInfo}
            t={t}
          />
        )}
        {bottomPanelTab === "autopilot" && <AutopilotPanelContent podKey={selectedPodKey} />}
        {bottomPanelTab === "delivery" && (
          <DeliveryTabContent
            selectedPodKey={selectedPodKey}
            pod={currentPod}
            t={t}
          />
        )}
        {bottomPanelTab === "info" && (
          <InfoTabContent
            selectedPodKey={selectedPodKey}
            pod={currentPod}
            orgSlug={orgSlug}
            t={t}
          />
        )}
      </div>
    </div>
  );
}

export default BottomPanel;
