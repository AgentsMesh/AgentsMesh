// Shape converters for proto.repository.v1 wire types → snake_case web shapes.
//
// Split out of repositoryConnect.ts so each file stays focused (200-line cap):
// this file is the pure data-mapping layer, repositoryConnect.ts is the RPC
// orchestration layer.

import type {
  MergeRequest as ProtoMergeRequest,
  WebhookStatus as ProtoWebhookStatus,
  WebhookSecret as ProtoWebhookSecret,
  WebhookResult as ProtoWebhookResult,
} from "@proto/repository/v1/repository_pb";
import type {
  WebhookStatus,
  WebhookSecretResponse,
  WebhookResult,
} from "@/lib/viewModels/repository";
import type { MergeRequestInfo } from "@/components/ide/BottomPanel/MergeRequestCard";

// Shared proto→RepositoryData projection — single SSOT, also consumed by the
// desktop electron-adapter. Re-exported for repositoryConnect.ts.
export { repositoryToCache as fromProtoRepository } from "@agentsmesh/electron-adapter/projections";

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
