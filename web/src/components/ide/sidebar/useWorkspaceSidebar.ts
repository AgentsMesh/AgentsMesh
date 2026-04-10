"use client";

import { useState, useCallback, useMemo, useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore, Pod, SIDEBAR_STATUS_MAP } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useConfirmDialog } from "@/components/ui/confirm-dialog";
import type { FilterType } from "./WorkspaceFilters";

export function useWorkspaceSidebar(
  t: (key: string, params?: Record<string, string>) => string,
  onTerminatePod?: () => void,
) {
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const user = useAuthStore((s) => s.user);
  const isAdmin = currentOrg?.role === "owner" || currentOrg?.role === "admin";
  const pods = usePodStore((s) => s.pods);
  const loading = usePodStore((s) => s.loading);
  const fetchSidebarPods = usePodStore((s) => s.fetchSidebarPods);
  const loadMorePods = usePodStore((s) => s.loadMorePods);
  const terminatePod = usePodStore((s) => s.terminatePod);
  const updatePodAlias = usePodStore((s) => s.updatePodAlias);
  const updatePodPerpetual = usePodStore((s) => s.updatePodPerpetual);
  const podHasMore = usePodStore((s) => s.podHasMore);
  const loadingMore = usePodStore((s) => s.loadingMore);
  const runners = useRunnerStore((s) => s.runners);
  const runnersLoading = useRunnerStore((s) => s.loading);
  const fetchRunners = useRunnerStore((s) => s.fetchRunners);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const removePaneByPodKey = useWorkspaceStore((s) => s.removePaneByPodKey);
  const panes = useWorkspaceStore((s) => s.panes);

  const [filter, setFilter] = useState<FilterType>("mine");
  const [searchQuery, setSearchQuery] = useState("");
  const [runnersExpanded, setRunnersExpanded] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [renamePod, setRenamePod] = useState<Pod | null>(null);

  const { dialogProps, confirm } = useConfirmDialog();

  useEffect(() => {
    if (currentOrg) { fetchSidebarPods(filter); fetchRunners(); }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrg, fetchSidebarPods, fetchRunners]);

  const handleFilterChange = useCallback((f: FilterType) => {
    setFilter(f); fetchSidebarPods(f);
  }, [fetchSidebarPods]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try { await Promise.all([fetchSidebarPods(filter), fetchRunners()]); } finally { setRefreshing(false); }
  }, [fetchSidebarPods, filter, fetchRunners]);

  const filteredPods = useMemo(() => {
    const allowedStatuses = SIDEBAR_STATUS_MAP[filter];
    const statusSet = allowedStatuses ? new Set(allowedStatuses.split(",")) : null;
    return pods.filter((pod) => {
      if (statusSet && !statusSet.has(pod.status)) return false;
      if (filter === "mine" && user?.id && pod.created_by?.id !== user.id) return false;
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        return pod.pod_key.toLowerCase().includes(q) || !!pod.ticket?.slug?.toLowerCase().includes(q) || !!pod.runner?.node_id?.toLowerCase().includes(q);
      }
      return true;
    });
  }, [pods, searchQuery, filter, user?.id]);

  const sortedPods = useMemo(() => {
    const priority: Record<string, number> = { running: 0, initializing: 1, paused: 2, terminated: 3, failed: 3 };
    return [...filteredPods].sort((a, b) => {
      const diff = (priority[a.status] ?? 4) - (priority[b.status] ?? 4);
      if (diff !== 0) return diff;
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }, [filteredPods]);

  const isPodOpen = useCallback((podKey: string) => panes.some((p) => p.podKey === podKey), [panes]);

  const handleOpenTerminal = useCallback((pod: Pod) => { addPane(pod.pod_key); }, [addPane]);

  const handleTerminateClick = useCallback(async (podKey: string) => {
    const confirmed = await confirm({
      title: t("workspace.terminateDialog.title"),
      description: t("workspace.terminateDialog.description"),
      variant: "destructive",
      confirmText: t("workspace.terminateDialog.confirm"),
      cancelText: t("workspace.terminateDialog.cancel"),
    });
    if (confirmed) { await terminatePod(podKey); removePaneByPodKey(podKey); onTerminatePod?.(); }
  }, [confirm, t, terminatePod, removePaneByPodKey, onTerminatePod]);

  const handleRenameConfirm = useCallback(async (newName: string) => {
    if (!renamePod) return;
    try { await updatePodAlias(renamePod.pod_key, newName || null); } catch (error) { console.error("Failed to rename pod:", error); }
    setRenamePod(null);
  }, [renamePod, updatePodAlias]);

  const handleTogglePerpetual = useCallback(async (podKey: string, perpetual: boolean) => {
    try { await updatePodPerpetual(podKey, perpetual); } catch (error) { console.error("Failed to toggle perpetual:", error); }
  }, [updatePodPerpetual]);

  return {
    currentOrg, loading, runners, runnersLoading, isAdmin,
    filter, searchQuery, setSearchQuery, runnersExpanded, setRunnersExpanded, refreshing,
    renamePod, setRenamePod, dialogProps, sortedPods, podHasMore, loadingMore,
    handleFilterChange, handleRefresh, isPodOpen, handleOpenTerminal,
    handleTerminateClick, handleRenameConfirm, handleTogglePerpetual, loadMorePods,
  };
}
