// Shape converters for proto.repository.v1 wire types → snake_case web shapes.
//
// Split out of repositoryConnect.ts so each file stays focused (200-line cap):
// this file is the pure data-mapping layer, repositoryConnect.ts is the RPC
// orchestration layer.

import type {
  Repository as ProtoRepository,
  MergeRequest as ProtoMergeRequest,
  WebhookStatus as ProtoWebhookStatus,
  WebhookSecret as ProtoWebhookSecret,
  WebhookResult as ProtoWebhookResult,
} from "@proto/repository/v1/repository_pb";
import type {
  RepositoryData,
  WebhookStatus,
  WebhookSecretResponse,
  WebhookResult,
} from "@/lib/api/repositoryTypes";
import type { MergeRequestInfo } from "@/components/ide/BottomPanel/MergeRequestCard";

export function fromProtoRepository(r: ProtoRepository): RepositoryData {
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

export function fromProtoWebhookStatus(s: ProtoWebhookStatus): WebhookStatus {
  return {
    registered: s.registered,
    webhook_id: s.webhookId,
    webhook_url: s.webhookUrl,
    events: s.events ?? [],
    is_active: s.isActive,
    needs_manual_setup: s.needsManualSetup,
    last_error: s.lastError,
    registered_at: s.registeredAt,
  };
}

export function fromProtoWebhookSecret(s: ProtoWebhookSecret): WebhookSecretResponse {
  return {
    webhook_url: s.webhookUrl,
    webhook_secret: s.webhookSecret,
    events: s.events ?? [],
  };
}

export function fromProtoWebhookResult(r: ProtoWebhookResult): WebhookResult {
  return {
    repo_id: Number(r.repoId),
    registered: r.registered,
    webhook_id: r.webhookId,
    needs_manual_setup: r.needsManualSetup,
    manual_webhook_url: r.manualWebhookUrl,
    manual_webhook_secret: r.manualWebhookSecret,
    error: r.errorMessage,
  };
}

export function fromProtoMergeRequest(m: ProtoMergeRequest): MergeRequestInfo {
  return {
    id: Number(m.id),
    mr_iid: m.mrIid,
    title: m.title,
    state: m.state,
    mr_url: m.mrUrl,
    source_branch: m.sourceBranch,
    target_branch: m.targetBranch,
    pipeline_status: m.pipelineStatus,
    pipeline_url: m.pipelineUrl,
  };
}
