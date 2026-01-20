"use client";

import React, { useEffect, useCallback, useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { getPodDisplayName } from "@/lib/pod-utils";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore, Pod } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Square,
  Terminal,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  Server,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useTranslations } from "@/lib/i18n/client";

interface WorkspaceSidebarContentProps {
  className?: string;
  onCreatePod?: () => void;
}

// Status badge colors - matches PodData status type
const statusColors: Record<string, { bg: string; text: string; dot: string }> = {
  initializing: { bg: "bg-yellow-500/10", text: "text-yellow-600 dark:text-yellow-400", dot: "bg-yellow-500" },
  running: { bg: "bg-blue-500/10", text: "text-blue-600 dark:text-blue-400", dot: "bg-blue-500" },
  paused: { bg: "bg-orange-500/10", text: "text-orange-600 dark:text-orange-400", dot: "bg-orange-500" },
  disconnected: { bg: "bg-gray-500/10", text: "text-gray-600 dark:text-gray-400", dot: "bg-gray-500" },
  orphaned: { bg: "bg-purple-500/10", text: "text-purple-600 dark:text-purple-400", dot: "bg-purple-500" },
  completed: { bg: "bg-green-500/10", text: "text-green-600 dark:text-green-400", dot: "bg-green-500" },
  terminated: { bg: "bg-gray-500/10", text: "text-gray-600 dark:text-gray-400", dot: "bg-gray-500" },
  error: { bg: "bg-red-500/10", text: "text-red-600 dark:text-red-400", dot: "bg-red-500" },
  failed: { bg: "bg-red-500/10", text: "text-red-600 dark:text-red-400", dot: "bg-red-500" },
};

type FilterType = "all" | "running" | "completed";

export function WorkspaceSidebarContent({ className, onCreatePod }: WorkspaceSidebarContentProps) {
  const t = useTranslations();
  useRouter(); // Reserved for navigation
  const { currentOrg } = useAuthStore();
  const { pods, loading, fetchPods, terminatePod } = usePodStore();
  const { runners, loading: runnersLoading, fetchRunners } = useRunnerStore();
  const { addPane, panes } = useWorkspaceStore();

  const [filter, setFilter] = useState<FilterType>("running");
  const [searchQuery, setSearchQuery] = useState("");
  const [runnersExpanded, setRunnersExpanded] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

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

  // Handle terminate
  const handleTerminate = useCallback(
    async (podKey: string, e: React.MouseEvent) => {
      e.stopPropagation();
      await terminatePod(podKey);
    },
    [terminatePod]
  );

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "initializing":
        return <Clock className="w-3 h-3" />;
      case "running":
        return <Loader2 className="w-3 h-3 animate-spin" />;
      case "paused":
        return <Square className="w-3 h-3" />;
      case "terminated":
        return <CheckCircle className="w-3 h-3" />;
      case "failed":
        return <XCircle className="w-3 h-3" />;
      default:
        return <Square className="w-3 h-3" />;
    }
  };

  const onlineRunners = runners.filter(r => r.status === "online");

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
      <div className="flex items-center gap-1 px-2 py-1 border-y border-border">
        {(["running", "completed", "all"] as const).map((f) => (
          <button
            key={f}
            className={cn(
              "px-2 py-1 text-xs rounded transition-colors",
              filter === f
                ? "bg-muted text-foreground font-medium"
                : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
            )}
            onClick={() => setFilter(f)}
          >
            {t(`workspace.filters.${f}`)}
          </button>
        ))}
      </div>

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
            {sortedPods.map((pod) => {
              const status = statusColors[pod.status] || statusColors.terminated;
              const isOpen = isPodOpen(pod.pod_key);

              return (
                <div
                  key={pod.pod_key}
                  className={cn(
                    "group flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer",
                    isOpen && "bg-muted/30"
                  )}
                  onClick={() => handleOpenTerminal(pod)}
                >
                  {/* Status indicator */}
                  <div className={cn("flex items-center justify-center", status.text)}>
                    {getStatusIcon(pod.status)}
                  </div>

                  {/* Pod info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm truncate font-mono">
                        {getPodDisplayName(pod)}
                      </span>
                      {isOpen && (
                        <Terminal className="w-3 h-3 text-primary flex-shrink-0" />
                      )}
                    </div>
                    {/* Show ticket if title is not from ticket */}
                    {!pod.title && pod.ticket?.identifier && (
                      <p className="text-xs text-muted-foreground truncate">
                        {pod.ticket.identifier}
                      </p>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    {pod.status === "running" && (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 w-6 p-0 text-destructive hover:text-destructive"
                        onClick={(e) => handleTerminate(pod.pod_key, e)}
                      >
                        <Square className="w-3 h-3" />
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Runners section */}
      <Collapsible open={runnersExpanded} onOpenChange={setRunnersExpanded}>
        <CollapsibleTrigger asChild>
          <div className="flex items-center justify-between px-3 py-2 border-t border-border cursor-pointer hover:bg-muted/50">
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">{t("workspace.runners.title")}</span>
              <span className="text-xs text-muted-foreground">
                ({onlineRunners.length} {t("workspace.runners.online")})
              </span>
            </div>
            {runnersExpanded ? (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border-t border-border">
            {runnersLoading ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
              </div>
            ) : runners.length === 0 ? (
              <div className="px-3 py-4 text-center">
                <p className="text-xs text-muted-foreground">{t("workspace.runners.noRunners")}</p>
              </div>
            ) : (
              <div className="py-1 max-h-32 overflow-y-auto">
                {runners.map((runner) => (
                  <div
                    key={runner.id}
                    className="flex items-center gap-2 px-3 py-1.5 text-sm"
                  >
                    <span
                      className={cn(
                        "w-2 h-2 rounded-full flex-shrink-0",
                        runner.status === "online" ? "bg-green-500" : "bg-gray-400"
                      )}
                    />
                    <span className="truncate flex-1">{runner.node_id}</span>
                    <span className="text-xs text-muted-foreground">
                      {runner.current_pods}/{runner.max_concurrent_pods}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}

export default WorkspaceSidebarContent;
