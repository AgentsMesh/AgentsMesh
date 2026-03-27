import { create } from "zustand";
import { TicketData, TicketStatus, TicketPriority } from "@/lib/api";
import { createTicketActions } from "./ticket-actions";

// Re-export types from API for component convenience
export type { TicketStatus, TicketPriority };
// Re-export selectors and helpers
export { useFilteredTickets, getStatusInfo, getPriorityInfo } from "./ticket-selectors";

export interface Label {
  id: number;
  name: string;
  color: string;
}

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

interface TicketUIFilters {
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
}

export type TicketViewMode = "list" | "board";

interface TicketState {
  tickets: Ticket[];
  currentTicket: Ticket | null;
  selectedTicketSlug: string | null;
  labels: Label[];
  filters: TicketFilters;
  uiFilters: TicketUIFilters;
  viewMode: TicketViewMode;
  loading: boolean;
  error: string | null;
  totalCount: number;

  fetchTickets: (filters?: TicketFilters) => Promise<void>;
  fetchTicket: (slug: string) => Promise<void>;
  setSelectedTicketSlug: (slug: string | null) => void;
  createTicket: (data: {
    repositoryId: number; title: string; content?: string;
    priority?: TicketPriority; assigneeIds?: number[]; labels?: string[]; parentId?: number;
  }) => Promise<Ticket>;
  updateTicket: (slug: string, data: Partial<{
    title: string; content: string; status: TicketStatus; priority: TicketPriority;
    repositoryId: number | null; assigneeIds: number[]; labels: string[];
  }>) => Promise<Ticket>;
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

  // Spread API actions from extracted module
  ...createTicketActions(
    set as Parameters<typeof createTicketActions>[0],
    get as Parameters<typeof createTicketActions>[1]
  ),

  setFilters: (filters) => set({ filters }),
  setUIFilters: (partial) => set((state) => ({ uiFilters: { ...state.uiFilters, ...partial } })),

  toggleStatus: (status) => set((state) => {
    const prev = state.uiFilters.selectedStatuses;
    return { uiFilters: { ...state.uiFilters, selectedStatuses: prev.includes(status) ? prev.filter((s) => s !== status) : [...prev, status] } };
  }),

  togglePriority: (priority) => set((state) => {
    const prev = state.uiFilters.selectedPriorities;
    return { uiFilters: { ...state.uiFilters, selectedPriorities: prev.includes(priority) ? prev.filter((p) => p !== priority) : [...prev, priority] } };
  }),

  clearUIFilters: () => set({ uiFilters: { selectedStatuses: [], selectedPriorities: [] } }),
  setViewMode: (mode) => set({ viewMode: mode }),
  setCurrentTicket: (ticket) => set({ currentTicket: ticket }),
  setSelectedTicketSlug: (slug) => set({ selectedTicketSlug: slug }),
  clearError: () => set({ error: null }),
}));
