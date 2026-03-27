import { apiClient, PaginatedResponse } from "./base";
import type { Organization, OrganizationMember, OrganizationListParams } from "./adminTypes";

export async function listOrganizations(params?: OrganizationListParams): Promise<PaginatedResponse<Organization>> {
  return apiClient.get<PaginatedResponse<Organization>>("/organizations", params as Record<string, string | number | undefined>);
}

export async function getOrganization(id: number): Promise<Organization> {
  return apiClient.get<Organization>(`/organizations/${id}`);
}

export async function getOrganizationMembers(id: number): Promise<{ organization: Organization; members: OrganizationMember[] }> {
  return apiClient.get<{ organization: Organization; members: OrganizationMember[] }>(`/organizations/${id}/members`);
}

export async function deleteOrganization(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/organizations/${id}`);
}
