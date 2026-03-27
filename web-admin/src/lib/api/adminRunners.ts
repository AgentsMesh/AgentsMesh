import { apiClient, PaginatedResponse } from "./base";
import type { Runner, RunnerListParams } from "./adminTypes";

export async function listRunners(params?: RunnerListParams): Promise<PaginatedResponse<Runner>> {
  return apiClient.get<PaginatedResponse<Runner>>("/runners", params as Record<string, string | number | undefined>);
}

export async function getRunner(id: number): Promise<Runner> {
  return apiClient.get<Runner>(`/runners/${id}`);
}

export async function disableRunner(id: number): Promise<Runner> {
  return apiClient.post<Runner>(`/runners/${id}/disable`);
}

export async function enableRunner(id: number): Promise<Runner> {
  return apiClient.post<Runner>(`/runners/${id}/enable`);
}

export async function deleteRunner(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/runners/${id}`);
}
