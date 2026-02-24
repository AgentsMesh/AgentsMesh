import { request, orgPath } from "./base";

// Ticket types
export type TicketType = "task" | "bug" | "feature" | "improvement" | "epic" | "subtask" | "story";
export type TicketStatus = "backlog" | "todo" | "in_progress" | "in_review" | "done" | "cancelled";
export type TicketPriority = "none" | "low" | "medium" | "high" | "urgent";

export interface TicketData {
  id: number;
  number: number;
  slug: string;
  type: TicketType;
  title: string;
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
  assignees?: Array<{
    ticket_id: number;
    user_id: number;
    user?: { id: number; username: string; name?: string; avatar_url?: string };
  }>;
  labels?: Array<{ id: number; name: string; color: string }>;
  repository_id?: number;
  repository?: { id: number; name: string };
  parent_ticket?: { id: number; slug: string; title: string };
}

export interface TicketRelation {
  id: number;
  source_ticket_id: number;
  target_ticket_id: number;
  relation_type: string;
  source_ticket?: { id: number; slug: string; title: string };
  target_ticket?: { id: number; slug: string; title: string };
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

export interface TicketComment {
  id: number;
  ticket_id: number;
  user_id: number;
  content: string;
  parent_id?: number;
  mentions?: Array<{ user_id: number; username: string }>;
  created_at: string;
  updated_at: string;
  user?: { id: number; username: string; name?: string; avatar_url?: string };
  replies?: TicketComment[];
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
      const keyMap: Record<string, string> = {
        assigneeId: "assignee_id",
        repositoryId: "repository_id",
        search: "query",
      };
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(keyMap[key] || key, String(value));
        }
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
    repositoryId?: number;
    type: string;
    title: string;
    content?: string;
    priority?: string;
    severity?: string;
    estimate?: number;
    assigneeIds?: number[];
    labels?: string[];
    parentSlug?: string;
  }) => {
    const response = await request<{ ticket: TicketData }>(orgPath("/tickets"), {
      method: "POST",
      body: {
        repository_id: data.repositoryId,
        type: data.type,
        title: data.title,
        content: data.content,
        priority: data.priority,
        severity: data.severity,
        estimate: data.estimate,
        assignee_ids: data.assigneeIds,
        labels: data.labels,
        parent_ticket_slug: data.parentSlug,
      },
    });
    return response.ticket;
  },

  update: async (slug: string, data: {
    title?: string;
    content?: string;
    type?: string;
    status?: string;
    priority?: string;
    severity?: string;
    estimate?: number;
    repositoryId?: number | null;
    assigneeIds?: number[];
    labels?: string[];
    dueDate?: string;
  }) => {
    const body: Record<string, unknown> = {
      title: data.title,
      content: data.content,
      type: data.type,
      status: data.status,
      priority: data.priority,
      severity: data.severity,
      estimate: data.estimate,
      assignee_ids: data.assigneeIds,
      labels: data.labels,
      due_date: data.dueDate,
    };
    // Handle repositoryId: convert null -> 0 (clear) for backend, number -> as-is
    if ("repositoryId" in data) {
      body.repository_id = data.repositoryId === null ? 0 : data.repositoryId;
    }
    const response = await request<{ ticket: TicketData }>(`${orgPath("/tickets")}/${slug}`, {
      method: "PUT",
      body,
    });
    return response.ticket;
  },

  delete: (slug: string) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}`, {
      method: "DELETE",
    }),

  updateStatus: (slug: string, status: string) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/status`, {
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
    return request<{ board: { columns: BoardColumn[] } }>(`${orgPath("/tickets/board")}${params}`);
  },

  // Sub-tickets
  getSubTickets: (slug: string) =>
    request<{ sub_tickets: TicketData[] }>(`${orgPath("/tickets")}/${slug}/sub-tickets`),

  // Relations
  listRelations: (slug: string) =>
    request<{ relations: TicketRelation[] }>(`${orgPath("/tickets")}/${slug}/relations`),

  createRelation: (slug: string, data: { target_slug: string; relation_type: string }) =>
    request<{ relation: TicketRelation }>(`${orgPath("/tickets")}/${slug}/relations`, {
      method: "POST",
      body: data,
    }),

  deleteRelation: (slug: string, relationId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/relations/${relationId}`, {
      method: "DELETE",
    }),

  // Commits
  listCommits: (slug: string) =>
    request<{ commits: TicketCommit[] }>(`${orgPath("/tickets")}/${slug}/commits`),

  linkCommit: (slug: string, data: {
    commit_sha: string;
    commit_message?: string;
    commit_url?: string;
    author_name?: string;
    author_email?: string;
    committed_at?: string;
  }) =>
    request<{ commit: TicketCommit }>(`${orgPath("/tickets")}/${slug}/commits`, {
      method: "POST",
      body: data,
    }),

  unlinkCommit: (slug: string, commitId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/commits/${commitId}`, {
      method: "DELETE",
    }),

  // Merge Requests
  listMergeRequests: (slug: string) =>
    request<{
      merge_requests: Array<{
        id: number;
        mr_iid: number;
        title: string;
        state: string;
        mr_url: string;
        web_url: string;
        source_branch: string;
        target_branch: string;
        // Pipeline information
        pipeline_status?: string;
        pipeline_id?: number;
        pipeline_url?: string;
        // Pod association
        pod_id?: number;
      }>;
    }>(`${orgPath("/tickets")}/${slug}/merge-requests`),

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
  addAssignee: (slug: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/assignees`, {
      method: "POST",
      body: { user_id: userId },
    }),

  removeAssignee: (slug: string, userId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/assignees/${userId}`, {
      method: "DELETE",
    }),

  // Ticket labels
  addLabel: (slug: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/labels`, {
      method: "POST",
      body: { label_id: labelId },
    }),

  removeLabel: (slug: string, labelId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/labels/${labelId}`, {
      method: "DELETE",
    }),

  // Pods (Mesh integration)
  getPods: (slug: string, activeOnly?: boolean) => {
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
    }>(`${orgPath("/tickets")}/${slug}/pods${params}`);
  },

  createPod: (slug: string, data: {
    runner_id: number;
    initial_prompt?: string;
    model?: string;
    permission_mode?: string;
  }) =>
    request<{
      message: string;
      pod: {
        pod_key: string;
        status: string;
      };
    }>(`${orgPath("/tickets")}/${slug}/pods`, {
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

  // Comments
  listComments: (slug: string, limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit) params.append("limit", String(limit));
    if (offset) params.append("offset", String(offset));
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ comments: TicketComment[]; total: number }>(
      `${orgPath("/tickets")}/${slug}/comments${query}`
    );
  },

  createComment: (
    slug: string,
    content: string,
    parentId?: number,
    mentions?: Array<{ user_id: number; username: string }>
  ) =>
    request<{ comment: TicketComment }>(
      `${orgPath("/tickets")}/${slug}/comments`,
      {
        method: "POST",
        body: {
          content,
          parent_id: parentId,
          mentions,
        },
      }
    ),

  updateComment: (
    slug: string,
    commentId: number,
    content: string,
    mentions?: Array<{ user_id: number; username: string }>
  ) =>
    request<{ comment: TicketComment }>(
      `${orgPath("/tickets")}/${slug}/comments/${commentId}`,
      {
        method: "PUT",
        body: { content, mentions },
      }
    ),

  deleteComment: (slug: string, commentId: number) =>
    request<{ message: string }>(
      `${orgPath("/tickets")}/${slug}/comments/${commentId}`,
      {
        method: "DELETE",
      }
    ),
};
