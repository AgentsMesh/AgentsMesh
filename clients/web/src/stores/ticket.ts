import { useMemo } from "react";
import { create } from "zustand";
import type { TicketData, TicketStatus, TicketPriority, BoardColumn } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getTicketService, parseWasmAny } from "@/lib/wasm-core";
import * as ticketApi from "@/lib/api/ticketConnect";
import { useAuthStore } from "@/stores/auth";

export type { TicketStatus, TicketPriority };
export interface Label { id: number; name: string; color: string }
export interface ColumnPagination { offset: number; hasMore: boolean; loading: boolean }
export interface Ticket extends TicketData { child_tickets?: Ticket[] }
export type TicketViewMode = "list" | "board";
export interface TicketFilters {
  status?: TicketStatus; priority?: TicketPriority;
  assigneeId?: number; repositoryId?: number; search?: string;
}
interface TicketUIFilters {
  selectedStatuses: TicketStatus[]; selectedPriorities: TicketPriority[]; selectedRepositoryIds: number[];
}
const EMPTY_UI: TicketUIFilters = { selectedStatuses: [], selectedPriorities: [], selectedRepositoryIds: [] };
const toggle = <T,>(arr: T[], v: T) => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
const initPag = (cols: BoardColumn[]) => Object.fromEntries(
  cols.map((c) => [c.status, { offset: c.tickets.length, hasMore: c.tickets.length < c.count, loading: false }]),
) as Record<string, ColumnPagination>;

const svc = () => getTicketService();
const bump = () => useTicketStore.setState((s) => ({ _tick: s._tick + 1 }));
const orgSlug = (): string => useAuthStore.getState().currentOrg?.slug || "";

interface TicketState {
  _tick: number; selectedTicketSlug: string | null;
  filters: TicketFilters; uiFilters: TicketUIFilters;
  viewMode: TicketViewMode; loading: boolean; error: string | null; totalCount: number;
  priorityCounts: Record<string, number>;
  columnPagination: Record<string, ColumnPagination>; doneCollapsed: boolean;
  fetchTickets: (f?: TicketFilters) => Promise<void>; fetchBoard: (f?: TicketFilters) => Promise<void>;
  loadMoreColumn: (status: string) => Promise<void>; fetchTicket: (slug: string) => Promise<void>;
  createTicket: (d: { repositoryId: number; title: string; content?: string; priority?: TicketPriority; assigneeIds?: number[]; labels?: string[]; parentId?: number; parent_ticket_slug?: string }) => Promise<Ticket>;
  updateTicket: (slug: string, d: Partial<{ title: string; content: string; status: TicketStatus; priority: TicketPriority; repositoryId: number | null; assigneeIds: number[]; labels: string[] }>) => Promise<Ticket>;
  deleteTicket: (slug: string) => Promise<void>; updateTicketStatus: (slug: string, s: TicketStatus) => Promise<void>;
  fetchLabels: (r?: number) => Promise<void>; createLabel: (n: string, c: string, r?: number) => Promise<Label>; deleteLabel: (id: number) => Promise<void>;
  updateTicketStatusFromEvent: (slug: string, status: string, previousStatus?: string) => void;
  removeTicketFromEvent: (slug: string) => void;
  setFilters: (f: TicketFilters) => void; setUIFilters: (p: Partial<TicketUIFilters>) => void;
  toggleStatus: (s: TicketStatus) => void; togglePriority: (p: TicketPriority) => void; toggleRepository: (id: number) => void;
  clearUIFilters: () => void; setViewMode: (m: TicketViewMode) => void; setCurrentTicket: (t: Ticket | null) => void;
  setSelectedTicketSlug: (s: string | null) => void; setDoneCollapsed: (c: boolean) => void; clearError: () => void;
}

export function useTickets(): Ticket[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(() => JSON.parse(svc().tickets_json()) as Ticket[], [tick]);
}

export function useCurrentTicket(): Ticket | null {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(() => parseWasmAny<Ticket>(svc().current_ticket_json()), [tick]);
}

export function useBoardColumns(): BoardColumn[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(() => { const raw = svc().board_columns_json(); return raw ? JSON.parse(raw) : []; }, [tick]);
}

export function useLabels(): Label[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(() => JSON.parse(svc().labels_json()) as Label[], [tick]);
}

