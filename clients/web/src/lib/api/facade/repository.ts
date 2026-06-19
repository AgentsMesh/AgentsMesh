import { readCurrentOrg } from "@/stores/auth";
import {
  listRepositories,
  listRepositoriesRaw,
  getRepository,
  createRepository,
  updateRepository,
  deleteRepository,
  getRepositoryWebhookStatus,
  registerRepositoryWebhook,
  getRepositoryWebhookSecret,
  deleteRepositoryWebhook,
  markRepositoryWebhookConfigured,
  listRepositoryMergeRequests,
  type CreateRepositoryInput,
  type UpdateRepositoryInput,
} from "../connect/repositoryConnect";

export type { RepositoryData, CreateRepositoryRequest, UpdateRepositoryRequest, WebhookStatus, WebhookResult, WebhookSecretResponse } from "@/lib/viewModels/repository";

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

export const repositoryApi = {
  list: async () => listRepositories(orgSlug()),
  listRaw: async () => listRepositoriesRaw(orgSlug()),
  get: async (id: number) => getRepository(orgSlug(), id),
  create: async (data: CreateRepositoryInput) => createRepository(orgSlug(), data),
  update: async (id: number, data: UpdateRepositoryInput) => updateRepository(orgSlug(), id, data),
  delete: async (id: number) => deleteRepository(orgSlug(), id),
  getWebhookStatus: async (id: number) => getRepositoryWebhookStatus(orgSlug(), id),
  registerWebhook: async (id: number) => registerRepositoryWebhook(orgSlug(), id),
  getWebhookSecret: async (id: number) => getRepositoryWebhookSecret(orgSlug(), id),
  deleteWebhook: async (id: number) => deleteRepositoryWebhook(orgSlug(), id),
  markWebhookConfigured: async (id: number) => markRepositoryWebhookConfigured(orgSlug(), id),
  listMergeRequests: async (repoId: number, branch?: string) =>
    listRepositoryMergeRequests(orgSlug(), repoId, { branch }),
};
