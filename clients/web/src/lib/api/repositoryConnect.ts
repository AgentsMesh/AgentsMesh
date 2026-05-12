// Connect-RPC adapter for proto.repository.v1.RepositoryService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing snake_case web shapes (RepositoryData, WebhookStatus,
// WebhookSecretResponse, MergeRequestInfo) so call sites don't have to flip
// off camelCase + BigInt — same dual-track pattern as podConnect.ts and
// skillRegistry.ts during the migration window.
//
// Data-mapping helpers live in repositoryConnectShapes.ts (SRP split).

import {
  CreateRepositoryRequestSchema,
  DeleteRepositoryRequestSchema,
  DeleteRepositoryResponseSchema,
  DeleteRepositoryWebhookRequestSchema,
  DeleteRepositoryWebhookResponseSchema,
  GetRepositoryRequestSchema,
  GetRepositoryWebhookSecretRequestSchema,
  GetRepositoryWebhookStatusRequestSchema,
  ListRepositoriesRequestSchema,
  ListRepositoriesResponseSchema,
  ListRepositoryBranchesRequestSchema,
  ListRepositoryBranchesResponseSchema,
  ListRepositoryMergeRequestsRequestSchema,
  ListRepositoryMergeRequestsResponseSchema,
  MarkRepositoryWebhookConfiguredRequestSchema,
  MarkRepositoryWebhookConfiguredResponseSchema,
  RegisterRepositoryWebhookRequestSchema,
  RegisterRepositoryWebhookResponseSchema,
  RepositorySchema,
  SyncRepositoryBranchesRequestSchema,
  UpdateRepositoryRequestSchema,
  WebhookSecretSchema,
  WebhookStatusSchema,
} from "@proto/repository/v1/repository_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getRepositoryService } from "@/lib/wasm-core";
import type {
  RepositoryData,
  WebhookStatus,
  WebhookSecretResponse,
  WebhookResult,
} from "@/lib/api/repositoryTypes";
import type { MergeRequestInfo } from "@/components/ide/BottomPanel/MergeRequestCard";
import {
  fromProtoRepository,
  fromProtoWebhookStatus,
  fromProtoWebhookSecret,
  fromProtoWebhookResult,
  fromProtoMergeRequest,
} from "./repositoryConnectShapes";

export { fromProtoRepository };

// ============== Repository CRUD ==============

