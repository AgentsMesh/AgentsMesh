import { ticketApi, TicketStatus, TicketPriority } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import type { Ticket, Label, TicketFilters, TicketStoreDeps } from "./ticket";
import { createBoardActions } from "./ticket-board-actions";
import { createListMutations, createBoardMutations, type TicketMutations } from "./ticket-mutations";

export function createTicketActions(set: TicketStoreDeps["set"], get: TicketStoreDeps["get"]) {
  const listMut = createListMutations(get, set);
  const boardMut = createBoardMutations(get, set);
  const mutations = (): TicketMutations => (get().boardColumns.length > 0 ? boardMut : listMut);

  return {
    // ── Board-specific actions (delegated) ─────────────────────────────
    ...createBoardActions(set, get),

    // ── List mode ──────────────────────────────────────────────────────
    fetchTickets: async (filters?: TicketFilters) => {
      const mergedFilters = { ...get().filters, ...filters };
      set({ error: null, filters: mergedFilters });
      try {
        const response = await ticketApi.list({ ...mergedFilters, limit: 500 });
        // Clear board state so sidebar counts fall back to computing from tickets
        set({
          tickets: response.tickets || [], totalCount: response.total || 0,
          boardColumns: [], priorityCounts: {}, columnPagination: {},
        });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch tickets") });
      }
    },

    // ── Ticket CRUD (strategy-driven, no mode branching) ───────────────
    fetchTicket: async (slug: string) => {
      try { set({ currentTicket: await ticketApi.get(slug) }); }
      catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch ticket") }); }
    },

    createTicket: async (data: {
      repositoryId: number; title: string; content?: string;
      priority?: TicketPriority; assigneeIds?: number[]; labels?: string[]; parentId?: number;
    }) => {
      set({ error: null });
      try {
        const ticket = await ticketApi.create(data);
        mutations().insert(ticket as Ticket, (ticket as Ticket).status || "backlog");
        return ticket;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create ticket") }); throw e; }
    },

    updateTicket: async (slug: string, data: Partial<{
      title: string; content: string; status: TicketStatus; priority: TicketPriority;
      repositoryId: number | null; assigneeIds: number[]; labels: string[];
    }>) => {
      try {
        const ticket = await ticketApi.update(slug, data);
        mutations().update(slug, ticket as Ticket);
        set({ currentTicket: get().currentTicket?.slug === slug ? ticket : get().currentTicket });
        return ticket;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update ticket") }); throw e; }
    },

    deleteTicket: async (slug: string) => {
      try {
        await ticketApi.delete(slug);
        mutations().remove(slug);
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete ticket") }); throw e; }
    },

    /** Optimistic status change. Mutates state before API call, rolls back on failure. */
    updateTicketStatus: async (slug: string, status: TicketStatus) => {
      const prevTickets = get().tickets;
      const prevCurrent = get().currentTicket;
      const prevColumns = get().boardColumns;
      const ticket = prevTickets.find((t) => t.slug === slug);
      const fromStatus = ticket?.status;

      if (fromStatus && fromStatus !== status) {
        const movedTicket = { ...ticket!, status };
        mutations().moveStatus(slug, movedTicket, fromStatus, status);
        set({ currentTicket: prevCurrent?.slug === slug ? { ...prevCurrent, status } : prevCurrent });
      }

      try {
        await ticketApi.updateStatus(slug, status);
      } catch (e: unknown) {
        set({ tickets: prevTickets, currentTicket: prevCurrent, boardColumns: prevColumns,
          error: getErrorMessage(e, "Failed to update ticket status") });
        throw e;
      }
    },

    // ── Labels ─────────────────────────────────────────────────────────
    fetchLabels: async (repositoryId?: number) => {
      try { set({ labels: (await ticketApi.listLabels(repositoryId)).labels || [] }); }
      catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch labels") }); }
    },

    createLabel: async (name: string, color: string, repositoryId?: number) => {
      try {
        const label = await ticketApi.createLabel(name, color, repositoryId);
        set((state: { labels: Label[] }) => ({ labels: [...state.labels, label] }));
        return label;
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create label") }); throw e; }
    },

    deleteLabel: async (id: number) => {
      try {
        await ticketApi.deleteLabel(id);
        set((state: { labels: Label[] }) => ({ labels: state.labels.filter((l) => l.id !== id) }));
      } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete label") }); throw e; }
    },

    updateTicketStatusFromEvent: (slug: string, status: string, previousStatus?: string) => {
      const ticket = get().tickets.find((t) => t.slug === slug);
      if (!ticket || ticket.status === status) return;
      const from = previousStatus || ticket.status;
      mutations().moveStatus(slug, { ...ticket, status: status as TicketStatus }, from, status);
      if (get().currentTicket?.slug === slug) {
        set({ currentTicket: { ...get().currentTicket!, status: status as TicketStatus } });
      }
    },

    removeTicketFromEvent: (slug: string) => {
      mutations().remove(slug);
    },
  };
}
