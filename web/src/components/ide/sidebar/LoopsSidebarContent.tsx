"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useLoopStore, LoopData } from "@/stores/loop";
import { LoopCreateDialog } from "@/components/loops/LoopCreateDialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Loader2,
  Plus,
  Search,
  RefreshCw,
  Clock,
  Bot,
  Zap,
  Repeat,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { formatTimeAgo } from "@/lib/utils/time";

export function LoopsSidebarContent({ className }: { className?: string }) {
  const t = useTranslations();
  const router = useRouter();
  const pathname = usePathname();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const loops = useLoopStore((s) => s.loops);
  const loading = useLoopStore((s) => s.loading);
  const fetchLoops = useLoopStore((s) => s.fetchLoops);

  const [searchQuery, setSearchQuery] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  useEffect(() => {
    if (currentOrg) {
      fetchLoops();
    }
  }, [currentOrg, fetchLoops]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchLoops();
    } finally {
      setRefreshing(false);
    }
  }, [fetchLoops]);

  const handleLoopClick = useCallback(
    (slug: string) => {
      router.push(`/${currentOrg?.slug}/loops/${slug}`);
    },
    [router, currentOrg]
  );

  const handleCreated = useCallback(() => {
    setCreateDialogOpen(false);
    fetchLoops();
  }, [fetchLoops]);

  const filteredLoops = searchQuery
    ? loops.filter(
        (l) =>
          l.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          l.slug.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : loops;

  // Determine which loop is currently active from URL
  const activeSlug = pathname?.match(/\/loops\/([^/]+)/)?.[1] || null;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <LoopCreateDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onCreated={handleCreated}
      />

      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
          <Input
            placeholder={t("loops.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm bg-muted/50 border-transparent focus:border-border focus:bg-background transition-colors"
          />
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-1 px-2 pb-2">
        <Button
          size="sm"
          variant="outline"
          className="flex-1 h-7 text-xs gap-1"
          onClick={() => setCreateDialogOpen(true)}
        >
          <Plus className="w-3 h-3" />
          {t("loops.createLoop")}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 w-7 p-0"
          onClick={handleRefresh}
          disabled={refreshing}
        >
          <RefreshCw className={cn("w-3.5 h-3.5", refreshing && "animate-spin")} />
        </Button>
      </div>

      {/* Loop list */}
      <div className="flex-1 overflow-y-auto border-t border-border">
        {/* Count header */}
        <div className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-muted-foreground font-medium">
          {t("loops.loopCount", { count: filteredLoops.length })}
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
          </div>
        ) : filteredLoops.length === 0 ? (
          <div className="flex flex-col items-center px-3 py-8 text-center">
            <Repeat className="w-8 h-8 text-muted-foreground/40 mb-2" />
            <p className="text-xs text-muted-foreground">
              {searchQuery ? t("loops.noMatch") : t("loops.emptyState")}
            </p>
          </div>
        ) : (
          <div className="px-1 pb-1">
            {filteredLoops.map((loop) => (
              <LoopListItem
                key={loop.id}
                loop={loop}
                onClick={handleLoopClick}
                isActive={activeSlug === loop.slug}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function LoopListItem({
  loop,
  onClick,
  isActive,
}: {
  loop: LoopData;
  onClick: (slug: string) => void;
  isActive: boolean;
}) {
  const t = useTranslations();
  const isEnabled = loop.status === "enabled";
  const isRunning = loop.active_run_count > 0;
  const successRate =
    loop.total_runs > 0
      ? Math.round((loop.successful_runs / loop.total_runs) * 100)
      : null;

  return (
    <button
      className={cn(
        "w-full text-left px-2.5 py-2 rounded-md",
        "transition-colors duration-150 cursor-pointer",
        isActive
          ? "bg-accent text-accent-foreground"
          : "hover:bg-muted/50 text-foreground"
      )}
      onClick={() => onClick(loop.slug)}
    >
      {/* Name row */}
      <div className="flex items-center gap-2 mb-1">
        <span className="relative flex-shrink-0">
          <span
            className={cn(
              "block w-2 h-2 rounded-full",
              isRunning ? "bg-blue-500" : isEnabled ? "bg-emerald-500" : "bg-gray-400 dark:bg-gray-600"
            )}
          />
          {isRunning && (
            <span className="absolute inset-0 w-2 h-2 rounded-full animate-ping opacity-30 bg-blue-500" />
          )}
        </span>
        <span className="text-sm font-medium truncate">{loop.name}</span>
      </div>

      {/* Trigger + Mode row */}
      <div className="flex items-center gap-1.5 ml-4 mb-1">
        {loop.cron_expression ? (
          <span className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground font-mono">
            <Clock className="w-2.5 h-2.5" />
            {loop.cron_expression}
          </span>
        ) : (
          <span className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground">
            <Repeat className="w-2.5 h-2.5" />
            {t("loops.onDemand")}
          </span>
        )}
        <span className="text-[10px] text-muted-foreground/60 mx-0.5">|</span>
        <span className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground">
          {loop.execution_mode === "autopilot" ? (
            <Bot className="w-2.5 h-2.5" />
          ) : (
            <Zap className="w-2.5 h-2.5" />
          )}
          {loop.execution_mode === "autopilot" ? t("loops.modeAutoShort") : t("loops.modeDirect")}
        </span>
      </div>

      {/* Last run + stats */}
      {loop.last_run_at && (
        <div className="flex items-center gap-1.5 ml-4 text-[10px] text-muted-foreground">
          <Clock className="w-2.5 h-2.5" />
          <span>{formatTimeAgo(loop.last_run_at, t)}</span>
          {successRate !== null && (
            <>
              <span className="text-muted-foreground/40">|</span>
              <span>{successRate}%</span>
            </>
          )}
        </div>
      )}
    </button>
  );
}

export default LoopsSidebarContent;
