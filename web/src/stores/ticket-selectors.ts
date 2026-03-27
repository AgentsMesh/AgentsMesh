import { useMemo } from "react";
import { useTicketStore } from "./ticket";
import type { Ticket } from "./ticket";
import type { TicketStatus, TicketPriority } from "@/lib/api";

/**
 * Selector hook: returns tickets filtered by current UI filters (search, status, priority).
 * Uses Zustand selectors + useMemo for efficient re-renders.
 */
export function useFilteredTickets(): Ticket[] {
  const tickets = useTicketStore((s) => s.tickets);
  const search = useTicketStore((s) => s.filters.search);
  const selectedStatuses = useTicketStore((s) => s.uiFilters.selectedStatuses);
  const selectedPriorities = useTicketStore((s) => s.uiFilters.selectedPriorities);

  return useMemo(() => {
    return tickets.filter((ticket) => {
      if (search) {
        const q = search.toLowerCase();
        if (!ticket.title.toLowerCase().includes(q) && !ticket.slug.toLowerCase().includes(q)) return false;
      }
      if (selectedStatuses.length > 0 && !selectedStatuses.includes(ticket.status)) return false;
      if (selectedPriorities.length > 0 && !selectedPriorities.includes(ticket.priority)) return false;
      return true;
    });
  }, [tickets, search, selectedStatuses, selectedPriorities]);
}

export const getStatusInfo = (status: TicketStatus) => {
  const statusMap: Record<TicketStatus, { label: string; color: string; bgColor: string }> = {
    backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" },
    todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bgColor: "bg-blue-100 dark:bg-blue-900/30" },
    in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bgColor: "bg-yellow-100 dark:bg-yellow-900/30" },
    in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bgColor: "bg-purple-100 dark:bg-purple-900/30" },
    done: { label: "Done", color: "text-green-600 dark:text-green-400", bgColor: "bg-green-100 dark:bg-green-900/30" },
  };
  return statusMap[status] || { label: status || "Unknown", color: "text-gray-500 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" };
};

export const getPriorityInfo = (priority: TicketPriority) => {
  const priorityMap: Record<TicketPriority, { label: string; color: string; icon: string }> = {
    none: { label: "None", color: "text-gray-400 dark:text-gray-500", icon: "\u2014" },
    low: { label: "Low", color: "text-green-500 dark:text-green-400", icon: "\u2193" },
    medium: { label: "Medium", color: "text-yellow-500 dark:text-yellow-400", icon: "\u2192" },
    high: { label: "High", color: "text-orange-500 dark:text-orange-400", icon: "\u2191" },
    urgent: { label: "Urgent", color: "text-red-500 dark:text-red-400", icon: "\u26A1" },
  };
  return priorityMap[priority] || { label: priority || "Unknown", color: "text-gray-400 dark:text-gray-500", icon: "?" };
};