export async function listRepositories(
  orgSlug: string,
  opts: { offset?: number; limit?: number } = {},
): Promise<{ items: RepositoryData[]; total: number; limit: number; offset: number }> {
  const req = create(ListRepositoriesRequestSchema, {
    orgSlug,
    offset: opts.offset,
    limit: opts.limit,
  });
  const bytes = toBinary(ListRepositoriesRequestSchema, req);
  const respBytes = await getRepositoryService().list_repositories_connect(bytes);
  const resp = fromBinary(ListRepositoriesResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoRepository),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function getRepository(orgSlug: string, id: number): Promise<RepositoryData> {
  const req = create(GetRepositoryRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetRepositoryRequestSchema, req);
  const respBytes = await getRepositoryService().get_repository_connect(bytes);
  return fromProtoRepository(fromBinary(RepositorySchema, new Uint8Array(respBytes)));
}

export interface CreateRepositoryInput {
  provider_type: string;
  provider_base_url: string;
  http_clone_url?: string;
  ssh_clone_url?: string;
  external_id: string;
  name: string;
  slug: string;
  default_branch?: string;
  ticket_prefix?: string;
  visibility?: string;
}

export async function createRepository(
  orgSlug: string,
  input: CreateRepositoryInput,
): Promise<RepositoryData> {
  const req = create(CreateRepositoryRequestSchema, {
    orgSlug,
    providerType: input.provider_type,
    providerBaseUrl: input.provider_base_url,
    httpCloneUrl: input.http_clone_url,
    sshCloneUrl: input.ssh_clone_url,
    externalId: input.external_id,
    name: input.name,
    slug: input.slug,
    defaultBranch: input.default_branch,
    ticketPrefix: input.ticket_prefix,
    visibility: input.visibility,
  });
  const bytes = toBinary(CreateRepositoryRequestSchema, req);
  const respBytes = await getRepositoryService().create_repository_connect(bytes);
  return fromProtoRepository(fromBinary(RepositorySchema, new Uint8Array(respBytes)));
}

export interface UpdateRepositoryInput {
  name?: string;
  default_branch?: string;
  ticket_prefix?: string;
  is_active?: boolean;
  http_clone_url?: string;
  ssh_clone_url?: string;
}

export async function updateRepository(
  orgSlug: string,
  id: number,
  input: UpdateRepositoryInput,
): Promise<RepositoryData> {
  const req = create(UpdateRepositoryRequestSchema, {
    orgSlug,
    id: BigInt(id),
    name: input.name,
    defaultBranch: input.default_branch,
    ticketPrefix: input.ticket_prefix,
    isActive: input.is_active,
    httpCloneUrl: input.http_clone_url,
    sshCloneUrl: input.ssh_clone_url,
  });
  const bytes = toBinary(UpdateRepositoryRequestSchema, req);
  const respBytes = await getRepositoryService().update_repository_connect(bytes);
  return fromProtoRepository(fromBinary(RepositorySchema, new Uint8Array(respBytes)));
}

export async function deleteRepository(orgSlug: string, id: number): Promise<void> {
  const req = create(DeleteRepositoryRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(DeleteRepositoryRequestSchema, req);
  const respBytes = await getRepositoryService().delete_repository_connect(bytes);
  fromBinary(DeleteRepositoryResponseSchema, new Uint8Array(respBytes));
}

// ============== Branches + MR introspection ==============

export async function listRepositoryBranches(
  orgSlug: string,
  id: number,
  accessToken = "",
): Promise<{ items: string[]; total: number; limit: number; offset: number }> {
  const req = create(ListRepositoryBranchesRequestSchema, {
    orgSlug,
    id: BigInt(id),
    accessToken,
  });
  const bytes = toBinary(ListRepositoryBranchesRequestSchema, req);
  const respBytes = await getRepositoryService().list_repository_branches_connect(bytes);
  const resp = fromBinary(ListRepositoryBranchesResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map((b) => b.name),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function syncRepositoryBranches(
  orgSlug: string,
  id: number,
  accessToken = "",
): Promise<{ items: string[]; total: number; limit: number; offset: number }> {
  const req = create(SyncRepositoryBranchesRequestSchema, {
    orgSlug,
    id: BigInt(id),
    accessToken,
  });
  const bytes = toBinary(SyncRepositoryBranchesRequestSchema, req);
  const respBytes = await getRepositoryService().sync_repository_branches_connect(bytes);
  const resp = fromBinary(ListRepositoryBranchesResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map((b) => b.name),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function listRepositoryMergeRequests(
  orgSlug: string,
  id: number,
  opts: { branch?: string; state?: string } = {},
): Promise<{ items: MergeRequestInfo[]; total: number; limit: number; offset: number }> {
  const req = create(ListRepositoryMergeRequestsRequestSchema, {
    orgSlug,
    id: BigInt(id),
    branch: opts.branch,
    state: opts.state,
  });
  const bytes = toBinary(ListRepositoryMergeRequestsRequestSchema, req);
  const respBytes = await getRepositoryService().list_repository_merge_requests_connect(bytes);
  const resp = fromBinary(ListRepositoryMergeRequestsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoMergeRequest),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

// ============== Webhook lifecycle ==============

export async function registerRepositoryWebhook(
  orgSlug: string,
  id: number,
): Promise<WebhookResult | undefined> {
  const req = create(RegisterRepositoryWebhookRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(RegisterRepositoryWebhookRequestSchema, req);
  const respBytes = await getRepositoryService().register_repository_webhook_connect(bytes);
  const resp = fromBinary(RegisterRepositoryWebhookResponseSchema, new Uint8Array(respBytes));
  return resp.result ? fromProtoWebhookResult(resp.result) : undefined;
}

export async function deleteRepositoryWebhook(orgSlug: string, id: number): Promise<void> {
  const req = create(DeleteRepositoryWebhookRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(DeleteRepositoryWebhookRequestSchema, req);
  const respBytes = await getRepositoryService().delete_repository_webhook_connect(bytes);
  fromBinary(DeleteRepositoryWebhookResponseSchema, new Uint8Array(respBytes));
}

export async function getRepositoryWebhookStatus(
  orgSlug: string,
  id: number,
): Promise<WebhookStatus> {
  const req = create(GetRepositoryWebhookStatusRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetRepositoryWebhookStatusRequestSchema, req);
  const respBytes = await getRepositoryService().get_repository_webhook_status_connect(bytes);
  return fromProtoWebhookStatus(fromBinary(WebhookStatusSchema, new Uint8Array(respBytes)));
}

export async function getRepositoryWebhookSecret(
  orgSlug: string,
  id: number,
): Promise<WebhookSecretResponse> {
  const req = create(GetRepositoryWebhookSecretRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetRepositoryWebhookSecretRequestSchema, req);
  const respBytes = await getRepositoryService().get_repository_webhook_secret_connect(bytes);
  return fromProtoWebhookSecret(fromBinary(WebhookSecretSchema, new Uint8Array(respBytes)));
}

export async function markRepositoryWebhookConfigured(
  orgSlug: string,
  id: number,
): Promise<void> {
  const req = create(MarkRepositoryWebhookConfiguredRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(MarkRepositoryWebhookConfiguredRequestSchema, req);
  const respBytes = await getRepositoryService().mark_repository_webhook_configured_connect(bytes);
  fromBinary(MarkRepositoryWebhookConfiguredResponseSchema, new Uint8Array(respBytes));
}
