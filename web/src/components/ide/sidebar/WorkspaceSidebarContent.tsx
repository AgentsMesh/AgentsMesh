"use client";

import React, { useEffect, useCallback, useState, useMemo } from "react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore, Pod } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Terminal,
  Loader2,
  Plus,
  RefreshCw,
  Search,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { PodListItem } from "./PodListItem";
import { RunnerSection } from "./RunnerSection";
import { WorkspaceFilters, type FilterType } from "./WorkspaceFilters";

interface WorkspaceSidebarContentProps {
  className?: string;
  onCreatePod?: () => void;
  onTerminatePod?: () => void;
}

export function WorkspaceSidebarContent({ className, onCreatePod, onTerminatePod }: WorkspaceSidebarContentProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const { pods, loading, fetchPods, terminatePod } = usePodStore();
  const { runners, loading: runnersLoading, fetchRunners } = useRunnerStore();
  const { addPane, panes } = useWorkspaceStore();

  const [filter, setFilter] = useState<FilterType>("running");
  const [searchQuery, setSearchQuery] = useState("");
  const [runnersExpanded, setRunnersExpanded] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Confirm dialog for terminate
  const { dialogProps, confirm } = useConfirmDialog();

  // Load pods and runners on mount
  useEffect(() => {
    if (currentOrg) {
      fetchPods();
      fetchRunners();
    }
  }, [currentOrg, fetchPods, fetchRunners]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([fetchPods(), fetchRunners()]);
    } finally {
      setRefreshing(false);
    }
  }, [fetchPods, fetchRunners]);

  // Filter and search pods
  const filteredPods = pods.filter((pod) => {
    // Status filter
    if (filter === "running" && pod.status !== "running" && pod.status !== "initializing") {
      return false;
    }
    if (filter === "completed" && pod.status !== "terminated" && pod.status !== "failed" && pod.status !== "paused") {
      return false;
    }

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesPodKey = pod.pod_key.toLowerCase().includes(query);
      const matchesTicket = pod.ticket?.identifier?.toLowerCase().includes(query);
      const matchesRunner = pod.runner?.node_id?.toLowerCase().includes(query);
      return matchesPodKey || matchesTicket || matchesRunner;
    }

    return true;
  });

  // Sort pods: running/initializing first, then by creation time (newest first)
  const sortedPods = useMemo(() => {
    const statusPriority: Record<string, number> = {
      running: 0,
      initializing: 1,
      paused: 2,
      terminated: 3,
      failed: 3,
    };

    return [...filteredPods].sort((a, b) => {
      const priorityDiff = (statusPriority[a.status] ?? 4) - (statusPriority[b.status] ?? 4);
      if (priorityDiff !== 0) return priorityDiff;
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }, [filteredPods]);

  // Check if pod is already open in workspace
  const isPodOpen = useCallback(
    (podKey: string) => panes.some((p) => p.podKey === podKey),
    [panes]
  );

  // Handle opening terminal
  const handleOpenTerminal = useCallback(
    (pod: Pod) => {
      if (!isPodOpen(pod.pod_key)) {
        addPane(pod.pod_key, pod.pod_key);
      }
    },
    [addPane, isPodOpen]
  );

  // Handle terminate with confirmation
  const handleTerminateClick = useCallback(
    async (podKey: string, e: React.MouseEvent) => {
      e.stopPropagation();
      const confirmed = await confirm({
        title: t("workspace.terminateDialog.title"),
        description: t("workspace.terminateDialog.description"),
        variant: "destructive",
        confirmText: t("workspace.terminateDialog.confirm"),
        cancelText: t("workspace.terminateDialog.cancel"),
      });
      if (confirmed) {
        await terminatePod(podKey);
        onTerminatePod?.();
      }
    },
    [confirm, t, terminatePod, onTerminatePod]
  );

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder={t("workspace.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-1 px-2 pb-2">
        <Button
          size="sm"
          variant="outline"
          className="flex-1 h-8 text-xs"
          onClick={onCreatePod}
        >
          <Plus className="w-3 h-3 mr-1" />
          {t("workspace.newPod")}
        </Button>
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

      {/* Filter tabs */}
      <WorkspaceFilters filter={filter} onFilterChange={setFilter} t={t} />

      {/* Pod list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : sortedPods.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Terminal className="w-8 h-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              {searchQuery
                ? t("workspace.emptyState.noMatch")
                : filter === "all"
                  ? t("workspace.emptyState.title")
                  : t("workspace.emptyState.noFiltered", { filter: t(`workspace.filters.${filter}`) })}
            </p>
            {!searchQuery && filter === "all" && (
              <Button
                size="sm"
                variant="outline"
                className="mt-3"
                onClick={onCreatePod}
              >
                {t("workspace.emptyState.createFirst")}
              </Button>
            )}
          </div>
        ) : (
          <div className="py-1">
            {sortedPods.map((pod) => (
              <PodListItem
                key={pod.pod_key}
                pod={pod}
                isOpen={isPodOpen(pod.pod_key)}
                onClick={() => handleOpenTerminal(pod)}
                onTerminate={(e) => handleTerminateClick(pod.pod_key, e)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Runners section */}
      <RunnerSection
        runners={runners}
        loading={runnersLoading}
        expanded={runnersExpanded}
        onToggle={setRunnersExpanded}
        currentOrgSlug={currentOrg?.slug}
        t={t}
      />

      {/* Confirm Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

export default WorkspaceSidebarContent;
