"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { useMeshStore, MeshNode } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Search,
  RefreshCw,
  Activity,
  Link2,
  Radio,
} from "lucide-react";
import { MeshNodesList } from "./MeshNodesList";
import { MeshSelectedDetails } from "./MeshSelectedDetails";

interface MeshSidebarContentProps {
  className?: string;
}

export function MeshSidebarContent({ className }: MeshSidebarContentProps) {
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const topology = useMeshStore((s) => s.topology);
  const loading = useMeshStore((s) => s.loading);
  const selectedNode = useMeshStore((s) => s.selectedNode);
  const fetchTopology = useMeshStore((s) => s.fetchTopology);
  const selectNode = useMeshStore((s) => s.selectNode);
  const addPane = useWorkspaceStore((s) => s.addPane);

  // State
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [nodesExpanded, setNodesExpanded] = useState(true);

  // Load topology on mount
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

  // Filter nodes
  const filteredNodes = useMemo(() => {
    return (topology?.nodes || []).filter((node) => {
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const matchesPodKey = node.pod_key.toLowerCase().includes(query);
        const matchesModel = node.model?.toLowerCase().includes(query);
        if (!matchesPodKey && !matchesModel) return false;
      }
      return true;
    });
  }, [topology?.nodes, searchQuery]);

  // Stats
  const activeNodes = useMemo(
    () => topology?.nodes.filter(n => n.status === "running" || n.status === "initializing").length || 0,
    [topology?.nodes]
  );
  const totalChannels = topology?.channels.length || 0;
  const totalBindings = topology?.edges.length || 0;

  const handleNodeClick = (node: MeshNode) => {
    selectNode(node.pod_key);
  };

  const handleOpenTerminal = (podKey: string, e: React.MouseEvent) => {
    e.stopPropagation();
    addPane(podKey);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  // Selected node details
  const selectedNodeData = useMemo(
    () => selectedNode ? topology?.nodes.find(n => n.pod_key === selectedNode) ?? null : null,
    [selectedNode, topology?.nodes]
  );
  const selectedNodeChannels = useMemo(
    () => {
      if (!selectedNode || !topology) return [];
      return topology.channels.filter(c => c.pod_keys.includes(selectedNode));
    },
    [selectedNode, topology]
  );

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

      {/* Selected node details */}
      <MeshSelectedDetails
        selectedNode={selectedNodeData}
        nodeChannels={selectedNodeChannels}
        onOpenTerminal={handleOpenTerminal}
        t={t}
      />
    </div>
  );
}

export default MeshSidebarContent;
