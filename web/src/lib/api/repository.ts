import { request, orgPath } from "./base";

// Webhook status types
export interface WebhookStatus {
  registered: boolean;
  webhook_id?: string;
  webhook_url?: string;
  events?: string[];
  is_active: boolean;
  needs_manual_setup: boolean;
  last_error?: string;
  registered_at?: string;
}

export interface WebhookResult {
  repo_id: number;
  registered: boolean;
  webhook_id?: string;
  needs_manual_setup: boolean;
  manual_webhook_url?: string;
  manual_webhook_secret?: string;
  error?: string;
}

export interface WebhookSecretResponse {
  webhook_url: string;
  webhook_secret: string;
  events: string[];
}

// Repository types (self-contained, no git_provider_id)
export interface RepositoryData {
  id: number;
  organization_id: number;
  provider_type: string; // github, gitlab, gitee, generic
  provider_base_url: string; // https://github.com
  http_clone_url?: string;
  ssh_clone_url?: string;
  external_id: string;
  name: string;
  slug: string;
  default_branch: string;
  ticket_prefix?: string;
  visibility: string; // "organization" or "private"
  imported_by_user_id?: number;
  is_active: boolean;
  webhook_config?: {
    id: string;
    url: string;
    events: string[];
    is_active: boolean;
    needs_manual_setup: boolean;
    last_error?: string;
    created_at?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface CreateRepositoryRequest {
  provider_type: string;
  provider_base_url: string;
  http_clone_url?: string;
  ssh_clone_url?: string;
  external_id: string;
  name: string;
  slug: string;
  default_branch?: string;
  ticket_prefix?: string;
  visibility?: string;
}

export interface UpdateRepositoryRequest {
  name?: string;
  default_branch?: string;
  ticket_prefix?: string;
  is_active?: boolean;
  http_clone_url?: string;
  ssh_clone_url?: string;
}

// Repository API
export const repositoryApi = {
  list: () => {
    return request<{ repositories: RepositoryData[] }>(`${orgPath("/repositories")}`);
  },

  get: (id: number) =>
    request<{ repository: RepositoryData }>(`${orgPath("/repositories")}/${id}`),

  create: (data: CreateRepositoryRequest) =>
    request<{ repository: RepositoryData }>(orgPath("/repositories"), {
      method: "POST",
      body: data,
    }),

  update: (id: number, data: UpdateRepositoryRequest) =>
    request<{ repository: RepositoryData }>(`${orgPath("/repositories")}/${id}`, {
      method: "PUT",
      body: data,
    }),

  delete: (id: number) =>
    request<{ message: string }>(`${orgPath("/repositories")}/${id}`, {
      method: "DELETE",
    }),

  listBranches: (id: number, accessToken: string) =>
    request<{ branches: string[] }>(`${orgPath("/repositories")}/${id}/branches`, {
      headers: { "X-Git-Access-Token": accessToken },
    }),

  syncBranches: (id: number, accessToken: string) =>
    request<{ branches: string[] }>(`${orgPath("/repositories")}/${id}/sync-branches`, {
      method: "POST",
      body: { access_token: accessToken },
    }),

  // Webhook management
  registerWebhook: (id: number) =>
    request<{ result: WebhookResult }>(`${orgPath("/repositories")}/${id}/webhook`, {
      method: "POST",
    }),

  deleteWebhook: (id: number) =>
    request<{ message: string }>(`${orgPath("/repositories")}/${id}/webhook`, {
      method: "DELETE",
    }),

  getWebhookStatus: (id: number) =>
    request<{ webhook_status: WebhookStatus }>(`${orgPath("/repositories")}/${id}/webhook/status`),

  getWebhookSecret: (id: number) =>
    request<WebhookSecretResponse>(`${orgPath("/repositories")}/${id}/webhook/secret`),

  markWebhookConfigured: (id: number) =>
    request<{ message: string }>(`${orgPath("/repositories")}/${id}/webhook/configured`, {
      method: "POST",
    }),

  // Merge requests
  listMergeRequests: (id: number, branch?: string, state?: string) => {
    const params = new URLSearchParams();
    if (branch) params.append("branch", branch);
    if (state) params.append("state", state);
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{
      merge_requests: Array<{
        id: number;
        mr_iid: number;
        title: string;
        state: string;
        mr_url: string;
        source_branch: string;
        target_branch: string;
        pipeline_status?: string;
        pipeline_id?: number;
        pipeline_url?: string;
        ticket_id?: number;
        pod_id?: number;
      }>;
    }>(`${orgPath("/repositories")}/${id}/merge-requests${query}`);
  },
};
