import { request, orgPath } from "./base";
import type { TicketData, BoardColumn } from "./ticketTypes";
import { ticketRelationsApi } from "./ticketRelationsApi";

// Re-export all types from ticketTypes for backward compatibility
export type { TicketStatus, TicketPriority, TicketData, TicketRelation, TicketCommit, TicketComment, BoardColumn } from "./ticketTypes";

// Core Tickets API (CRUD, board, status, sub-tickets, pods, labels, assignees)
// Merged with ticketRelationsApi for backward-compatible single import.
export const ticketApi = {
  list: (filters?: {
    status?: string; priority?: string; assigneeId?: number;
    repositoryId?: number; search?: string; limit?: number; offset?: number;
  }) => {
    const params = new URLSearchParams();
    if (filters) {
      const keyMap: Record<string, string> = {
        assigneeId: "assignee_id", repositoryId: "repository_id", search: "query",
      };
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) params.append(keyMap[key] || key, String(value));
      });
    }
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ tickets: TicketData[]; total: number }>(`${orgPath("/tickets")}${query}`);
  },

  get: async (slug: string) => {
    const response = await request<{ ticket: TicketData }>(`${orgPath("/tickets")}/${slug}`);
    return response.ticket;
  },

  create: async (data: {
    repositoryId?: number; title: string; content?: string; priority?: string;
    severity?: string; estimate?: number; assigneeIds?: number[];
    labels?: string[]; parentSlug?: string;
  }) => {
    const response = await request<{ ticket: TicketData }>(orgPath("/tickets"), {
      method: "POST",
      body: {
        repository_id: data.repositoryId, title: data.title, content: data.content,
        priority: data.priority, severity: data.severity, estimate: data.estimate,
        assignee_ids: data.assigneeIds, labels: data.labels,
        parent_ticket_slug: data.parentSlug,
      },
    });
    return response.ticket;
  },

  update: async (slug: string, data: {
    title?: string; content?: string; status?: string; priority?: string;
    severity?: string; estimate?: number; repositoryId?: number | null;
    assigneeIds?: number[]; labels?: string[]; dueDate?: string;
  }) => {
    const body: Record<string, unknown> = {
      title: data.title, content: data.content, status: data.status,
      priority: data.priority, severity: data.severity, estimate: data.estimate,
      assignee_ids: data.assigneeIds, labels: data.labels, due_date: data.dueDate,
    };
    if ("repositoryId" in data) {
      body.repository_id = data.repositoryId === null ? 0 : data.repositoryId;
    }
    const response = await request<{ ticket: TicketData }>(`${orgPath("/tickets")}/${slug}`, {
      method: "PUT", body,
    });
    return response.ticket;
  },

  delete: (slug: string) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}`, { method: "DELETE" }),

  updateStatus: (slug: string, status: string) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/status`, {
      method: "PATCH", body: { status },
    }),

  getActive: (limit?: number) => {
    const params = limit ? `?limit=${limit}` : "";
    return request<{ tickets: TicketData[] }>(`${orgPath("/tickets/active")}${params}`);
  },

  getBoard: (repositoryId?: number) => {
    const params = repositoryId ? `?repository_id=${repositoryId}` : "";
    return request<{ board: { columns: BoardColumn[] } }>(`${orgPath("/tickets/board")}${params}`);
  },

  getSubTickets: (slug: string) =>
    request<{ sub_tickets: TicketData[] }>(`${orgPath("/tickets")}/${slug}/sub-tickets`),

  // Labels
  listLabels: (repositoryId?: number) => {
    const params = repositoryId ? `?repository_id=${repositoryId}` : "";
    return request<{ labels: Array<{ id: number; name: string; color: string }> }>(
      `${orgPath("/labels")}${params}`
    );
  },
  createLabel: (name: string, color: string, repositoryId?: number) =>
    request<{ id: number; name: string; color: string }>(orgPath("/labels"), {
      method: "POST", body: { name, color, repository_id: repositoryId },
    }),
  updateLabel: (id: number, data: { name?: string; color?: string }) =>
    request<{ id: number; name: string; color: string }>(`${orgPath("/labels")}/${id}`, {
      method: "PUT", body: data,
    }),
  deleteLabel: (id: number) =>
    request<{ message: string }>(`${orgPath("/labels")}/${id}`, { method: "DELETE" }),

  // Assignees
  addAssignee: (slug: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/assignees`, {
      method: "POST", body: { user_id: userId },
    }),
  removeAssignee: (slug: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/assignees/${userId}`, {
      method: "DELETE",
    }),

  // Ticket labels
  addLabel: (slug: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/labels`, {
      method: "POST", body: { label_id: labelId },
    }),
  removeLabel: (slug: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/labels/${labelId}`, {
      method: "DELETE",
    }),

  // Pods
  getPods: (slug: string, activeOnly?: boolean) => {
    const params = activeOnly ? "?active=true" : "";
    return request<{
      pods: Array<{
        pod_key: string; status: string; agent_status: string;
        model?: string; started_at?: string; runner_id: number; created_by_id: number;
      }>;
    }>(`${orgPath("/tickets")}/${slug}/pods${params}`);
  },
  createPod: (slug: string, data: {
    runner_id: number; initial_prompt?: string; model?: string; permission_mode?: string;
  }) =>
    request<{ message: string; pod: { pod_key: string; status: string } }>(
      `${orgPath("/tickets")}/${slug}/pods`, { method: "POST", body: data }
    ),
  batchGetPods: (ticketIds: number[]) =>
    request<{
      ticket_pods: Record<number, Array<{ pod_key: string; status: string; agent_status: string }>>;
    }>(orgPath("/tickets/batch-pods"), { method: "POST", body: { ticket_ids: ticketIds } }),

  // Delegated to ticketRelationsApi
  ...ticketRelationsApi,
};
