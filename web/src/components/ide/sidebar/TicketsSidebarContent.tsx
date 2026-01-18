"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, TicketStatus, TicketType, TicketPriority } from "@/stores/ticket";
import { StatusIcon, TypeIcon, PriorityIcon, getStatusDisplayInfo, getTypeDisplayInfo, getPriorityDisplayInfo } from "@/components/tickets";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { TicketCreateDialog } from "@/components/tickets";
import {
  Circle,
  CheckCircle2,
  Clock,
  AlertCircle,
  Loader2,
  Plus,
  Search,
  LayoutList,
  LayoutGrid,
  ChevronDown,
  ChevronRight,
  XCircle,
  RefreshCw,
} from "lucide-react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useTranslations } from "@/lib/i18n/client";

interface TicketsSidebarContentProps {
  className?: string;
}

// Status icons
const statusIcons: Record<TicketStatus, React.ReactNode> = {
  backlog: <Circle className="w-3 h-3 text-gray-400 dark:text-gray-500" />,
  todo: <Circle className="w-3 h-3 text-blue-500 dark:text-blue-400" />,
  in_progress: <Clock className="w-3 h-3 text-yellow-500 dark:text-yellow-400" />,
  in_review: <AlertCircle className="w-3 h-3 text-purple-500 dark:text-purple-400" />,
  done: <CheckCircle2 className="w-3 h-3 text-green-500 dark:text-green-400" />,
  cancelled: <XCircle className="w-3 h-3 text-gray-400 dark:text-gray-500" />,
};

// Filter options
const statusOptions: TicketStatus[] = ["backlog", "todo", "in_progress", "in_review", "done", "cancelled"];
const typeOptions: TicketType[] = ["task", "bug", "feature", "improvement", "epic"];
const priorityOptions: TicketPriority[] = ["urgent", "high", "medium", "low", "none"];

