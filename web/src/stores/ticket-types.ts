import type { TicketData, TicketStatus, TicketPriority } from "@/lib/api";

// Re-export types from API for component convenience
export type { TicketStatus, TicketPriority };

export interface Label {
  id: number;
  name: string;
  color: string;
}

// Re-export TicketData as Ticket with optional child_tickets
export interface Ticket extends TicketData {
  child_tickets?: Ticket[];
}

export interface TicketFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  assigneeId?: number;
  repositoryId?: number;
  search?: string;
}

// Local UI filter selections (multi-select checkboxes in sidebar)
export interface TicketUIFilters {
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
}

export type TicketViewMode = "list" | "board";
