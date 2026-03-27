import { request, orgPath } from "./base";
import type { TicketRelation, TicketCommit, TicketComment } from "./ticketTypes";

/** Relations, Commits, Comments, and MR APIs for tickets */
export const ticketRelationsApi = {
  // Relations
  listRelations: (slug: string) =>
    request<{ relations: TicketRelation[] }>(`${orgPath("/tickets")}/${slug}/relations`),

  createRelation: (slug: string, data: { target_slug: string; relation_type: string }) =>
    request<{ relation: TicketRelation }>(`${orgPath("/tickets")}/${slug}/relations`, {
      method: "POST", body: data,
    }),

  deleteRelation: (slug: string, relationId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/relations/${relationId}`, {
      method: "DELETE",
    }),

  // Commits
  listCommits: (slug: string) =>
    request<{ commits: TicketCommit[] }>(`${orgPath("/tickets")}/${slug}/commits`),

  linkCommit: (slug: string, data: {
    commit_sha: string; commit_message?: string; commit_url?: string;
    author_name?: string; author_email?: string; committed_at?: string;
  }) =>
    request<{ commit: TicketCommit }>(`${orgPath("/tickets")}/${slug}/commits`, {
      method: "POST", body: data,
    }),

  unlinkCommit: (slug: string, commitId: number) =>
    request<{ message: string }>(`${orgPath("/tickets")}/${slug}/commits/${commitId}`, {
      method: "DELETE",
    }),

  // Merge Requests
  listMergeRequests: (slug: string) =>
    request<{
      merge_requests: Array<{
        id: number; mr_iid: number; title: string; state: string;
        mr_url: string; web_url: string; source_branch: string; target_branch: string;
        pipeline_status?: string; pipeline_id?: number; pipeline_url?: string;
        pod_id?: number;
      }>;
    }>(`${orgPath("/tickets")}/${slug}/merge-requests`),

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
    slug: string, content: string, parentId?: number,
    mentions?: Array<{ user_id: number; username: string }>
  ) =>
    request<{ comment: TicketComment }>(
      `${orgPath("/tickets")}/${slug}/comments`,
      { method: "POST", body: { content, parent_id: parentId, mentions } }
    ),

  updateComment: (
    slug: string, commentId: number, content: string,
    mentions?: Array<{ user_id: number; username: string }>
  ) =>
    request<{ comment: TicketComment }>(
      `${orgPath("/tickets")}/${slug}/comments/${commentId}`,
      { method: "PUT", body: { content, mentions } }
    ),

  deleteComment: (slug: string, commentId: number) =>
    request<{ message: string }>(
      `${orgPath("/tickets")}/${slug}/comments/${commentId}`,
      { method: "DELETE" }
    ),
};
