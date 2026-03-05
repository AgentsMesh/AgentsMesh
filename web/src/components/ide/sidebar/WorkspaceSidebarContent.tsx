"use client";

import React, { useEffect, useCallback, useState, useMemo } from "react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore, Pod, SIDEBAR_STATUS_MAP } from "@/stores/pod";
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
  ChevronDown,
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
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const pods = usePodStore((s) => s.pods);
  const loading = usePodStore((s) => s.loading);
  const fetchSidebarPods = usePodStore((s) => s.fetchSidebarPods);
  const loadMorePods = usePodStore((s) => s.loadMorePods);
  const terminatePod = usePodStore((s) => s.terminatePod);
  const podHasMore = usePodStore((s) => s.podHasMore);
  const loadingMore = usePodStore((s) => s.loadingMore);
  const runners = useRunnerStore((s) => s.runners);
  const runnersLoading = useRunnerStore((s) => s.loading);
  const fetchRunners = useRunnerStore((s) => s.fetchRunners);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const panes = useWorkspaceStore((s) => s.panes);

  const [filter, setFilter] = useState<FilterType>("running");
  const [searchQuery, setSearchQuery] = useState("");
  const [runnersExpanded, setRunnersExpanded] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Confirm dialog for terminate
  const { dialogProps, confirm } = useConfirmDialog();

  // Load pods and runners on mount
  useEffect(() => {
    if (currentOrg) {
      fetchSidebarPods(filter);
      fetchRunners();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrg, fetchSidebarPods, fetchRunners]);

  // Filter change handler
  const handleFilterChange = useCallback((f: FilterType) => {
    setFilter(f);
    fetchSidebarPods(f);
  }, [fetchSidebarPods]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([fetchSidebarPods(filter), fetchRunners()]);
    } finally {
      setRefreshing(false);
    }
  }, [fetchSidebarPods, filter, fetchRunners]);

  // Client-side status + search filter.
  // Status filtering is primarily server-side (fetchSidebarPods), but fetchPod and
  // WebSocket events can inject pods that don't match the current tab. Guard here.
  const filteredPods = useMemo(() => {
    const allowedStatuses = SIDEBAR_STATUS_MAP[filter];
    const statusSet = allowedStatuses
      ? new Set(allowedStatuses.split(","))
      : null; // "all" → show everything

    return pods.filter((pod) => {
      // Status guard
      if (statusSet && !statusSet.has(pod.status)) return false;

      // Search filter
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        return (
          pod.pod_key.toLowerCase().includes(query) ||
          !!pod.ticket?.slug?.toLowerCase().includes(query) ||
          !!pod.runner?.node_id?.toLowerCase().includes(query)
        );
      }

      return true;
    });
  }, [pods, searchQuery, filter]);

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

  // Handle opening terminal (or switching to it if already open)
  const handleOpenTerminal = useCallback(
    (pod: Pod) => {
      addPane(pod.pod_key);
    },
    [addPane]
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
      <WorkspaceFilters filter={filter} onFilterChange={handleFilterChange} t={t} />

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
            {podHasMore && (
              <div className="px-3 py-2">
                <Button
                  size="sm"
                  variant="ghost"
                  className="w-full h-8 text-xs text-muted-foreground"
                  onClick={loadMorePods}
                  disabled={loadingMore}
                >
                  {loadingMore ? (
                    <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  ) : (
                    <ChevronDown className="w-3 h-3 mr-1" />
                  )}
                  {t("workspace.loadMore")}
                </Button>
              </div>
            )}
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
