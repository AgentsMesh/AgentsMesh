import { useMemo } from "react";
import { create } from "zustand";
import { create as protoCreate, toBinary, fromBinary } from "@bufbuild/protobuf";
import type { TicketData, TicketStatus, TicketPriority, BoardColumn } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getTicketState } from "@/lib/wasm-core";
import * as ticketApi from "@/lib/api/facade/ticketConnect";
import { readCurrentOrg } from "@/stores/auth";
import {
  ApplyTicketStatusEventRequestSchema, ApplyTicketDeletedEventRequestSchema,
  ReplaceCachedTicketsRequestSchema, InsertCreatedTicketRequestSchema,
  PatchCachedTicketRequestSchema, ReplaceBoardColumnsRequestSchema,
  SetCurrentTicketRequestSchema,
  ReplaceCachedLabelsRequestSchema, InsertCreatedLabelRequestSchema,
  RemoveCachedLabelRequestSchema, FilterTicketsRequestSchema,
  FilterTicketsResponseSchema,
} from "@proto/ticket_state/v1/ticket_state_pb";
import { BoardSchema, ListTicketsResponseSchema } from "@proto/ticket/v1/ticket_pb";
import {
  ticketToProto, labelToProto, protoTicketToTicket,
} from "@/lib/api/ticketProtoMap";
import { ticketToCache } from "@agentsmesh/electron-adapter/projections";

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

const state = () => getTicketState();
const bump = () => useTicketStore.setState((s) => ({ _tick: s._tick + 1 }));
const orgSlug = (): string => readCurrentOrg()?.slug || "";

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

// Read side (B, zero-JSON): UI is a projection of state proto bytes
// (tickets_bytes) decoded via fromBinary + ticketToCache (shared projection).
export function useTickets(): Ticket[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceCachedTicketsRequestSchema, state().tickets_bytes()).tickets.map(ticketToCache) as Ticket[],
    [tick],
  );
}

export function useCurrentTicket(): Ticket | null {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(() => {
    const bytes = state().current_ticket_bytes();
    if (bytes.length === 0) return null;
    const t = fromBinary(SetCurrentTicketRequestSchema, bytes).ticket;
    return t ? (ticketToCache(t) as Ticket) : null;
  }, [tick]);
}

// Web facade BoardColumn carries `count` (KanbanColumn falls back to
// tickets.length); decode the state proto column + project to that shape.
export function useBoardColumns(): BoardColumn[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceBoardColumnsRequestSchema, state().board_columns_bytes()).columns.map((c) => ({
      status: c.status, count: Number(c.totalCount), tickets: c.tickets.map(ticketToCache),
    })) as BoardColumn[],
    [tick],
  );
}

export function useLabels(): Label[] {
  const tick = useTicketStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceCachedLabelsRequestSchema, state().labels_bytes()).labels.map((l) => ({
      id: Number(l.id), name: l.name, color: l.color,
    })),
    [tick],
  );
}

