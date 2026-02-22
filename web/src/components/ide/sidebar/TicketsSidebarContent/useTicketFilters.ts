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
    toggleType,
    togglePriority,
    clearUIFilters,
  } = useTicketStore();

  const { selectedStatuses, selectedTypes, selectedPriorities } = uiFilters;

  // Local search input state (debounced before pushing to store)
  const [searchQuery, setSearchQuery] = useState(filters.search || "");

  // Debounce search → store
  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters({ ...filters, search: searchQuery || undefined });
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery]);

  // Single source of truth for filtered tickets (shared with main page)
  const filteredTickets = useFilteredTickets();

  const clearAllFilters = useCallback(() => {
    setSearchQuery("");
    clearUIFilters();
    setFilters({});
  }, [clearUIFilters, setFilters]);

  const hasActiveFilters = searchQuery.length > 0 ||
    selectedStatuses.length > 0 ||
    selectedTypes.length > 0 ||
    selectedPriorities.length > 0;

  return {
    // State
    searchQuery,
    selectedStatuses,
    selectedTypes,
    selectedPriorities,
    filteredTickets,

    // Actions
    setSearchQuery,
    toggleStatus,
    toggleType,
    togglePriority,
    clearAllFilters,
    hasActiveFilters,
  };
}
