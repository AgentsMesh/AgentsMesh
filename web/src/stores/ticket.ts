import { create } from "zustand";
import { ticketApi, TicketData, TicketType, TicketStatus, TicketPriority } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/utils";

// Re-export types from API for backward compatibility
export type { TicketType, TicketStatus, TicketPriority };

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
  type?: TicketType;
  assigneeId?: number;
  repositoryId?: number;
  search?: string;
}

export type TicketViewMode = "list" | "board";

interface TicketState {
  // State
  tickets: Ticket[];
  currentTicket: Ticket | null;
  selectedTicketIdentifier: string | null; // For panel selection (without full fetch)
  labels: Label[];
  filters: TicketFilters;
  viewMode: TicketViewMode;
  loading: boolean;
  error: string | null;
  totalCount: number;

  // Actions
  fetchTickets: (filters?: TicketFilters) => Promise<void>;
  fetchTicket: (identifier: string) => Promise<void>;
  setSelectedTicketIdentifier: (identifier: string | null) => void;
  createTicket: (data: {
    repositoryId: number;
    type: TicketType;
    title: string;
    description?: string;
    content?: string;
    priority?: TicketPriority;
    assigneeIds?: number[];
    labels?: string[];
    parentId?: number;
  }) => Promise<Ticket>;
  updateTicket: (
    identifier: string,
    data: Partial<{
      title: string;
      description: string;
      content: string;
      type: TicketType;
      status: TicketStatus;
      priority: TicketPriority;
      assigneeIds: number[];
      labels: string[];
    }>
  ) => Promise<Ticket>;
  deleteTicket: (identifier: string) => Promise<void>;
  updateTicketStatus: (identifier: string, status: TicketStatus) => Promise<void>;
  fetchLabels: (repositoryId?: number) => Promise<void>;
  createLabel: (name: string, color: string, repositoryId?: number) => Promise<Label>;
  deleteLabel: (id: number) => Promise<void>;
  setFilters: (filters: TicketFilters) => void;
  setViewMode: (mode: TicketViewMode) => void;
  setCurrentTicket: (ticket: Ticket | null) => void;
  clearError: () => void;
}

export const useTicketStore = create<TicketState>((set, get) => ({
  tickets: [],
  currentTicket: null,
  selectedTicketIdentifier: null,
  labels: [],
  filters: {},
  viewMode: "board",
  loading: false,
  error: null,
  totalCount: 0,

  fetchTickets: async (filters) => {
    const mergedFilters = { ...get().filters, ...filters };
    set({ loading: true, error: null, filters: mergedFilters });
    try {
      const response = await ticketApi.list(mergedFilters);
      set({
        tickets: response.tickets || [],
        totalCount: response.total || 0,
        loading: false,
      });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch tickets"),
        loading: false,
      });
    }
  },

  fetchTicket: async (identifier) => {
    set({ loading: true, error: null });
    try {
      const ticket = await ticketApi.get(identifier);
      set({ currentTicket: ticket, loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch ticket"),
        loading: false,
      });
    }
  },

  createTicket: async (data) => {
    set({ loading: true, error: null });
    try {
      const ticket = await ticketApi.create(data);
      set((state) => ({
        tickets: [ticket, ...state.tickets],
        totalCount: state.totalCount + 1,
        loading: false,
      }));
      return ticket;
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to create ticket"),
        loading: false,
      });
      throw error;
    }
  },

  updateTicket: async (identifier, data) => {
    try {
      const ticket = await ticketApi.update(identifier, data);
      set((state) => ({
        tickets: state.tickets.map((t) =>
          t.identifier === identifier ? ticket : t
        ),
        currentTicket:
          state.currentTicket?.identifier === identifier
            ? ticket
            : state.currentTicket,
      }));
      return ticket;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update ticket") });
      throw error;
    }
  },

  deleteTicket: async (identifier) => {
    try {
      await ticketApi.delete(identifier);
      set((state) => ({
        tickets: state.tickets.filter((t) => t.identifier !== identifier),
        totalCount: state.totalCount - 1,
        currentTicket:
          state.currentTicket?.identifier === identifier
            ? null
            : state.currentTicket,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to delete ticket") });
      throw error;
    }
  },

  updateTicketStatus: async (identifier, status) => {
    try {
      await ticketApi.updateStatus(identifier, status);
      set((state) => ({
        tickets: state.tickets.map((t) =>
          t.identifier === identifier ? { ...t, status } : t
        ),
        currentTicket:
          state.currentTicket?.identifier === identifier
            ? { ...state.currentTicket, status }
            : state.currentTicket,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update ticket status") });
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

  setViewMode: (mode) => {
    set({ viewMode: mode });
  },

  setCurrentTicket: (ticket) => {
    set({ currentTicket: ticket });
  },

  setSelectedTicketIdentifier: (identifier) => {
    set({ selectedTicketIdentifier: identifier });
  },

  clearError: () => {
    set({ error: null });
  },
}));

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
    cancelled: { label: "Cancelled", color: "text-red-600 dark:text-red-400", bgColor: "bg-red-100 dark:bg-red-900/30" },
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

// Helper function to get type display info
export const getTypeInfo = (type: TicketType) => {
  const typeMap: Record<TicketType, { label: string; color: string; icon: string }> = {
    task: { label: "Task", color: "text-blue-500 dark:text-blue-400", icon: "✓" },
    bug: { label: "Bug", color: "text-red-500 dark:text-red-400", icon: "🐛" },
    feature: { label: "Feature", color: "text-green-500 dark:text-green-400", icon: "✨" },
    improvement: { label: "Improvement", color: "text-cyan-500 dark:text-cyan-400", icon: "📈" },
    epic: { label: "Epic", color: "text-purple-500 dark:text-purple-400", icon: "⚡" },
    subtask: { label: "Subtask", color: "text-gray-500 dark:text-gray-400", icon: "◦" },
    story: { label: "Story", color: "text-teal-500 dark:text-teal-400", icon: "📖" },
  };
  // Return default if type not found
  return typeMap[type] || { label: type || "Unknown", color: "text-gray-500 dark:text-gray-400", icon: "?" };
};
