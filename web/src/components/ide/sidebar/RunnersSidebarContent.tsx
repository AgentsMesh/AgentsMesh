"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useRunnerStore, Runner, RunnerStatus, getRunnerStatusInfo, formatHostInfo } from "@/stores/runner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Server,
  Cpu,
  MemoryStick,
  Loader2,
  Plus,
  Search,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Key,
  Activity,
} from "lucide-react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

interface RunnersSidebarContentProps {
  className?: string;
}

// Status filter options
const statusOptions = [
  { value: "all", label: "All" },
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
];

export function RunnersSidebarContent({ className }: RunnersSidebarContentProps) {
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const {
    runners,
    tokens,
    loading,
    fetchRunners,
    fetchTokens,
  } = useRunnerStore();

  // State
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | RunnerStatus>("all");
  const [selectedRunnerId, setSelectedRunnerId] = useState<number | null>(null);
  const [tokensExpanded, setTokensExpanded] = useState(false);

  // Load runners on mount
  useEffect(() => {
    if (currentOrg) {
      fetchRunners();
      fetchTokens();
    }
  }, [currentOrg, fetchRunners, fetchTokens]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([fetchRunners(), fetchTokens()]);
    } finally {
      setRefreshing(false);
    }
  }, [fetchRunners, fetchTokens]);

  // Filter runners
  const filteredRunners = runners.filter((runner) => {
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesNodeId = runner.node_id.toLowerCase().includes(query);
      const matchesDescription = runner.description?.toLowerCase().includes(query);
      if (!matchesNodeId && !matchesDescription) return false;
    }

    // Status filter
    if (statusFilter !== "all" && runner.status !== statusFilter) {
      return false;
    }

    return true;
  });

  // Computed stats
  const onlineCount = runners.filter(r => r.status === "online").length;
  const totalPods = runners.reduce((sum, r) => sum + r.current_pods, 0);
  const totalCapacity = runners.reduce((sum, r) => sum + r.max_concurrent_pods, 0);

  // Handle runner click
  const handleRunnerClick = (runner: Runner) => {
    setSelectedRunnerId(runner.id);
    router.push(`/${currentOrg?.slug}/runners?selected=${runner.id}`);
  };

  // Navigate to add runner page
  const handleAddRunner = () => {
    router.push(`/${currentOrg?.slug}/settings?tab=runners`);
  };

  // Active tokens count
  const activeTokens = tokens.filter(t => t.is_active).length;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search runners..."
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
          onClick={handleAddRunner}
        >
          <Plus className="w-3 h-3 mr-1" />
          Add Runner
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

      {/* Status filter */}
      <div className="px-2 pb-2">
        <div className="flex items-center gap-1">
          {statusOptions.map((opt) => (
            <button
              key={opt.value}
              className={cn(
                "px-2 py-1 text-xs rounded transition-colors",
                statusFilter === opt.value
                  ? "bg-muted text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
              )}
              onClick={() => setStatusFilter(opt.value as typeof statusFilter)}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Resource overview */}
      <div className="px-3 py-2 border-t border-border space-y-2">
        <div className="text-xs font-medium text-muted-foreground">Overview</div>
        <div className="grid grid-cols-2 gap-2">
          <div className="flex items-center gap-2 text-xs">
            <Server className="w-3.5 h-3.5 text-green-500" />
            <span>{onlineCount} online</span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <Activity className="w-3.5 h-3.5 text-blue-500" />
            <span>{totalPods}/{totalCapacity} pods</span>
          </div>
        </div>
        {/* Capacity bar */}
        {totalCapacity > 0 && (
          <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className="h-full bg-primary transition-all"
              style={{ width: `${Math.min(100, (totalPods / totalCapacity) * 100)}%` }}
            />
          </div>
        )}
      </div>

      {/* Runner list */}
      <div className="flex-1 overflow-y-auto border-t border-border">
        <div className="px-3 py-2 text-xs text-muted-foreground border-b border-border">
          {filteredRunners.length} runners
        </div>

        {loading && runners.length === 0 ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : filteredRunners.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Server className="w-8 h-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              {searchQuery || statusFilter !== "all"
                ? "No runners match filters"
                : "No runners yet"}
            </p>
            {!searchQuery && statusFilter === "all" && (
              <Button
                size="sm"
                variant="outline"
                className="mt-3"
                onClick={handleAddRunner}
              >
                Add Runner
              </Button>
            )}
          </div>
        ) : (
          <div className="py-1">
            {filteredRunners.map((runner) => {
              const isSelected = selectedRunnerId === runner.id;
              const statusInfo = getRunnerStatusInfo(runner.status);
              const hostInfo = formatHostInfo(runner.host_info);

              return (
                <div
                  key={runner.id}
                  className={cn(
                    "group flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer",
                    isSelected && "bg-muted/30"
                  )}
                  onClick={() => handleRunnerClick(runner)}
                >
                  {/* Status dot */}
                  <span
                    className={cn(
                      "w-2 h-2 rounded-full flex-shrink-0",
                      statusInfo.dotColor
                    )}
                  />

                  {/* Runner info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5">
                      <p className="text-sm truncate font-medium">
                        {runner.node_id}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{runner.current_pods}/{runner.max_concurrent_pods} pods</span>
                      {runner.host_info?.os && (
                        <>
                          <span>·</span>
                          <span>{runner.host_info.os}</span>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Enabled indicator */}
                  {!runner.is_enabled && (
                    <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                      Disabled
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Registration tokens section */}
      <Collapsible open={tokensExpanded} onOpenChange={setTokensExpanded}>
        <CollapsibleTrigger asChild>
          <div className="flex items-center justify-between px-3 py-2 border-t border-border cursor-pointer hover:bg-muted/50">
            <div className="flex items-center gap-2">
              <Key className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">Registration Tokens</span>
              <span className="text-xs text-muted-foreground">
                ({activeTokens} active)
              </span>
            </div>
            {tokensExpanded ? (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border-t border-border px-3 py-2">
            {tokens.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-2">
                No registration tokens
              </p>
            ) : (
              <div className="space-y-1.5">
                {tokens.slice(0, 3).map((token) => (
                  <div
                    key={token.id}
                    className="flex items-center justify-between text-xs"
                  >
                    <span className={cn(
                      "truncate",
                      !token.is_active && "text-muted-foreground line-through"
                    )}>
                      {token.description || `Token #${token.id}`}
                    </span>
                    <span className="text-muted-foreground flex-shrink-0 ml-2">
                      {token.used_count}/{token.max_uses || "∞"}
                    </span>
                  </div>
                ))}
                {tokens.length > 3 && (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="w-full h-7 text-xs"
                    onClick={() => router.push(`/${currentOrg?.slug}/settings?tab=runners`)}
                  >
                    View all tokens →
                  </Button>
                )}
              </div>
            )}
            <Button
              size="sm"
              variant="outline"
              className="w-full mt-2 h-7 text-xs"
              onClick={() => router.push(`/${currentOrg?.slug}/settings?tab=runners`)}
            >
              Manage Tokens
            </Button>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}

export default RunnersSidebarContent;
