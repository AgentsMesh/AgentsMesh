"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "@/lib/i18n/client";
import { useMeshStore, MeshNode, ChannelInfo, getPodStatusInfo, getAgentStatusInfo } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Radio,
  Loader2,
  Search,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Terminal,
  Activity,
  Users,
  Link2,
} from "lucide-react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

interface MeshSidebarContentProps {
  className?: string;
}

export function MeshSidebarContent({ className }: MeshSidebarContentProps) {
  const router = useRouter();
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const {
    topology,
    loading,
    selectedNode,
    selectedChannel,
    fetchTopology,
    selectNode,
    selectChannel,
    getChannelsForNode,
  } = useMeshStore();
  const { addPane } = useWorkspaceStore();

  // State
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [channelsExpanded, setChannelsExpanded] = useState(true);
  const [nodesExpanded, setNodesExpanded] = useState(true);

  // Load topology on mount - realtime events handle subsequent updates
  useEffect(() => {
    if (currentOrg) {
      fetchTopology();
    }
  }, [currentOrg, fetchTopology]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchTopology();
    } finally {
      setRefreshing(false);
    }
  }, [fetchTopology]);

  // Filter channels
  const filteredChannels = (topology?.channels || []).filter((channel) => {
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      if (!channel.name.toLowerCase().includes(query)) return false;
    }
    return true;
  });

  // Filter nodes
  const filteredNodes = (topology?.nodes || []).filter((node) => {
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesPodKey = node.pod_key.toLowerCase().includes(query);
      const matchesModel = node.model?.toLowerCase().includes(query);
      if (!matchesPodKey && !matchesModel) return false;
    }
    return true;
  });

  // Stats
  const activeNodes = topology?.nodes.filter(n => n.status === "running" || n.status === "initializing").length || 0;
  const totalChannels = topology?.channels.length || 0;
  const totalBindings = topology?.edges.length || 0;

  // Handle node click
  const handleNodeClick = (node: MeshNode) => {
    selectNode(node.pod_key);
  };

  // Handle channel click
  const handleChannelClick = (channel: ChannelInfo) => {
    selectChannel(channel.id);
  };

  // Open terminal for pod
  const handleOpenTerminal = (podKey: string, e: React.MouseEvent) => {
    e.stopPropagation();
    addPane(podKey, podKey);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  // Get selected node details
  const selectedNodeData = selectedNode
    ? topology?.nodes.find(n => n.pod_key === selectedNode)
    : null;
  const selectedNodeChannels = selectedNode ? getChannelsForNode(selectedNode) : [];

  // Get selected channel details
  const selectedChannelData = selectedChannel
    ? topology?.channels.find(c => c.id === selectedChannel)
    : null;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder={t("ide.sidebar.mesh.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      {/* Refresh button */}
      <div className="flex items-center justify-end px-2 pb-2">
        <Button
          size="sm"
          variant="ghost"
          className="h-8 w-8 p-0"
          onClick={handleRefresh}
          disabled={refreshing}
        >
          <RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
        </Button>
      </div>

      {/* Network stats */}
      <div className="px-3 py-2 border-t border-border space-y-2">
        <div className="text-xs font-medium text-muted-foreground">{t("ide.sidebar.mesh.networkStats")}</div>
        <div className="grid grid-cols-3 gap-2">
          <div className="flex flex-col items-center text-xs">
            <Activity className="w-3.5 h-3.5 text-green-500 dark:text-green-400 mb-0.5" />
            <span className="font-medium">{activeNodes}</span>
            <span className="text-muted-foreground">{t("ide.sidebar.mesh.active")}</span>
          </div>
          <div className="flex flex-col items-center text-xs">
            <Radio className="w-3.5 h-3.5 text-blue-500 dark:text-blue-400 mb-0.5" />
            <span className="font-medium">{totalChannels}</span>
            <span className="text-muted-foreground">{t("ide.sidebar.mesh.channels")}</span>
          </div>
          <div className="flex flex-col items-center text-xs">
            <Link2 className="w-3.5 h-3.5 text-purple-500 dark:text-purple-400 mb-0.5" />
            <span className="font-medium">{totalBindings}</span>
            <span className="text-muted-foreground">{t("ide.sidebar.mesh.bindings")}</span>
          </div>
        </div>
      </div>

      {/* Channels section */}
      <Collapsible open={channelsExpanded} onOpenChange={setChannelsExpanded}>
        <CollapsibleTrigger asChild>
          <div className="flex items-center justify-between px-3 py-2 border-t border-border cursor-pointer hover:bg-muted/50">
            <div className="flex items-center gap-2">
              <Radio className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">{t("ide.sidebar.mesh.channelsSection")}</span>
              <span className="text-xs text-muted-foreground">
                ({filteredChannels.length})
              </span>
            </div>
            {channelsExpanded ? (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="max-h-40 overflow-y-auto">
            {loading && filteredChannels.length === 0 ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
              </div>
            ) : filteredChannels.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-muted-foreground">
                {t("ide.sidebar.mesh.noChannels")}
              </div>
            ) : (
              <div className="py-1">
                {filteredChannels.map((channel) => {
                  const isSelected = selectedChannel === channel.id;
                  return (
                    <div
                      key={channel.id}
                      className={cn(
                        "flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-muted/50",
                        isSelected && "bg-muted/30"
                      )}
                      onClick={() => handleChannelClick(channel)}
                    >
                      <Radio className="w-3 h-3 text-blue-500 dark:text-blue-400" />
                      <span className="text-sm truncate flex-1">{channel.name}</span>
                      <span className="text-xs text-muted-foreground">
                        {t("ide.sidebar.mesh.podsCount", { count: channel.pod_keys.length })}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Nodes section */}
      <Collapsible open={nodesExpanded} onOpenChange={setNodesExpanded}>
        <CollapsibleTrigger asChild>
          <div className="flex items-center justify-between px-3 py-2 border-t border-border cursor-pointer hover:bg-muted/50">
            <div className="flex items-center gap-2">
              <Users className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">{t("ide.sidebar.mesh.podsSection")}</span>
              <span className="text-xs text-muted-foreground">
                ({filteredNodes.length})
              </span>
            </div>
            {nodesExpanded ? (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="flex-1 overflow-y-auto max-h-48">
            {loading && filteredNodes.length === 0 ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
              </div>
            ) : filteredNodes.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-muted-foreground">
                {t("ide.sidebar.mesh.noPods")}
              </div>
            ) : (
              <div className="py-1">
                {filteredNodes.map((node) => {
                  const isSelected = selectedNode === node.pod_key;
                  const statusInfo = getPodStatusInfo(node.status);
                  const agentInfo = getAgentStatusInfo(node.agent_status);
                  return (
                    <div
                      key={node.pod_key}
                      className={cn(
                        "group flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-muted/50",
                        isSelected && "bg-muted/30"
                      )}
                      onClick={() => handleNodeClick(node)}
                    >
                      <span className={cn("w-2 h-2 rounded-full flex-shrink-0", statusInfo.bgColor)} />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm truncate">{node.pod_key.substring(0, 12)}...</p>
                        {node.model && (
                          <p className="text-xs text-muted-foreground">{node.model}</p>
                        )}
                      </div>
                      {/* Open terminal button */}
                      {(node.status === "running" || node.status === "initializing") && (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
                          onClick={(e) => handleOpenTerminal(node.pod_key, e)}
                        >
                          <Terminal className="w-3 h-3" />
                        </Button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Selected node/channel details */}
      {(selectedNodeData || selectedChannelData) && (
        <div className="border-t border-border p-3 space-y-2">
          <div className="text-xs font-medium text-muted-foreground">
            {selectedNodeData ? t("ide.sidebar.mesh.selectedPod") : t("ide.sidebar.mesh.selectedChannel")}
          </div>

          {selectedNodeData && (
            <div className="space-y-2">
              <div className="text-sm font-medium truncate">{selectedNodeData.pod_key}</div>
              <div className="flex items-center gap-2 text-xs">
                <span className={cn("px-1.5 py-0.5 rounded", getPodStatusInfo(selectedNodeData.status).bgColor, getPodStatusInfo(selectedNodeData.status).color)}>
                  {getPodStatusInfo(selectedNodeData.status).label}
                </span>
                {selectedNodeData.model && (
                  <span className="text-muted-foreground">{selectedNodeData.model}</span>
                )}
              </div>
              {selectedNodeChannels.length > 0 && (
                <div className="text-xs text-muted-foreground">
                  {t("ide.sidebar.mesh.channelsLabel")}: {selectedNodeChannels.map(c => c.name).join(", ")}
                </div>
              )}
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  className="h-7 text-xs flex-1"
                  onClick={(e) => handleOpenTerminal(selectedNodeData.pod_key, e)}
                >
                  <Terminal className="w-3 h-3 mr-1" />
                  {t("ide.sidebar.mesh.terminal")}
                </Button>
              </div>
            </div>
          )}

          {selectedChannelData && (
            <div className="space-y-2">
              <div className="text-sm font-medium">{selectedChannelData.name}</div>
              <div className="text-xs text-muted-foreground">
                {t("ide.sidebar.mesh.connectedPods", { count: selectedChannelData.pod_keys.length })}
              </div>
              <div className="text-xs text-muted-foreground break-all">
                {t("ide.sidebar.mesh.podsLabel")}: {selectedChannelData.pod_keys.map(k => k.substring(0, 8)).join(", ")}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default MeshSidebarContent;
