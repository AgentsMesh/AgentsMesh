"use client";

import { useState, useEffect, useCallback } from "react";
import { useTicketStore, useFilteredTickets, type Ticket } from "@/stores/ticket";
import type { TicketFilterState, TicketFilterActions } from "./types";

/**
 * Custom hook for managing ticket filter UI in the sidebar.
 * Filter selections live in the Zustand store so that both the sidebar
 * and the main content area (board/list) share the same filtered data.
 */
export function useTicketFilters(): TicketFilterState & TicketFilterActions & {
  filteredTickets: Ticket[];
} {
  const {
    filters,
    uiFilters,
    setFilters,
    toggleStatus,
    togglePriority,
    toggleRepository,
    clearUIFilters,
  } = useTicketStore();

  const { selectedStatuses, selectedPriorities, selectedRepositoryIds } = uiFilters;

  const [searchQuery, setSearchQuery] = useState(filters.search || "");

  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters({ ...filters, search: searchQuery || undefined });
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery]);

  const filteredTickets = useFilteredTickets();

  const clearAllFilters = useCallback(() => {
    setSearchQuery("");
    clearUIFilters();
    setFilters({});
  }, [clearUIFilters, setFilters]);

  const hasActiveFilters = searchQuery.length > 0 ||
    selectedStatuses.length > 0 ||
    selectedPriorities.length > 0 ||
    selectedRepositoryIds.length > 0;

  return {
    searchQuery,
    selectedStatuses,
    selectedPriorities,
    selectedRepositoryIds,
    filteredTickets,

    setSearchQuery,
    toggleStatus,
    togglePriority,
    toggleRepository,
    clearAllFilters,
    hasActiveFilters,
  };
}
