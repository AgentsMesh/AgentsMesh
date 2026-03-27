import { ticketApi, TicketStatus, TicketPriority } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import type { Ticket, Label } from "./ticket";

interface TicketFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  assigneeId?: number;
  repositoryId?: number;
  search?: string;
}

type GetFn = () => {
  tickets: Ticket[];
  currentTicket: Ticket | null;
  filters: TicketFilters;
  totalCount: number;
};

type SetFn = (updater: object | ((state: ReturnType<GetFn>) => object)) => void;

export function createTicketActions(set: SetFn, get: GetFn) {
  return {
    fetchTickets: async (filters?: TicketFilters) => {
      const mergedFilters = { ...get().filters, ...filters };
      set({ error: null, filters: mergedFilters });
      try {
        const response = await ticketApi.list(mergedFilters);
        set({ tickets: response.tickets || [], totalCount: response.total || 0 });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch tickets") });
      }
    },

    fetchTicket: async (slug: string) => {
      try {
        const ticket = await ticketApi.get(slug);
        set({ currentTicket: ticket });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch ticket") });
      }
    },

    createTicket: async (data: {
      repositoryId: number; title: string; content?: string;
      priority?: TicketPriority; assigneeIds?: number[]; labels?: string[]; parentId?: number;
    }) => {
      set({ error: null });
      try {
        const ticket = await ticketApi.create(data);
        set((state: { tickets: Ticket[]; totalCount: number }) => ({
          tickets: [ticket, ...state.tickets], totalCount: state.totalCount + 1,
        }));
        return ticket;
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to create ticket") });
        throw error;
      }
    },

    updateTicket: async (slug: string, data: Partial<{
      title: string; content: string; status: TicketStatus; priority: TicketPriority;
      repositoryId: number | null; assigneeIds: number[]; labels: string[];
    }>) => {
      try {
        const ticket = await ticketApi.update(slug, data);
        set((state: { tickets: Ticket[]; currentTicket: Ticket | null }) => ({
          tickets: state.tickets.map((t) => (t.slug === slug ? ticket : t)),
          currentTicket: state.currentTicket?.slug === slug ? ticket : state.currentTicket,
        }));
        return ticket;
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to update ticket") });
        throw error;
      }
    },

    deleteTicket: async (slug: string) => {
      try {
        await ticketApi.delete(slug);
        set((state: { tickets: Ticket[]; totalCount: number; currentTicket: Ticket | null }) => ({
          tickets: state.tickets.filter((t) => t.slug !== slug),
          totalCount: state.totalCount - 1,
          currentTicket: state.currentTicket?.slug === slug ? null : state.currentTicket,
        }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to delete ticket") });
        throw error;
      }
    },

    updateTicketStatus: async (slug: string, status: TicketStatus) => {
      const prevTickets = get().tickets;
      const prevCurrent = get().currentTicket;
      set((state: { tickets: Ticket[]; currentTicket: Ticket | null }) => ({
        tickets: state.tickets.map((t) => (t.slug === slug ? { ...t, status } : t)),
        currentTicket: state.currentTicket?.slug === slug
          ? { ...state.currentTicket, status } : state.currentTicket,
      }));
      try {
        await ticketApi.updateStatus(slug, status);
      } catch (error: unknown) {
        set({ tickets: prevTickets, currentTicket: prevCurrent, error: getErrorMessage(error, "Failed to update ticket status") });
        throw error;
      }
    },

    fetchLabels: async (repositoryId?: number) => {
      try {
        const response = await ticketApi.listLabels(repositoryId);
        set({ labels: response.labels || [] });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch labels") });
      }
    },

    createLabel: async (name: string, color: string, repositoryId?: number) => {
      try {
        const label = await ticketApi.createLabel(name, color, repositoryId);
        set((state: { labels: Label[] }) => ({ labels: [...state.labels, label] }));
        return label;
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to create label") });
        throw error;
      }
    },

    deleteLabel: async (id: number) => {
      try {
        await ticketApi.deleteLabel(id);
        set((state: { labels: Label[] }) => ({ labels: state.labels.filter((l) => l.id !== id) }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to delete label") });
        throw error;
      }
    },
  };
}
