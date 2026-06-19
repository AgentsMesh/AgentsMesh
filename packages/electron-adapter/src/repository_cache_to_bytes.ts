// renderer cache (snake_case RepositoryData JSON) → state proto bytes. Inverse
// of projections/repository.repositoryToCache; mirrors the wasm
// repositories_bytes() reader so the shared web selector decodes desktop and web
// identically.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  RepositorySchema, RepositoryWebhookConfigSchema,
  type Repository as ProtoRepository,
} from "@agentsmesh/proto/repository/v1/repository_pb";
import { ReplaceCachedRepositoriesRequestSchema } from "@agentsmesh/proto/repo_state/v1/repo_state_pb";
import type { RepositoryData } from "@agentsmesh/service-interface";

export function cacheToProtoRepository(r: RepositoryData): ProtoRepository {
  return create(RepositorySchema, {
    id: BigInt(r.id), organizationId: BigInt(r.organization_id),
    providerType: r.provider_type, providerBaseUrl: r.provider_base_url,
    httpCloneUrl: r.http_clone_url ?? "", sshCloneUrl: r.ssh_clone_url ?? "",
    externalId: r.external_id, name: r.name, slug: r.slug,
    defaultBranch: r.default_branch, ticketPrefix: r.ticket_prefix, visibility: r.visibility,
    importedByUserId: r.imported_by_user_id === undefined ? undefined : BigInt(r.imported_by_user_id),
    isActive: r.is_active,
    webhookConfig: r.webhook_config
      ? create(RepositoryWebhookConfigSchema, {
          id: r.webhook_config.id, url: r.webhook_config.url, events: r.webhook_config.events ?? [],
          isActive: r.webhook_config.is_active, needsManualSetup: r.webhook_config.needs_manual_setup,
          lastError: r.webhook_config.last_error, createdAt: r.webhook_config.created_at,
        })
      : undefined,
    createdAt: r.created_at, updatedAt: r.updated_at,
  });
}

export function repositoriesBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as RepositoryData[];
  return toBinary(ReplaceCachedRepositoriesRequestSchema,
    create(ReplaceCachedRepositoriesRequestSchema, { repositories: list.map(cacheToProtoRepository) }));
}
