import type {
  Repository as ProtoRepository,
  Branch as ProtoBranch,
} from "@agentsmesh/proto/repository/v1/repository_pb";
import type { RepositoryData } from "@agentsmesh/service-interface";

// Single source of truth for the proto.repository.v1 → RepositoryData
// projection, shared by web (fromProtoRepository in repositoryConnectShapes.ts)
// and desktop (ElectronRepositoryService).
export function repositoryToCache(r: ProtoRepository): RepositoryData {
  return {
    id: Number(r.id),
    organization_id: Number(r.organizationId),
    provider_type: r.providerType,
    provider_base_url: r.providerBaseUrl,
    http_clone_url: r.httpCloneUrl || undefined,
    ssh_clone_url: r.sshCloneUrl || undefined,
    external_id: r.externalId,
    name: r.name,
    slug: r.slug,
    default_branch: r.defaultBranch,
    ticket_prefix: r.ticketPrefix,
    visibility: r.visibility,
    imported_by_user_id:
      r.importedByUserId === undefined ? undefined : Number(r.importedByUserId),
    is_active: r.isActive,
    webhook_config: r.webhookConfig
      ? {
          id: r.webhookConfig.id,
          url: r.webhookConfig.url,
          events: r.webhookConfig.events ?? [],
          is_active: r.webhookConfig.isActive,
          needs_manual_setup: r.webhookConfig.needsManualSetup,
          last_error: r.webhookConfig.lastError,
          created_at: r.webhookConfig.createdAt,
        }
      : undefined,
    created_at: r.createdAt,
    updated_at: r.updatedAt,
  };
}

export function branchToCache(b: ProtoBranch): { name: string } {
  return { name: b.name };
}
