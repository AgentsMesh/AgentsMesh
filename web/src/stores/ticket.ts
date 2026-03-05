import { useMemo } from "react";
import { create } from "zustand";
import { ticketApi, TicketData, TicketStatus, TicketPriority } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";

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

interface TicketFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  assigneeId?: number;
  repositoryId?: number;
  search?: string;
}

// Local UI filter selections (multi-select checkboxes in sidebar)
interface TicketUIFilters {
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
}

export type TicketViewMode = "list" | "board";

interface TicketState {
  // State
  tickets: Ticket[];
  currentTicket: Ticket | null;
  selectedTicketSlug: string | null; // For panel selection (without full fetch)
  labels: Label[];
  filters: TicketFilters;
  uiFilters: TicketUIFilters;
  viewMode: TicketViewMode;
  loading: boolean;
  error: string | null;
  totalCount: number;

  // Actions
  fetchTickets: (filters?: TicketFilters) => Promise<void>;
  fetchTicket: (slug: string) => Promise<void>;
  setSelectedTicketSlug: (slug: string | null) => void;
  createTicket: (data: {
    repositoryId: number;
    title: string;
    content?: string;
    priority?: TicketPriority;
    assigneeIds?: number[];
    labels?: string[];
    parentId?: number;
  }) => Promise<Ticket>;
  updateTicket: (
    slug: string,
    data: Partial<{
      title: string;
      content: string;
      status: TicketStatus;
      priority: TicketPriority;
      repositoryId: number | null;
      assigneeIds: number[];
      labels: string[];
    }>
  ) => Promise<Ticket>;
  deleteTicket: (slug: string) => Promise<void>;
  updateTicketStatus: (slug: string, status: TicketStatus) => Promise<void>;
  fetchLabels: (repositoryId?: number) => Promise<void>;
  createLabel: (name: string, color: string, repositoryId?: number) => Promise<Label>;
  deleteLabel: (id: number) => Promise<void>;
  setFilters: (filters: TicketFilters) => void;
  setUIFilters: (uiFilters: Partial<TicketUIFilters>) => void;
  toggleStatus: (status: TicketStatus) => void;
  togglePriority: (priority: TicketPriority) => void;
  clearUIFilters: () => void;
  setViewMode: (mode: TicketViewMode) => void;
  setCurrentTicket: (ticket: Ticket | null) => void;
  clearError: () => void;
}

