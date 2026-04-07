"use client";

import { useState, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore } from "@/stores/ticket";
import { TicketCreateDialog } from "@/components/tickets";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Plus, Search, LayoutList, LayoutGrid, RefreshCw } from "lucide-react";
import { useTranslations } from "next-intl";
import { useTicketFilters } from "./useTicketFilters";
import { TicketFilterSection } from "./TicketFilterSection";
import { RepoFilterSection } from "./RepoFilterSection";
import type { TicketsSidebarContentProps } from "./types";

/**
 * TicketsSidebarContent - Sidebar content for browsing and filtering tickets
 */
export function TicketsSidebarContent({ className }: TicketsSidebarContentProps) {
  const t = useTranslations();
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const viewMode = useTicketStore((s) => s.viewMode);
  const allTickets = useTicketStore((s) => s.tickets);
  const fetchTickets = useTicketStore((s) => s.fetchTickets);
  const fetchBoard = useTicketStore((s) => s.fetchBoard);
  const setViewMode = useTicketStore((s) => s.setViewMode);
  const boardColumns = useTicketStore((s) => s.boardColumns);
  const storePriorityCounts = useTicketStore((s) => s.priorityCounts);

  const externalStatusCounts = useMemo(() => {
    if (boardColumns.length === 0) return undefined;
    const counts: Record<string, number> = {};
    for (const col of boardColumns) counts[col.status] = col.count;
    return counts;
  }, [boardColumns]);

  const externalPriorityCounts = useMemo(() => {
    return Object.keys(storePriorityCounts).length > 0 ? storePriorityCounts : undefined;
  }, [storePriorityCounts]);

  const {
    searchQuery, selectedStatuses, selectedPriorities, selectedRepositoryIds,
    setSearchQuery, toggleStatus, togglePriority, toggleRepository,
    clearAllFilters, hasActiveFilters,
  } = useTicketFilters();

  const [refreshing, setRefreshing] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [statusExpanded, setStatusExpanded] = useState(true);
  const [priorityExpanded, setPriorityExpanded] = useState(false);
  const [repoExpanded, setRepoExpanded] = useState(false);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try { await (viewMode === "board" ? fetchBoard() : fetchTickets()); }
    finally { setRefreshing(false); }
  }, [fetchTickets, fetchBoard, viewMode]);

  const handleTicketCreated = useCallback((_ticketId: number, slug: string) => {
    viewMode === "board" ? fetchBoard() : fetchTickets();
    if (currentOrg?.slug) router.push(`/${currentOrg.slug}/tickets/${slug}`);
  }, [fetchTickets, fetchBoard, viewMode, currentOrg, router]);

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <TicketCreateDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onCreated={handleTicketCreated}
      />

      {/* Search */}
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder={t("tickets.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-1 px-2 pb-2">
        <Button size="sm" variant="outline" className="flex-1 h-8 text-xs" onClick={() => setCreateDialogOpen(true)}>
          <Plus className="w-3 h-3 mr-1" />
          {t("tickets.newTicket")}
        </Button>
        <Button size="sm" variant="ghost" className="h-8 w-8 p-0" onClick={handleRefresh} disabled={refreshing}>
          <RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
        </Button>
      </div>

      {/* View Mode Toggle */}
      <div className="flex items-center gap-1 px-2 pb-2">
        <span className="text-xs text-muted-foreground mr-2">{t("tickets.view")}:</span>
        <div className="flex bg-muted rounded-full p-0.5">
          <button
            className={cn("flex items-center gap-1 px-2 py-1 rounded-full text-xs transition-all",
              viewMode === "list" ? "bg-background text-foreground shadow-sm font-medium" : "text-muted-foreground hover:text-foreground")}
            onClick={() => setViewMode("list")}
          >
            <LayoutList className="h-3 w-3" />
            {viewMode === "list" && <span>{t("tickets.list.ticket") || "List"}</span>}
          </button>
          <button
            className={cn("flex items-center gap-1 px-2 py-1 rounded-full text-xs transition-all",
              viewMode === "board" ? "bg-background text-foreground shadow-sm font-medium" : "text-muted-foreground hover:text-foreground")}
            onClick={() => setViewMode("board")}
          >
            <LayoutGrid className="h-3 w-3" />
            {viewMode === "board" && <span>{t("tickets.board")}</span>}
          </button>
        </div>
        {hasActiveFilters && (
          <Button size="sm" variant="ghost" className="h-7 text-xs ml-auto" onClick={clearAllFilters}>
            {t("tickets.clear")}
          </Button>
        )}
      </div>

      {/* Filters */}
      <div className="overflow-y-auto flex-1">
        <TicketFilterSection
          statusExpanded={statusExpanded}
          priorityExpanded={priorityExpanded}
          onStatusExpandedChange={setStatusExpanded}
          onPriorityExpandedChange={setPriorityExpanded}
          selectedStatuses={selectedStatuses}
          selectedPriorities={selectedPriorities}
          onToggleStatus={toggleStatus}
          onTogglePriority={togglePriority}
          allTickets={allTickets}
          externalStatusCounts={externalStatusCounts}
          externalPriorityCounts={externalPriorityCounts}
          t={t}
        />
        <RepoFilterSection
          expanded={repoExpanded}
          onExpandedChange={setRepoExpanded}
          selectedRepositoryIds={selectedRepositoryIds}
          onToggleRepository={toggleRepository}
          allTickets={allTickets}
          t={t}
        />
      </div>
    </div>
  );
}

export default TicketsSidebarContent;