export const useTicketStore = create<TicketState>((set, get) => {
  const refresh = () => {
    const cols = fromBinary(ReplaceBoardColumnsRequestSchema, state().board_columns_bytes()).columns;
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
        const respBytes = await ticketApi.listTicketsRaw(orgSlug(), { status: m.status, limit: 500 });
        state().apply_fetched_tickets(respBytes);
        set({ priorityCounts: {}, columnPagination: {}, _tick: get()._tick + 1 });
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch tickets") }); }
    },

    fetchBoard: async (filters) => {
      const m = { ...get().filters, ...filters }; set({ error: null, filters: m });
      try {
        // total/pagination are derived off the same wire Board the apply folds —
        // decode once for counts, hand the raw bytes to Rust set_board_columns.
        const respBytes = await ticketApi.getBoardRaw(orgSlug(), { repository_id: m.repositoryId });
        state().apply_fetched_board_columns(respBytes);
        const columns = fromBinary(BoardSchema, respBytes).columns.map((c) => ({
          status: c.status, count: Number(c.totalCount), tickets: c.tickets,
        }));
        set({
          totalCount: columns.reduce((s: number, c) => s + c.count, 0),
          priorityCounts: {},
          columnPagination: initPag(columns as unknown as BoardColumn[]),
          _tick: get()._tick + 1,
        });
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch board") }); }
    },

    loadMoreColumn: async (status) => {
      const { columnPagination: cp } = get();
      const pag = cp[status]; if (!pag?.hasMore || pag.loading) return;
      set({ columnPagination: { ...cp, [status]: { ...pag, loading: true } } });
      try {
        const respBytes = await ticketApi.listTicketsRaw(orgSlug(), {
          status, offset: pag.offset, limit: 20, repositoryId: get().filters.repositoryId,
        });
        state().apply_appended_board_column_tickets(status, respBytes);
        const resp = fromBinary(ListTicketsResponseSchema, respBytes);
        const off = pag.offset + resp.items.length;
        set({
          columnPagination: { ...get().columnPagination, [status]: { offset: off, hasMore: off < Number(resp.total || 0), loading: false } },
          _tick: get()._tick + 1,
        });
      } catch (e: unknown) { set({ columnPagination: { ...get().columnPagination, [status]: { ...pag, loading: false } }, error: getErrorMessage(e, "Failed to load more") }); }
    },

    fetchTicket: async (slug) => {
      try {
        const respBytes = await ticketApi.getTicketRaw(orgSlug(), slug);
        state().apply_fetched_current_ticket(respBytes);
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
        const req = protoCreate(InsertCreatedTicketRequestSchema, { ticket: ticketToProto(t as Ticket) });
        state().insert_created_ticket(toBinary(InsertCreatedTicketRequestSchema, req));
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
        const req = protoCreate(PatchCachedTicketRequestSchema, { slug, ticket: ticketToProto(t as Ticket) });
        state().patch_cached_ticket(toBinary(PatchCachedTicketRequestSchema, req));
        bump();
        refresh();
        return t as Ticket;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update ticket") }); throw e; }
    },

    deleteTicket: async (slug) => {
      try {
        await ticketApi.deleteTicket(orgSlug(), slug);
        const req = protoCreate(ApplyTicketDeletedEventRequestSchema, { slug });
        state().apply_ticket_deleted_event(toBinary(ApplyTicketDeletedEventRequestSchema, req));
        bump();
        refresh();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete ticket") }); throw e; }
    },

    updateTicketStatus: async (slug, status) => {
      try {
        await ticketApi.updateTicketStatus(orgSlug(), slug, status);
        const req = protoCreate(ApplyTicketStatusEventRequestSchema, { slug, status });
        state().apply_ticket_status_event(toBinary(ApplyTicketStatusEventRequestSchema, req));
        bump();
        refresh();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update status") }); throw e; }
    },

    fetchLabels: async (repositoryId) => {
      try {
        const respBytes = await ticketApi.listLabelsRaw(orgSlug(), { repository_id: repositoryId });
        state().apply_fetched_labels(respBytes);
        bump();
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch labels") }); }
    },

    createLabel: async (name, color, repositoryId) => {
      try {
        const l = await ticketApi.createLabel(orgSlug(), name, color, { repository_id: repositoryId });
        const req = protoCreate(InsertCreatedLabelRequestSchema, { label: labelToProto(l) });
        state().insert_created_label(toBinary(InsertCreatedLabelRequestSchema, req));
        bump();
        return l as Label;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create label") }); throw e; }
    },

    deleteLabel: async (id) => {
      try {
        await ticketApi.deleteLabel(orgSlug(), id);
        const req = protoCreate(RemoveCachedLabelRequestSchema, { id: BigInt(id) });
        state().remove_cached_label(toBinary(RemoveCachedLabelRequestSchema, req));
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
      const req = protoCreate(SetCurrentTicketRequestSchema, {
        ticket: ticket ? ticketToProto(ticket) : undefined,
      });
      state().set_current_ticket(toBinary(SetCurrentTicketRequestSchema, req));
      bump();
    },
    setSelectedTicketSlug: (slug) => set({ selectedTicketSlug: slug }),
    setDoneCollapsed: (collapsed) => set({ doneCollapsed: collapsed }), clearError: () => set({ error: null }),

    updateTicketStatusFromEvent: (slug, status, previousStatus) => {
      const req = protoCreate(ApplyTicketStatusEventRequestSchema, {
        slug, status, previousStatus,
      });
      state().apply_ticket_status_event(toBinary(ApplyTicketStatusEventRequestSchema, req));
      bump();
    },

    removeTicketFromEvent: (slug) => {
      const req = protoCreate(ApplyTicketDeletedEventRequestSchema, { slug });
      state().apply_ticket_deleted_event(toBinary(ApplyTicketDeletedEventRequestSchema, req));
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
      const req = protoCreate(FilterTicketsRequestSchema, {
        search: search || "",
        statuses: selectedStatuses,
        priorities: selectedPriorities,
        repositoryIds: selectedRepositoryIds.map((n) => BigInt(n)),
      });
      const respBytes = state().filter_tickets(toBinary(FilterTicketsRequestSchema, req));
      const resp = fromBinary(FilterTicketsResponseSchema, respBytes);
      return resp.tickets.map(protoTicketToTicket);
    }
    return fromBinary(ReplaceCachedTicketsRequestSchema, state().tickets_bytes()).tickets.map(ticketToCache) as Ticket[];
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
