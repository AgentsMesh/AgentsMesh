import { request, orgPath } from "./base";

// Ticket types
export type TicketType = "task" | "bug" | "feature" | "improvement" | "epic" | "subtask" | "story";
export type TicketStatus = "backlog" | "todo" | "in_progress" | "in_review" | "done" | "cancelled";
export type TicketPriority = "none" | "low" | "medium" | "high" | "urgent";

export interface TicketData {
  id: number;
  number: number;
  identifier: string;
  type: TicketType;
  title: string;
  description?: string;
  content?: string;
  status: TicketStatus;
  priority: TicketPriority;
  severity?: string;
  estimate?: number;
  due_date?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  reporter?: { id: number; username: string; name?: string; avatar_url?: string };
  assignees?: Array<{ id: number; username: string; name?: string; avatar_url?: string }>;
  labels?: Array<{ id: number; name: string; color: string }>;
  repository_id?: number;
  repository?: { id: number; name: string };
  parent_ticket?: { id: number; identifier: string; title: string };
}

export interface TicketRelation {
  id: number;
  source_ticket_id: number;
  target_ticket_id: number;
  relation_type: string;
  source_ticket?: { id: number; identifier: string; title: string };
  target_ticket?: { id: number; identifier: string; title: string };
  created_at: string;
}

export interface TicketCommit {
  id: number;
  ticket_id: number;
  commit_sha: string;
  commit_message?: string;
  commit_url?: string;
  author_name?: string;
  author_email?: string;
  committed_at?: string;
  created_at: string;
}

export interface BoardColumn {
  status: string;
  tickets: TicketData[];
  count: number;
}

