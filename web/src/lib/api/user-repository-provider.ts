import { request } from "./base";

// User Repository Provider types
export interface RepositoryProviderData {
  id: number;
  user_id: number;
  provider_type: string; // github, gitlab, gitee
  name: string;
  base_url: string;
  client_id?: string;
  has_client_id: boolean;
  has_bot_token: boolean;
  has_identity: boolean; // Has linked OAuth identity with access token
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface RepositoryData {
  id: string;
  name: string;
  slug: string;
  description: string;
  default_branch: string;
  visibility: string;
  http_clone_url: string;
  ssh_clone_url: string;
  web_url: string;
}

export interface CreateRepositoryProviderRequest {
  provider_type: string;
  name: string;
  base_url: string;
  client_id?: string;
  client_secret?: string;
  bot_token?: string;
}

export interface UpdateRepositoryProviderRequest {
  name?: string;
  base_url?: string;
  client_id?: string;
  client_secret?: string;
  bot_token?: string;
  is_active?: boolean;
}

// User Repository Provider API
export const userRepositoryProviderApi = {
  // List all repository providers for the current user
  list: () =>
    request<{ providers: RepositoryProviderData[] }>(
      "/api/v1/users/repository-providers"
    ),

  // Create a new repository provider
  create: (data: CreateRepositoryProviderRequest) =>
    request<{ provider: RepositoryProviderData }>(
      "/api/v1/users/repository-providers",
      {
        method: "POST",
        body: data,
      }
    ),

  // Get a single repository provider
  get: (id: number) =>
    request<{ provider: RepositoryProviderData }>(
      `/api/v1/users/repository-providers/${id}`
    ),

  // Update a repository provider
  update: (id: number, data: UpdateRepositoryProviderRequest) =>
    request<{ provider: RepositoryProviderData }>(
      `/api/v1/users/repository-providers/${id}`,
      {
        method: "PUT",
        body: data,
      }
    ),

  // Delete a repository provider
  delete: (id: number) =>
    request<{ message: string }>(`/api/v1/users/repository-providers/${id}`, {
      method: "DELETE",
    }),

  // Set as default provider
  setDefault: (id: number) =>
    request<{ message: string }>(
      `/api/v1/users/repository-providers/${id}/default`,
      {
        method: "POST",
      }
    ),

  // Test connection to a repository provider
  testConnection: (id: number) =>
    request<{ success: boolean; message?: string; error?: string }>(
      `/api/v1/users/repository-providers/${id}/test`,
      {
        method: "POST",
      }
    ),

  // List repositories accessible through a provider
  listRepositories: (
    id: number,
    options?: { page?: number; perPage?: number; search?: string }
  ) => {
    const params = new URLSearchParams();
    if (options?.page) params.append("page", String(options.page));
    if (options?.perPage) params.append("per_page", String(options.perPage));
    if (options?.search) params.append("search", options.search);
    const query = params.toString();
    return request<{
      repositories: RepositoryData[];
      page: number;
      per_page: number;
    }>(
      `/api/v1/users/repository-providers/${id}/repositories${query ? `?${query}` : ""}`
    );
  },
};
