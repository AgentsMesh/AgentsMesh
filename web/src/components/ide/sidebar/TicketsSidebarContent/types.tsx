import React from "react";
import { Circle, CheckCircle2, Clock, AlertCircle } from "lucide-react";
import type { TicketStatus, TicketPriority } from "@/stores/ticket";

/**
 * Props for TicketsSidebarContent component
 */
export interface TicketsSidebarContentProps {
  className?: string;
}

/**
 * Status icons mapping
 */
export const statusIcons: Record<TicketStatus, React.ReactNode> = {
  backlog: <Circle className="w-3 h-3 text-gray-400 dark:text-gray-500" />,
  todo: <Circle className="w-3 h-3 text-blue-500 dark:text-blue-400" />,
  in_progress: <Clock className="w-3 h-3 text-yellow-500 dark:text-yellow-400" />,
  in_review: <AlertCircle className="w-3 h-3 text-purple-500 dark:text-purple-400" />,
  done: <CheckCircle2 className="w-3 h-3 text-green-500 dark:text-green-400" />,
};

/**
 * Filter options
 */
export const statusOptions: TicketStatus[] = ["backlog", "todo", "in_progress", "in_review", "done"];
export const priorityOptions: TicketPriority[] = ["urgent", "high", "medium", "low", "none"];

/**
 * Filter state interface
 */
export interface TicketFilterState {
  searchQuery: string;
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
  selectedRepositoryIds: number[];
}

/**
 * Filter actions interface
 */
export interface TicketFilterActions {
  setSearchQuery: (query: string) => void;
  toggleStatus: (status: TicketStatus) => void;
  togglePriority: (priority: TicketPriority) => void;
  toggleRepository: (id: number) => void;
  clearAllFilters: () => void;
  hasActiveFilters: boolean;
}
