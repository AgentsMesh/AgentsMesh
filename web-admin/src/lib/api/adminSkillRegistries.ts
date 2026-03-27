import { apiClient } from "./base";
import type { SkillRegistry, CreateSkillRegistryRequest } from "./adminTypesExtended";

export async function listSkillRegistries(): Promise<{ items: SkillRegistry[]; total: number }> {
  return apiClient.get<{ items: SkillRegistry[]; total: number }>("/skill-registries");
}

export async function createSkillRegistry(data: CreateSkillRegistryRequest): Promise<SkillRegistry> {
  return apiClient.post<SkillRegistry>("/skill-registries", data);
}

export async function syncSkillRegistry(id: number): Promise<{ message: string; registry: SkillRegistry }> {
  return apiClient.post<{ message: string; registry: SkillRegistry }>(`/skill-registries/${id}/sync`);
}

export async function deleteSkillRegistry(id: number): Promise<void> {
  return apiClient.delete<void>(`/skill-registries/${id}`);
}