export const useTicketStore = create<TicketState>((set, get) => ({
  tickets: [],
  currentTicket: null,
  selectedTicketSlug: null,
  labels: [],
  filters: {},
  uiFilters: { selectedStatuses: [], selectedPriorities: [] },
  viewMode: "board",
  loading: false,
  error: null,
  totalCount: 0,

  fetchTickets: async (filters) => {
    const mergedFilters = { ...get().filters, ...filters };
    set({ error: null, filters: mergedFilters });
    try {
      const response = await ticketApi.list(mergedFilters);
      set({
        tickets: response.tickets || [],
        totalCount: response.total || 0,
      });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch tickets"),
      });
    }
  },

  fetchTicket: async (slug) => {
    try {
      const ticket = await ticketApi.get(slug);
      set({ currentTicket: ticket });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch ticket"),
      });
    }
  },

  createTicket: async (data) => {
    set({ error: null });
    try {
      const ticket = await ticketApi.create(data);
      set((state) => ({
        tickets: [ticket, ...state.tickets],
        totalCount: state.totalCount + 1,
      }));
      return ticket;
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to create ticket"),
      });
      throw error;
    }
  },

  updateTicket: async (slug, data) => {
    try {
      const ticket = await ticketApi.update(slug, data);
      set((state) => ({
        tickets: state.tickets.map((t) =>
          t.slug === slug ? ticket : t
        ),
        currentTicket:
          state.currentTicket?.slug === slug
            ? ticket
            : state.currentTicket,
      }));
      return ticket;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update ticket") });
      throw error;
    }
  },

  deleteTicket: async (slug) => {
    try {
      await ticketApi.delete(slug);
      set((state) => ({
        tickets: state.tickets.filter((t) => t.slug !== slug),
        totalCount: state.totalCount - 1,
        currentTicket:
          state.currentTicket?.slug === slug
            ? null
            : state.currentTicket,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to delete ticket") });
      throw error;
    }
  },

  updateTicketStatus: async (slug, status) => {
    const prevTickets = get().tickets;
    const prevCurrent = get().currentTicket;

    // Optimistic update
    set((state) => ({
      tickets: state.tickets.map((t) =>
        t.slug === slug ? { ...t, status } : t
      ),
      currentTicket:
        state.currentTicket?.slug === slug
          ? { ...state.currentTicket, status }
          : state.currentTicket,
    }));

    try {
      await ticketApi.updateStatus(slug, status);
    } catch (error: unknown) {
      // Rollback on failure
      set({ tickets: prevTickets, currentTicket: prevCurrent, error: getErrorMessage(error, "Failed to update ticket status") });
      throw error;
    }
  },

  fetchLabels: async (repositoryId) => {
    try {
      const response = await ticketApi.listLabels(repositoryId);
      set({ labels: response.labels || [] });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch labels") });
    }
  },

  createLabel: async (name, color, repositoryId) => {
    try {
      const label = await ticketApi.createLabel(name, color, repositoryId);
      set((state) => ({
        labels: [...state.labels, label],
      }));
      return label;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to create label") });
      throw error;
    }
  },

  deleteLabel: async (id) => {
    try {
      await ticketApi.deleteLabel(id);
      set((state) => ({
        labels: state.labels.filter((l) => l.id !== id),
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to delete label") });
      throw error;
    }
  },

  setFilters: (filters) => {
    set({ filters });
  },

  setUIFilters: (partial) => {
    set((state) => ({ uiFilters: { ...state.uiFilters, ...partial } }));
  },

  toggleStatus: (status) => {
    set((state) => {
      const prev = state.uiFilters.selectedStatuses;
      return {
        uiFilters: {
          ...state.uiFilters,
          selectedStatuses: prev.includes(status)
            ? prev.filter((s) => s !== status)
            : [...prev, status],
        },
      };
    });
  },

  togglePriority: (priority) => {
    set((state) => {
      const prev = state.uiFilters.selectedPriorities;
      return {
        uiFilters: {
          ...state.uiFilters,
          selectedPriorities: prev.includes(priority)
            ? prev.filter((p) => p !== priority)
            : [...prev, priority],
        },
      };
    });
  },

  clearUIFilters: () => {
    set({
      uiFilters: { selectedStatuses: [], selectedPriorities: [] },
    });
  },

  setViewMode: (mode) => {
    set({ viewMode: mode });
  },

  setCurrentTicket: (ticket) => {
    set({ currentTicket: ticket });
  },

  setSelectedTicketSlug: (slug) => {
    set({ selectedTicketSlug: slug });
  },

  clearError: () => {
    set({ error: null });
  },
}));

/**
 * Selector hook: returns tickets filtered by current UI filters (search, status, priority).
 * Uses Zustand selectors + useMemo for efficient re-renders.
 * Single source of truth for filtered tickets — used by both sidebar and main content.
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
        if (!ticket.title.toLowerCase().includes(q) && !ticket.slug.toLowerCase().includes(q)) {
          return false;
        }
      }
      if (selectedStatuses.length > 0 && !selectedStatuses.includes(ticket.status)) return false;
      if (selectedPriorities.length > 0 && !selectedPriorities.includes(ticket.priority)) return false;
      return true;
    });
  }, [tickets, search, selectedStatuses, selectedPriorities]);
}

// Helper function to get status display info
export const getStatusInfo = (status: TicketStatus) => {
  const statusMap: Record<
    TicketStatus,
    { label: string; color: string; bgColor: string }
  > = {
    backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" },
    todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bgColor: "bg-blue-100 dark:bg-blue-900/30" },
    in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bgColor: "bg-yellow-100 dark:bg-yellow-900/30" },
    in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bgColor: "bg-purple-100 dark:bg-purple-900/30" },
    done: { label: "Done", color: "text-green-600 dark:text-green-400", bgColor: "bg-green-100 dark:bg-green-900/30" },
  };
  // Return default if status not found
  return statusMap[status] || { label: status || "Unknown", color: "text-gray-500 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" };
};

// Helper function to get priority display info
export const getPriorityInfo = (priority: TicketPriority) => {
  const priorityMap: Record<
    TicketPriority,
    { label: string; color: string; icon: string }
  > = {
    none: { label: "None", color: "text-gray-400 dark:text-gray-500", icon: "—" },
    low: { label: "Low", color: "text-green-500 dark:text-green-400", icon: "↓" },
    medium: { label: "Medium", color: "text-yellow-500 dark:text-yellow-400", icon: "→" },
    high: { label: "High", color: "text-orange-500 dark:text-orange-400", icon: "↑" },
    urgent: { label: "Urgent", color: "text-red-500 dark:text-red-400", icon: "⚡" },
  };
  // Return default if priority not found
  return priorityMap[priority] || { label: priority || "Unknown", color: "text-gray-400 dark:text-gray-500", icon: "?" };
};

