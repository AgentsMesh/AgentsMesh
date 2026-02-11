"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore } from "@/stores/ticket";
import { TicketCreateDialog } from "@/components/tickets";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Loader2,
  Plus,
  Search,
  LayoutList,
  LayoutGrid,
  RefreshCw,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { useTicketFilters } from "./useTicketFilters";
import { TicketFilterSection } from "./TicketFilterSection";
import { TicketListItem } from "./TicketListItem";
import type { TicketsSidebarContentProps } from "./types";

/**
 * TicketsSidebarContent - Sidebar content for browsing and filtering tickets
 */
export function TicketsSidebarContent({ className }: TicketsSidebarContentProps) {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const { loading, viewMode, fetchTickets, setViewMode } = useTicketStore();

  // Filter state and actions
  const {
    searchQuery,
    selectedStatuses,
    selectedTypes,
    selectedPriorities,
    filteredTickets,
    setSearchQuery,
    toggleStatus,
    toggleType,
    togglePriority,
    clearAllFilters,
    hasActiveFilters,
  } = useTicketFilters();

  // Local UI state
  const [refreshing, setRefreshing] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [statusExpanded, setStatusExpanded] = useState(true);
  const [typeExpanded, setTypeExpanded] = useState(false);
  const [priorityExpanded, setPriorityExpanded] = useState(false);

  // Load tickets on mount
  useEffect(() => {
    if (currentOrg) {
      fetchTickets();
    }
  }, [currentOrg, fetchTickets]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchTickets();
    } finally {
      setRefreshing(false);
    }
  }, [fetchTickets]);

  const handleTicketClick = useCallback((identifier: string) => {
    router.push(`/${currentOrg?.slug}/tickets/${identifier}`);
  }, [router, currentOrg]);

  // Handle ticket created
  const handleTicketCreated = useCallback((ticketId: number, identifier: string) => {
    fetchTickets();
    if (currentOrg?.slug) {
      router.push(`/${currentOrg.slug}/tickets/${identifier}`);
    }
  }, [fetchTickets, currentOrg, router]);

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Create Ticket Dialog */}
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
        <Button
          size="sm"
          variant="outline"
          className="flex-1 h-8 text-xs"
          onClick={() => setCreateDialogOpen(true)}
        >
          <Plus className="w-3 h-3 mr-1" />
          {t("tickets.newTicket")}
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

      {/* View Mode Toggle */}
      <div className="flex items-center gap-1 px-2 pb-2">
        <span className="text-xs text-muted-foreground mr-2">{t("tickets.view")}:</span>
        <div className="flex border border-border rounded-md overflow-hidden">
          <button
            className={cn(
              "p-1.5 transition-colors",
              viewMode === "list" ? "bg-muted" : "hover:bg-muted/50"
            )}
            onClick={() => setViewMode("list")}
            title="List view"
          >
            <LayoutList className="h-3.5 w-3.5" />
          </button>
          <button
            className={cn(
              "p-1.5 transition-colors",
              viewMode === "board" ? "bg-muted" : "hover:bg-muted/50"
            )}
            onClick={() => setViewMode("board")}
            title="Board view"
          >
            <LayoutGrid className="h-3.5 w-3.5" />
          </button>
        </div>
        {hasActiveFilters && (
          <Button
            size="sm"
            variant="ghost"
            className="h-7 text-xs ml-auto"
            onClick={clearAllFilters}
          >
            {t("tickets.clear")}
          </Button>
        )}
      </div>

      {/* Filters */}
      <TicketFilterSection
        statusExpanded={statusExpanded}
        typeExpanded={typeExpanded}
        priorityExpanded={priorityExpanded}
        onStatusExpandedChange={setStatusExpanded}
        onTypeExpandedChange={setTypeExpanded}
        onPriorityExpandedChange={setPriorityExpanded}
        selectedStatuses={selectedStatuses}
        selectedTypes={selectedTypes}
        selectedPriorities={selectedPriorities}
        onToggleStatus={toggleStatus}
        onToggleType={toggleType}
        onTogglePriority={togglePriority}
        t={t}
      />

      {/* Ticket list preview */}
      <div className="flex-1 overflow-y-auto border-t border-border">
        <div className="px-3 py-2 text-xs text-muted-foreground border-b border-border">
          {filteredTickets.length} {t("tickets.ticketCount")}
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : filteredTickets.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <p className="text-sm text-muted-foreground">
              {hasActiveFilters ? t("tickets.emptyState.noMatch") : t("tickets.emptyState.title")}
            </p>
          </div>
        ) : (
          <div className="py-1">
            {filteredTickets.slice(0, 20).map((ticket) => (
              <TicketListItem
                key={ticket.id}
                ticket={ticket}
                onClick={handleTicketClick}
              />
            ))}
            {filteredTickets.length > 20 && (
              <div className="px-3 py-2 text-xs text-muted-foreground text-center">
                {t("tickets.moreTickets", { count: filteredTickets.length - 20 })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default TicketsSidebarContent;
