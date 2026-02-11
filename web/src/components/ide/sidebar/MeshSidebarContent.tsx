"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { useMeshStore, MeshNode, ChannelInfo } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Radio,
  Search,
  RefreshCw,
  Activity,
  Link2,
} from "lucide-react";
import { MeshChannelsList } from "./MeshChannelsList";
import { MeshNodesList } from "./MeshNodesList";
import { MeshSelectedDetails } from "./MeshSelectedDetails";

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
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      if (!channel.name.toLowerCase().includes(query)) return false;
    }
    return true;
  });

  // Filter nodes
  const filteredNodes = (topology?.nodes || []).filter((node) => {
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
      <MeshChannelsList
        channels={filteredChannels}
        loading={loading}
        expanded={channelsExpanded}
        onToggle={setChannelsExpanded}
        selectedChannelId={selectedChannel}
        onChannelClick={handleChannelClick}
        t={t}
      />

      {/* Nodes section */}
      <MeshNodesList
        nodes={filteredNodes}
        loading={loading}
        expanded={nodesExpanded}
        onToggle={setNodesExpanded}
        selectedNodeId={selectedNode}
        onNodeClick={handleNodeClick}
        onOpenTerminal={handleOpenTerminal}
        t={t}
      />

      {/* Selected node/channel details */}
      <MeshSelectedDetails
        selectedNode={selectedNodeData || null}
        selectedChannel={selectedChannelData || null}
        nodeChannels={selectedNodeChannels}
        onOpenTerminal={handleOpenTerminal}
        t={t}
      />
    </div>
  );
}

export default MeshSidebarContent;
