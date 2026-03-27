import { apiClient, PaginatedResponse } from "./base";
import type { User, UserListParams, DashboardStats } from "./adminTypes";

export async function getDashboardStats(): Promise<DashboardStats> {
  return apiClient.get<DashboardStats>("/dashboard/stats");
}

export async function listUsers(params?: UserListParams): Promise<PaginatedResponse<User>> {
  return apiClient.get<PaginatedResponse<User>>("/users", params as Record<string, string | number | undefined>);
}

export async function getUser(id: number): Promise<User> {
  return apiClient.get<User>(`/users/${id}`);
}

export async function updateUser(id: number, data: { name?: string; username?: string; email?: string }): Promise<User> {
  return apiClient.put<User>(`/users/${id}`, data);
}

export async function disableUser(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/disable`);
}

export async function enableUser(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/enable`);
}

export async function grantAdmin(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/grant-admin`);
}

export async function revokeAdmin(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/revoke-admin`);
}

export async function verifyUserEmail(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/verify-email`);
}

export async function unverifyUserEmail(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/unverify-email`);
}
