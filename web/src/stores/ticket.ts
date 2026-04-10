import { create } from "zustand";
import { TicketData, TicketStatus, TicketPriority, BoardColumn } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
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

export interface ColumnPagination {
  offset: number;
  hasMore: boolean;
  loading: boolean;
}

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

/** Shared dependency types for store action creators (get/set). */
export interface TicketStoreDeps {
  get: () => {
    tickets: Ticket[];
    currentTicket: Ticket | null;
    filters: TicketFilters;
    totalCount: number;
    boardColumns: BoardColumn[];
    priorityCounts: Record<string, number>;
    columnPagination: Record<string, ColumnPagination>;
  };
  set: (updater: object | ((state: ReturnType<TicketStoreDeps["get"]>) => object)) => void;
}

interface TicketUIFilters {
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
  selectedRepositoryIds: number[];
}

export type TicketViewMode = "list" | "board";

interface TicketState {
  /**
   * Flat ticket array consumed by useFilteredTickets and list view.
   * - List mode: source of truth, populated by fetchTickets.
   * - Board mode: **derived** from boardColumns via flattenColumns().
   *   All mutations (loadMore, drag-drop) update boardColumns first,
   *   then re-derive this array. Do NOT mutate directly in board mode.
   */
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
  /** Board mode source of truth: per-column tickets + DB total count. */
  boardColumns: BoardColumn[];
  priorityCounts: Record<string, number>;
  columnPagination: Record<string, ColumnPagination>;
  doneCollapsed: boolean;

  fetchTickets: (filters?: TicketFilters) => Promise<void>;
  fetchBoard: (filters?: TicketFilters) => Promise<void>;
  loadMoreColumn: (status: string) => Promise<void>;
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
  updateTicketStatusFromEvent: (slug: string, status: string, previousStatus?: string) => void;
  removeTicketFromEvent: (slug: string) => void;
  setFilters: (filters: TicketFilters) => void;
  setUIFilters: (uiFilters: Partial<TicketUIFilters>) => void;
  toggleStatus: (status: TicketStatus) => void;
  togglePriority: (priority: TicketPriority) => void;
  toggleRepository: (id: number) => void;
  clearUIFilters: () => void;
  setViewMode: (mode: TicketViewMode) => void;
  setCurrentTicket: (ticket: Ticket | null) => void;
  setDoneCollapsed: (collapsed: boolean) => void;
  clearError: () => void;
}

export const useTicketStore = create<TicketState>((set, get) => ({
  tickets: [],
  currentTicket: null,
  selectedTicketSlug: null,
  labels: [],
  filters: {},
  uiFilters: { selectedStatuses: [], selectedPriorities: [], selectedRepositoryIds: [] },
  viewMode: "board",
  loading: false,
  error: null,
  totalCount: 0,
  boardColumns: [],
  priorityCounts: {},
  columnPagination: {},
  doneCollapsed: true,

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

  toggleRepository: (id) => set((state) => {
    const prev = state.uiFilters.selectedRepositoryIds;
    return { uiFilters: { ...state.uiFilters, selectedRepositoryIds: prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id] } };
  }),

  clearUIFilters: () => set({ uiFilters: { selectedStatuses: [], selectedPriorities: [], selectedRepositoryIds: [] } }),
  setViewMode: (mode) => set({ viewMode: mode }),
  setCurrentTicket: (ticket) => set({ currentTicket: ticket }),
  setSelectedTicketSlug: (slug) => set({ selectedTicketSlug: slug }),
  setDoneCollapsed: (collapsed) => set({ doneCollapsed: collapsed }),
  clearError: () => set({ error: null }),
}));

reconnectRegistry.register({
  name: "ticket:list",
  fn: () => useTicketStore.getState().fetchTickets?.(),
  priority: "deferred",
});