// Tickets API
export const ticketApi = {
  list: (filters?: {
    status?: string;
    priority?: string;
    type?: string;
    assigneeId?: number;
    repositoryId?: number;
    search?: string;
    limit?: number;
    offset?: number;
  }) => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value));
        }
      });
    }
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ tickets: TicketData[]; total: number }>(`${orgPath("/tickets")}${query}`);
  },

  get: async (identifier: string) => {
    const response = await request<{ ticket: TicketData }>(`${orgPath("/tickets")}/${identifier}`);
    return response.ticket;
  },

  create: async (data: {
    repositoryId?: number;
    type: string;
    title: string;
    description?: string;
    content?: string;
    priority?: string;
    severity?: string;
    estimate?: number;
    assigneeIds?: number[];
    labels?: string[];
    parentId?: number;
  }) => {
    const response = await request<{ ticket: TicketData }>(orgPath("/tickets"), {
      method: "POST",
      body: {
        repository_id: data.repositoryId,
        type: data.type,
        title: data.title,
        description: data.description,
        content: data.content,
        priority: data.priority,
        severity: data.severity,
        estimate: data.estimate,
        assignee_ids: data.assigneeIds,
        labels: data.labels,
        parent_ticket_id: data.parentId,
      },
    });
    return response.ticket;
  },

  update: async (identifier: string, data: {
    title?: string;
    description?: string;
    content?: string;
    type?: string;
    status?: string;
    priority?: string;
    severity?: string;
    estimate?: number;
    assigneeIds?: number[];
    labels?: string[];
  }) => {
    const response = await request<{ ticket: TicketData }>(`${orgPath("/tickets")}/${identifier}`, {
      method: "PUT",
      body: data,
    });
    return response.ticket;
  },

  delete: (identifier: string) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}`, {
      method: "DELETE",
    }),

  updateStatus: (identifier: string, status: string) =>
    request<TicketData>(`${orgPath("/tickets")}/${identifier}/status`, {
      method: "PATCH",
      body: { status },
    }),

  // Active tickets (in_progress or in_review)
  getActive: (limit?: number) => {
    const params = limit ? `?limit=${limit}` : "";
    return request<{ tickets: TicketData[] }>(`${orgPath("/tickets/active")}${params}`);
  },

  // Board view
  getBoard: (repositoryId?: number) => {
    const params = repositoryId ? `?repository_id=${repositoryId}` : "";
    return request<{ columns: BoardColumn[] }>(`${orgPath("/tickets/board")}${params}`);
  },

  // Sub-tickets
  getSubTickets: (identifier: string) =>
    request<{ tickets: TicketData[] }>(`${orgPath("/tickets")}/${identifier}/sub-tickets`),

  // Relations
  listRelations: (identifier: string) =>
    request<{ relations: TicketRelation[] }>(`${orgPath("/tickets")}/${identifier}/relations`),

  createRelation: (identifier: string, data: { target_ticket_id: number; relation_type: string }) =>
    request<{ relation: TicketRelation }>(`${orgPath("/tickets")}/${identifier}/relations`, {
      method: "POST",
      body: data,
    }),

  deleteRelation: (identifier: string, relationId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/relations/${relationId}`, {
      method: "DELETE",
    }),

  // Commits
  listCommits: (identifier: string) =>
    request<{ commits: TicketCommit[] }>(`${orgPath("/tickets")}/${identifier}/commits`),

  linkCommit: (identifier: string, data: {
    commit_sha: string;
    commit_message?: string;
    commit_url?: string;
    author_name?: string;
    author_email?: string;
    committed_at?: string;
  }) =>
    request<{ commit: TicketCommit }>(`${orgPath("/tickets")}/${identifier}/commits`, {
      method: "POST",
      body: data,
    }),

  unlinkCommit: (identifier: string, commitId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/commits/${commitId}`, {
      method: "DELETE",
    }),

  // Merge Requests
  listMergeRequests: (identifier: string) =>
    request<{
      merge_requests: Array<{
        id: number;
        mr_iid: number;
        title: string;
        state: string;
        web_url: string;
        source_branch: string;
        target_branch: string;
      }>;
    }>(`${orgPath("/tickets")}/${identifier}/merge-requests`),

  // Labels
  listLabels: (repositoryId?: number) => {
    const params = repositoryId ? `?repository_id=${repositoryId}` : "";
    return request<{ labels: Array<{ id: number; name: string; color: string }> }>(
      `${orgPath("/labels")}${params}`
    );
  },

  createLabel: (name: string, color: string, repositoryId?: number) =>
    request<{ id: number; name: string; color: string }>(orgPath("/labels"), {
      method: "POST",
      body: { name, color, repository_id: repositoryId },
    }),

  updateLabel: (id: number, data: { name?: string; color?: string }) =>
    request<{ id: number; name: string; color: string }>(`${orgPath("/labels")}/${id}`, {
      method: "PUT",
      body: data,
    }),

  deleteLabel: (id: number) =>
    request<{ message: string }>(`${orgPath("/labels")}/${id}`, {
      method: "DELETE",
    }),

  // Assignees
  addAssignee: (identifier: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/assignees`, {
      method: "POST",
      body: { user_id: userId },
    }),

  removeAssignee: (identifier: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/assignees/${userId}`, {
      method: "DELETE",
    }),

  // Ticket labels
  addLabel: (identifier: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/labels`, {
      method: "POST",
      body: { label_id: labelId },
    }),

  removeLabel: (identifier: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${identifier}/labels/${labelId}`, {
      method: "DELETE",
    }),

  // Pods (Mesh integration)
  getPods: (identifier: string, activeOnly?: boolean) => {
    const params = activeOnly ? "?active=true" : "";
    return request<{
      pods: Array<{
        pod_key: string;
        status: string;
        agent_status: string;
        model?: string;
        started_at?: string;
        runner_id: number;
        created_by_id: number;
      }>;
    }>(`${orgPath("/tickets")}/${identifier}/pods${params}`);
  },

  createPod: (identifier: string, data: {
    runner_id: number;
    initial_prompt?: string;
    model?: string;
    permission_mode?: string;
    think_level?: string;
  }) =>
    request<{
      message: string;
      pod: {
        pod_key: string;
        status: string;
      };
    }>(`${orgPath("/tickets")}/${identifier}/pods`, {
      method: "POST",
      body: data,
    }),

  // Batch pods
  batchGetPods: (ticketIds: number[]) =>
    request<{
      ticket_pods: Record<number, Array<{
        pod_key: string;
        status: string;
        agent_status: string;
      }>>;
    }>(orgPath("/tickets/batch-pods"), {
      method: "POST",
      body: { ticket_ids: ticketIds },
    }),
};