export function TicketsSidebarContent({ className }: TicketsSidebarContentProps) {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const {
    tickets,
    loading,
    filters,
    viewMode,
    fetchTickets,
    setFilters,
    setViewMode,
  } = useTicketStore();

  // Local state for search (debounced)
  const [searchQuery, setSearchQuery] = useState(filters.search || "");
  const [refreshing, setRefreshing] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  // Collapsible sections
  const [statusExpanded, setStatusExpanded] = useState(true);
  const [typeExpanded, setTypeExpanded] = useState(false);
  const [priorityExpanded, setPriorityExpanded] = useState(false);

  // Multi-select filter state
  const [selectedStatuses, setSelectedStatuses] = useState<TicketStatus[]>([]);
  const [selectedTypes, setSelectedTypes] = useState<TicketType[]>([]);
  const [selectedPriorities, setSelectedPriorities] = useState<TicketPriority[]>([]);

  // Load tickets on mount
  useEffect(() => {
    if (currentOrg) {
      fetchTickets();
    }
  }, [currentOrg, fetchTickets]);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters({ ...filters, search: searchQuery || undefined });
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchTickets();
    } finally {
      setRefreshing(false);
    }
  }, [fetchTickets]);

  // Filter tickets locally based on multi-select
  const filteredTickets = tickets.filter((ticket) => {
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesTitle = ticket.title.toLowerCase().includes(query);
      const matchesId = ticket.identifier.toLowerCase().includes(query);
      if (!matchesTitle && !matchesId) return false;
    }

    // Status filter (multi-select)
    if (selectedStatuses.length > 0 && !selectedStatuses.includes(ticket.status)) {
      return false;
    }

    // Type filter (multi-select)
    if (selectedTypes.length > 0 && !selectedTypes.includes(ticket.type)) {
      return false;
    }

    // Priority filter (multi-select)
    if (selectedPriorities.length > 0 && !selectedPriorities.includes(ticket.priority)) {
      return false;
    }

    return true;
  });

  const handleTicketClick = (identifier: string) => {
    router.push(`/${currentOrg?.slug}/tickets/${identifier}`);
  };

  const toggleStatus = (status: TicketStatus) => {
    setSelectedStatuses(prev =>
      prev.includes(status)
        ? prev.filter(s => s !== status)
        : [...prev, status]
    );
  };

  const toggleType = (type: TicketType) => {
    setSelectedTypes(prev =>
      prev.includes(type)
        ? prev.filter(t => t !== type)
        : [...prev, type]
    );
  };

  const togglePriority = (priority: TicketPriority) => {
    setSelectedPriorities(prev =>
      prev.includes(priority)
        ? prev.filter(p => p !== priority)
        : [...prev, priority]
    );
  };

  const clearAllFilters = () => {
    setSearchQuery("");
    setSelectedStatuses([]);
    setSelectedTypes([]);
    setSelectedPriorities([]);
    setFilters({});
  };

  const hasActiveFilters = searchQuery || selectedStatuses.length > 0 || selectedTypes.length > 0 || selectedPriorities.length > 0;

  // Handle ticket created
  const handleTicketCreated = useCallback((ticketId: number, identifier: string) => {
    // Refresh tickets to show the new one
    fetchTickets();
    // Navigate to the new ticket
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
      <div className="border-t border-border">
        {/* Status Filter */}
        <Collapsible open={statusExpanded} onOpenChange={setStatusExpanded}>
          <CollapsibleTrigger asChild>
            <div className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/50">
              <span className="text-xs font-medium">{t("tickets.filters.status")}</span>
              <div className="flex items-center gap-1">
                {selectedStatuses.length > 0 && (
                  <span className="text-xs bg-primary/10 text-primary px-1.5 rounded">
                    {selectedStatuses.length}
                  </span>
                )}
                {statusExpanded ? (
                  <ChevronDown className="w-3.5 h-3.5 text-muted-foreground" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
                )}
              </div>
            </div>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="px-3 pb-2 space-y-1">
              {statusOptions.map((status) => {
                const info = getStatusDisplayInfo(status);
                return (
                  <label
                    key={status}
                    className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
                  >
                    <Checkbox
                      checked={selectedStatuses.includes(status)}
                      onCheckedChange={() => toggleStatus(status)}
                      className="h-3.5 w-3.5"
                    />
                    <StatusIcon status={status} size="xs" />
                    <span>{info.label}</span>
                  </label>
                );
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>

        {/* Type Filter */}
        <Collapsible open={typeExpanded} onOpenChange={setTypeExpanded}>
          <CollapsibleTrigger asChild>
            <div className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/50 border-t border-border">
              <span className="text-xs font-medium">{t("tickets.filters.type")}</span>
              <div className="flex items-center gap-1">
                {selectedTypes.length > 0 && (
                  <span className="text-xs bg-primary/10 text-primary px-1.5 rounded">
                    {selectedTypes.length}
                  </span>
                )}
                {typeExpanded ? (
                  <ChevronDown className="w-3.5 h-3.5 text-muted-foreground" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
                )}
              </div>
            </div>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="px-3 pb-2 space-y-1">
              {typeOptions.map((type) => {
                const info = getTypeDisplayInfo(type);
                return (
                  <label
                    key={type}
                    className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
                  >
                    <Checkbox
                      checked={selectedTypes.includes(type)}
                      onCheckedChange={() => toggleType(type)}
                      className="h-3.5 w-3.5"
                    />
                    <TypeIcon type={type} size="xs" />
                    <span>{info.label}</span>
                  </label>
                );
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>

        {/* Priority Filter */}
        <Collapsible open={priorityExpanded} onOpenChange={setPriorityExpanded}>
          <CollapsibleTrigger asChild>
            <div className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/50 border-t border-border">
              <span className="text-xs font-medium">{t("tickets.filters.priority")}</span>
              <div className="flex items-center gap-1">
                {selectedPriorities.length > 0 && (
                  <span className="text-xs bg-primary/10 text-primary px-1.5 rounded">
                    {selectedPriorities.length}
                  </span>
                )}
                {priorityExpanded ? (
                  <ChevronDown className="w-3.5 h-3.5 text-muted-foreground" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
                )}
              </div>
            </div>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="px-3 pb-2 space-y-1">
              {priorityOptions.map((priority) => {
                const info = getPriorityDisplayInfo(priority);
                return (
                  <label
                    key={priority}
                    className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
                  >
                    <Checkbox
                      checked={selectedPriorities.includes(priority)}
                      onCheckedChange={() => togglePriority(priority)}
                      className="h-3.5 w-3.5"
                    />
                    <PriorityIcon priority={priority} size="xs" />
                    <span>{info.label}</span>
                  </label>
                );
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>
      </div>

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
              <div
                key={ticket.id}
                className="group flex items-start gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer"
                onClick={() => handleTicketClick(ticket.identifier)}
              >
                {/* Status icon */}
                <div className="mt-0.5">
                  {statusIcons[ticket.status]}
                </div>

                {/* Ticket info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground font-mono">
                      {ticket.identifier}
                    </span>
                    {ticket.priority === "urgent" && (
                      <span className="text-red-500 dark:text-red-400 text-xs">!</span>
                    )}
                    {ticket.priority === "high" && (
                      <span className="text-orange-500 dark:text-orange-400 text-xs">!!</span>
                    )}
                  </div>
                  <p className="text-sm truncate">{ticket.title}</p>
                </div>
              </div>
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