export const useTicketStore = create<TicketState>((set, get) => {
  const refresh = () => {
    const cols = JSON.parse(svc().board_columns_json() || "[]") as BoardColumn[];
    if (cols.length > 0) get().fetchBoard(get().filters); else get().fetchTickets(get().filters);
  };
  return {
    _tick: 0, selectedTicketSlug: null,
    filters: {}, uiFilters: EMPTY_UI, viewMode: "board", loading: false,
    error: null, totalCount: 0, priorityCounts: {},
    columnPagination: {} as Record<string, ColumnPagination>, doneCollapsed: true,

    fetchTickets: async (filters) => {
      const m = { ...get().filters, ...filters }; set({ error: null, filters: m });
      try {
        const r = await ticketApi.listTickets(orgSlug(), { status: m.status, limit: 500 });
        svc().set_tickets(JSON.stringify(r.items));
        set({ totalCount: r.total || 0, priorityCounts: {}, columnPagination: {}, _tick: get()._tick + 1 });
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch tickets") }); }
    },

    fetchBoard: async (filters) => {
      const m = { ...get().filters, ...filters }; set({ error: null, filters: m });
      try {
        const columns = await ticketApi.getBoard(orgSlug(), { repository_id: m.repositoryId });
        svc().set_board_columns(JSON.stringify(columns));
        set({
          totalCount: columns.reduce((s: number, c: BoardColumn) => s + c.count, 0),
          priorityCounts: {},
          columnPagination: initPag(columns),
          _tick: get()._tick + 1,
        });
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch board") }); }
    },

    loadMoreColumn: async (status) => {
      const { columnPagination: cp } = get();
      const pag = cp[status]; if (!pag?.hasMore || pag.loading) return;
      set({ columnPagination: { ...cp, [status]: { ...pag, loading: true } } });
      try {
        const r = await ticketApi.listTickets(orgSlug(), { status, offset: pag.offset, limit: 20 });
        svc().append_column_tickets(status, JSON.stringify(r.items));
        const off = pag.offset + r.items.length;
        set({
          columnPagination: { ...get().columnPagination, [status]: { offset: off, hasMore: off < (r.total || 0), loading: false } },
          _tick: get()._tick + 1,
        });
      } catch (e: unknown) { set({ columnPagination: { ...get().columnPagination, [status]: { ...pag, loading: false } }, error: getErrorMessage(e, "Failed to load more") }); }
    },

    fetchTicket: async (slug) => {
      try {
        const ticket = await ticketApi.getTicket(orgSlug(), slug);
        svc().set_current_ticket(JSON.stringify(ticket));
        bump();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch ticket") }); }
    },

    createTicket: async (data) => {
      set({ error: null });
      try {
        const t = await ticketApi.createTicket(orgSlug(), {
          title: data.title,
          content: data.content,
          priority: data.priority,
          repository_id: data.repositoryId,
          assignee_ids: data.assigneeIds,
          labels: data.labels,
          parent_ticket_slug: data.parent_ticket_slug,
        });
        svc().add_ticket(JSON.stringify(t));
        refresh();
        return t as Ticket;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create ticket") }); throw e; }
    },

    updateTicket: async (slug, data) => {
      try {
        const t = await ticketApi.updateTicket(orgSlug(), slug, {
          title: data.title,
          content: data.content,
          status: data.status,
          priority: data.priority,
          repository_id: data.repositoryId === null ? 0 : data.repositoryId,
          assignee_ids: data.assigneeIds,
          labels: data.labels,
        });
        svc().update_ticket_local(slug, JSON.stringify(t));
        bump();
        refresh();
        return t as Ticket;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update ticket") }); throw e; }
    },

    deleteTicket: async (slug) => {
      try {
        await ticketApi.deleteTicket(orgSlug(), slug);
        svc().remove_ticket(slug);
        bump();
        refresh();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete ticket") }); throw e; }
    },

    updateTicketStatus: async (slug, status) => {
      try {
        await ticketApi.updateTicketStatus(orgSlug(), slug, status);
        svc().update_ticket_status_local(slug, status);
        bump();
        refresh();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update status") }); throw e; }
    },

    fetchLabels: async (repositoryId) => {
      try {
        const labels = await ticketApi.listLabels(orgSlug(), { repository_id: repositoryId });
        svc().set_labels(JSON.stringify(labels));
        bump();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch labels") }); }
    },

    createLabel: async (name, color, repositoryId) => {
      try {
        const l = await ticketApi.createLabel(orgSlug(), name, color, { repository_id: repositoryId });
        svc().add_label(JSON.stringify(l));
        bump();
        return l as Label;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create label") }); throw e; }
    },

    deleteLabel: async (id) => {
      try {
        await ticketApi.deleteLabel(orgSlug(), id);
        svc().remove_label(id);
        bump();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete label") }); throw e; }
    },

    setFilters: (filters) => set({ filters }),
    setUIFilters: (p) => set((s) => ({ uiFilters: { ...s.uiFilters, ...p } })),
    toggleStatus: (st) => set((s) => ({ uiFilters: { ...s.uiFilters, selectedStatuses: toggle(s.uiFilters.selectedStatuses, st) } })),
    togglePriority: (pr) => set((s) => ({ uiFilters: { ...s.uiFilters, selectedPriorities: toggle(s.uiFilters.selectedPriorities, pr) } })),
    toggleRepository: (id) => set((s) => ({ uiFilters: { ...s.uiFilters, selectedRepositoryIds: toggle(s.uiFilters.selectedRepositoryIds, id) } })),
    clearUIFilters: () => set({ uiFilters: EMPTY_UI }), setViewMode: (mode) => set({ viewMode: mode }),
    setCurrentTicket: (ticket) => {
      svc().set_current_ticket(ticket ? JSON.stringify(ticket) : "");
      bump();
    },
    setSelectedTicketSlug: (slug) => set({ selectedTicketSlug: slug }),
    setDoneCollapsed: (collapsed) => set({ doneCollapsed: collapsed }), clearError: () => set({ error: null }),

    updateTicketStatusFromEvent: (slug, status) => {
      svc().update_ticket_status_local(slug, status);
      bump();
    },

    removeTicketFromEvent: (slug) => {
      svc().remove_ticket(slug);
      bump();
    },
  };
});

export function useFilteredTickets(): Ticket[] {
  const tick = useTicketStore((s) => s._tick);
  const search = useTicketStore((s) => s.filters.search);
  const { selectedStatuses, selectedPriorities, selectedRepositoryIds } = useTicketStore((s) => s.uiFilters);
  return useMemo(() => {
    if (search || selectedStatuses.length || selectedPriorities.length || selectedRepositoryIds.length) {
      return JSON.parse(getTicketService().filter_tickets_json(
        search || "",
        JSON.stringify(selectedStatuses),
        JSON.stringify(selectedPriorities),
        JSON.stringify(selectedRepositoryIds),
      )) as Ticket[];
    }
    return JSON.parse(svc().tickets_json()) as Ticket[];
  }, [tick, search, selectedStatuses, selectedPriorities, selectedRepositoryIds]);
}

export const getStatusInfo = (status: TicketStatus) => {
  const m: Record<TicketStatus, { label: string; color: string; bgColor: string }> = {
    backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" },
    todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bgColor: "bg-blue-100 dark:bg-blue-900/30" },
    in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bgColor: "bg-yellow-100 dark:bg-yellow-900/30" },
    in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bgColor: "bg-purple-100 dark:bg-purple-900/30" },
    done: { label: "Done", color: "text-green-600 dark:text-green-400", bgColor: "bg-green-100 dark:bg-green-900/30" },
  };
  return m[status] || { label: status || "Unknown", color: "text-gray-500 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" };
};

export const getPriorityInfo = (priority: TicketPriority) => {
  const m: Record<TicketPriority, { label: string; color: string; icon: string }> = {
    none: { label: "None", color: "text-gray-400 dark:text-gray-500", icon: "minus" },
    low: { label: "Low", color: "text-blue-500 dark:text-blue-400", icon: "arrow-down" },
    medium: { label: "Medium", color: "text-yellow-500 dark:text-yellow-400", icon: "arrow-right" },
    high: { label: "High", color: "text-orange-500 dark:text-orange-400", icon: "arrow-up" },
    urgent: { label: "Urgent", color: "text-red-500 dark:text-red-400", icon: "alert-triangle" },
  };
  return m[priority] || { label: priority || "None", color: "text-gray-400 dark:text-gray-500", icon: "minus" };
};

reconnectRegistry.register({
  name: "ticket:list",
  fn: () => useTicketStore.getState().fetchTickets?.(),
  priority: "deferred",
});
