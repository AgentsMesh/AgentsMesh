"use client";

import React, { useRef, useEffect, useState, useMemo, useCallback } from "react";
import { cn } from "@/lib/utils";
import { useIDEStore, type BottomPanelTab } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { useMeshStore, type ChannelInfo, type MeshEdge } from "@/stores/mesh";
import { useChannelStore } from "@/stores/channel";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { ChannelHeader } from "@/components/mesh/ChannelHeader";
import { MessageList } from "@/components/mesh/MessageList";
import { MessageInput } from "@/components/mesh/MessageInput";
import { Markdown } from "@/components/ui/markdown";
import {
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  X,
  Terminal,
  MessageSquare,
  Activity,
  ArrowRight,
  ArrowLeft,
  Hash,
  Users,
} from "lucide-react";

interface BottomPanelProps {
  className?: string;
}

const TAB_ICONS: Record<BottomPanelTab, React.ReactNode> = {
  channels: <MessageSquare className="w-3.5 h-3.5" />,
  activity: <Activity className="w-3.5 h-3.5" />,
};

const TAB_IDS: BottomPanelTab[] = ["channels", "activity"];

export function BottomPanel({ className }: BottomPanelProps) {
  const t = useTranslations();
  const {
    bottomPanelOpen,
    bottomPanelHeight,
    bottomPanelTab,
    setBottomPanelOpen,
    setBottomPanelHeight,
    setBottomPanelTab,
    toggleBottomPanel,
  } = useIDEStore();

  const { panes, activePane } = useWorkspaceStore();
  const { topology, getChannelsForNode, getEdgesForNode, fetchTopology } = useMeshStore();

  // Channel detail state
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const {
    currentChannel,
    messages,
    messagesLoading,
    fetchChannel,
    fetchMessages,
    sendMessage,
    setCurrentChannel,
  } = useChannelStore();

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

  // Get channels and bindings for selected pod
  const podChannels = useMemo(() => {
    if (!selectedPodKey) return [];
    return getChannelsForNode(selectedPodKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- topology triggers recalculation when data changes
  }, [selectedPodKey, getChannelsForNode, topology]);

  const podEdges = useMemo(() => {
    if (!selectedPodKey) return [];
    return getEdgesForNode(selectedPodKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- topology triggers recalculation when data changes
  }, [selectedPodKey, getEdgesForNode, topology]);

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
  const getPodInfo = (podKey: string) => {
    return topology?.nodes.find((n) => n.pod_key === podKey);
  };

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

  // Channel detail handlers
  const handleChannelClick = (channelId: number) => {
    setSelectedChannelId(channelId);
  };

  const handleBackToChannelList = () => {
    setSelectedChannelId(null);
    setCurrentChannel(null);
  };

  const handleSendMessage = useCallback(
    async (content: string) => {
      if (!selectedChannelId) return;
      try {
        await sendMessage(selectedChannelId, content);
      } catch (error) {
        console.error("Failed to send message:", error);
      }
    },
    [selectedChannelId, sendMessage]
  );

  const handleLoadMoreMessages = useCallback(() => {
    if (!selectedChannelId) return;
    fetchMessages(selectedChannelId, 50, messages.length);
  }, [selectedChannelId, messages.length, fetchMessages]);

  const handleRefreshMessages = useCallback(() => {
    if (!selectedChannelId) return;
    fetchMessages(selectedChannelId);
  }, [selectedChannelId, fetchMessages]);

  // Transform messages for MessageList component
  const transformedMessages = messages.map((msg) => ({
    id: msg.id,
    content: msg.content,
    messageType: msg.message_type as "text" | "system" | "code" | "command",
    metadata: msg.metadata,
    createdAt: msg.created_at,
    pod: msg.pod
      ? {
          podKey: msg.pod.pod_key,
          agentType: msg.pod.agent_type ? { name: msg.pod.agent_type.name } : undefined,
        }
      : undefined,
    user: msg.user
      ? {
          id: msg.user.id,
          username: msg.user.username,
          name: msg.user.name,
          avatarUrl: msg.user.avatar_url,
        }
      : undefined,
  }));

  // Render empty state when no pod is selected
  const renderSelectPodFirst = () => (
    <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
      <Terminal className="w-4 h-4 mr-2" />
      <span>{t("ide.bottomPanel.selectPodFirst")}</span>
    </div>
  );

  // Render channel detail view
  const renderChannelDetail = () => {
    const channelInfo = topology?.channels.find((c) => c.id === selectedChannelId);
    const podCount = channelInfo?.pod_keys.length || currentChannel?.pods?.length || 0;

    return (
      <div className="flex flex-col h-full">
        {/* Channel Header with back button - softer styling */}
        <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/30">
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 hover:bg-muted"
            onClick={handleBackToChannelList}
          >
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <div className="flex-1 min-w-0">
            <ChannelHeader
              name={currentChannel?.name || channelInfo?.name || "Channel"}
              description={currentChannel?.description}
              podCount={podCount}
              onClose={handleBackToChannelList}
              onRefresh={handleRefreshMessages}
              loading={messagesLoading}
              compact
            />
          </div>
        </div>

        {/* Document section - collapsible if exists */}
        {currentChannel?.document && (
          <div className="px-3 py-2 bg-muted/20">
            <details className="text-xs">
              <summary className="cursor-pointer text-muted-foreground hover:text-foreground flex items-center gap-1">
                <span>{t("ide.bottomPanel.channelDocument")}</span>
              </summary>
              <div className="mt-2 text-muted-foreground">
                <Markdown content={currentChannel.document} compact />
              </div>
            </details>
          </div>
        )}

        {/* Messages */}
        <div className="flex-1 overflow-hidden">
          <MessageList
            messages={transformedMessages}
            loading={messagesLoading}
            hasMore={messages.length >= 50 && messages.length % 50 === 0}
            onLoadMore={handleLoadMoreMessages}
          />
        </div>

        {/* Input - softer top border */}
        <div className="flex-shrink-0 bg-muted/20">
          <MessageInput
            onSend={handleSendMessage}
            placeholder={t("ide.bottomPanel.sendMessagePlaceholder")}
          />
        </div>
      </div>
    );
  };

  // Render channel list for selected pod
  const renderChannelsContent = () => {
    // If a channel is selected, show channel detail
    if (selectedChannelId) {
      return renderChannelDetail();
    }

    if (!selectedPodKey) {
      return renderSelectPodFirst();
    }

    if (podChannels.length === 0) {
      return (
        <div className="text-xs text-muted-foreground">
          <p>{t("ide.bottomPanel.noChannels")}</p>
        </div>
      );
    }

    return (
      <div className="space-y-2">
        <p className="text-xs text-muted-foreground mb-2">
          {t("ide.bottomPanel.podChannels", { count: podChannels.length })}
        </p>
        <div className="space-y-1">
          {podChannels.map((channel: ChannelInfo) => (
            <button
              key={channel.id}
              className="w-full flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 hover:bg-muted transition-colors cursor-pointer text-left"
              onClick={() => handleChannelClick(channel.id)}
            >
              <Hash className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="text-xs font-medium flex-1">{channel.name}</span>
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                <Users className="w-3 h-3" />
                <span>{t("ide.bottomPanel.members")}: {channel.pod_keys.length}</span>
              </div>
              <ChevronRight className="w-3 h-3 text-muted-foreground" />
            </button>
          ))}
        </div>
      </div>
    );
  };

  // Render binding relationships (activity tab)
  const renderActivityContent = () => {
    if (!selectedPodKey) {
      return renderSelectPodFirst();
    }

    const hasBindings = incomingBindings.length > 0 || outgoingBindings.length > 0;

    if (!hasBindings) {
      return (
        <div className="text-xs text-muted-foreground">
          <p>{t("ide.bottomPanel.noBindings")}</p>
        </div>
      );
    }

    return (
      <div className="space-y-4">
        {/* Incoming bindings */}
        {incomingBindings.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2 flex items-center gap-1">
              <ArrowRight className="w-3 h-3" />
              {t("ide.bottomPanel.incomingBindings")} ({incomingBindings.length})
            </h4>
            <div className="space-y-1">
              {incomingBindings.map((edge) => {
                const sourcePod = getPodInfo(edge.source);
                return (
                  <div
                    key={`${edge.source}-${edge.target}`}
                    className="flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 text-xs"
                  >
                    <span className="font-mono text-muted-foreground">
                      {edge.source.substring(0, 8)}
                    </span>
                    {sourcePod?.model && (
                      <span className="text-muted-foreground">
                        ({sourcePod.model})
                      </span>
                    )}
                    <ArrowRight className="w-3 h-3 text-green-500" />
                    <span className="font-medium">{t("ide.bottomPanel.currentPod")}</span>
                    <span className={cn(
                      "ml-auto px-1.5 py-0.5 rounded text-[10px]",
                      edge.status === "active"
                        ? "bg-green-500/10 text-green-500"
                        : "bg-yellow-500/10 text-yellow-500"
                    )}>
                      {edge.status}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Outgoing bindings */}
        {outgoingBindings.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2 flex items-center gap-1">
              <ArrowLeft className="w-3 h-3" />
              {t("ide.bottomPanel.outgoingBindings")} ({outgoingBindings.length})
            </h4>
            <div className="space-y-1">
              {outgoingBindings.map((edge) => {
                const targetPod = getPodInfo(edge.target);
                return (
                  <div
                    key={`${edge.source}-${edge.target}`}
                    className="flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 text-xs"
                  >
                    <span className="font-medium">{t("ide.bottomPanel.currentPod")}</span>
                    <ArrowRight className="w-3 h-3 text-blue-500" />
                    <span className="font-mono text-muted-foreground">
                      {edge.target.substring(0, 8)}
                    </span>
                    {targetPod?.model && (
                      <span className="text-muted-foreground">
                        ({targetPod.model})
                      </span>
                    )}
                    <span className={cn(
                      "ml-auto px-1.5 py-0.5 rounded text-[10px]",
                      edge.status === "active"
                        ? "bg-green-500/10 text-green-500"
                        : "bg-yellow-500/10 text-yellow-500"
                    )}>
                      {edge.status}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    );
  };

  if (!bottomPanelOpen) {
    return (
      <div
        className={cn(
          "h-8 bg-background border-t border-border flex items-center px-2 gap-2",
          className
        )}
      >
        {TAB_IDS.map((tabId) => (
          <button
            key={tabId}
            className={cn(
              "flex items-center gap-1.5 px-2 py-1 text-xs rounded hover:bg-muted",
              bottomPanelTab === tabId
                ? "text-foreground"
                : "text-muted-foreground"
            )}
            onClick={() => {
              setBottomPanelTab(tabId);
              setBottomPanelOpen(true);
              // Reset channel selection when switching tabs
              if (tabId !== "channels") {
                setSelectedChannelId(null);
              }
            }}
          >
            {TAB_ICONS[tabId]}
            <span>{t(`ide.bottomPanel.${tabId}`)}</span>
          </button>
        ))}
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
        {TAB_IDS.map((tabId) => (
          <button
            key={tabId}
            className={cn(
              "flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors",
              bottomPanelTab === tabId
                ? "text-foreground bg-muted"
                : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
            )}
            onClick={() => {
              setBottomPanelTab(tabId);
              // Reset channel selection when switching tabs
              if (tabId !== "channels") {
                setSelectedChannelId(null);
              }
            }}
          >
            {TAB_ICONS[tabId]}
            <span>{t(`ide.bottomPanel.${tabId}`)}</span>
          </button>
        ))}
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
        {bottomPanelTab === "channels" && renderChannelsContent()}
        {bottomPanelTab === "activity" && renderActivityContent()}
      </div>
    </div>
  );
}

export default BottomPanel;
