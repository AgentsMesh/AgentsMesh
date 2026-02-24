import { request } from "./base";

// Organization Member type
export interface OrganizationMember {
  id: number;
  user_id: number;
  role: "owner" | "admin" | "member";
  joined_at: string;
  user?: {
    id: number;
    email: string;
    username: string;
    name?: string;
    avatar_url?: string;
  };
}

// Organization data type (matches backend response)
export interface OrganizationData {
  id: number;
  name: string;
  slug: string;
  logo_url?: string;
  subscription_plan?: string;
  subscription_status?: string;
  created_at?: string;
  updated_at?: string;
}

// Organization API
export const organizationApi = {
  list: () =>
    request<{ organizations: OrganizationData[] }>(
      "/api/v1/orgs"
    ),

  get: (slug: string) =>
    request<{ organization: OrganizationData }>(
      `/api/v1/orgs/${slug}`
    ),

  create: (data: { name: string; slug: string }) =>
    request<{ message: string }>("/api/v1/orgs", {
      method: "POST",
      body: data,
    }),

  update: (slug: string, data: { name?: string }) =>
    request<{ message: string }>(`/api/v1/orgs/${slug}`, {
      method: "PUT",
      body: data,
    }),

  delete: (slug: string) =>
    request<{ message: string }>(`/api/v1/orgs/${slug}`, {
      method: "DELETE",
    }),

  // Member management
  listMembers: (slug: string) =>
    request<{ members: OrganizationMember[]; total: number }>(
      `/api/v1/orgs/${slug}/members`
    ),

  inviteMember: (slug: string, email: string, role?: string) =>
    request<{ message: string; member?: OrganizationMember }>(
      `/api/v1/orgs/${slug}/members`,
      {
        method: "POST",
        body: { email, role: role || "member" },
      }
    ),

  removeMember: (slug: string, userId: number) =>
    request<{ message: string }>(
      `/api/v1/orgs/${slug}/members/${userId}`,
      {
        method: "DELETE",
      }
    ),

  updateMemberRole: (slug: string, userId: number, role: string) =>
    request<{ message: string }>(
      `/api/v1/orgs/${slug}/members/${userId}`,
      {
        method: "PUT",
        body: { role },
      }
    ),
};
