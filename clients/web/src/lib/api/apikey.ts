// Connect-RPC adapter for proto.apikey.v1.ApiKeyService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), and
// decodes responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing web APIKeyData shape (snake_case + number) so call
// sites don't have to change. The proto types are camelCase + BigInt; the
// adapter does the mapping.
//
// PR #345 lineage: `createApiKey` returns `{ apiKey, rawKey }` (multi-field
// per conventions §9 exception). The wire carries both tag 1 and tag 2;
// the secret cannot be silently dropped on the wasm hop.

import {
  ApiKeySchema,
  CreateApiKeyRequestSchema,
  CreateApiKeyResponseSchema,
  DeleteApiKeyRequestSchema,
  DeleteApiKeyResponseSchema,
  GetApiKeyRequestSchema,
  ListApiKeysRequestSchema,
  ListApiKeysResponseSchema,
  RevokeApiKeyRequestSchema,
  RevokeApiKeyResponseSchema,
  UpdateApiKeyRequestSchema,
  type ApiKey as ProtoApiKey,
} from "@proto/apikey/v1/api_key_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getApiKeyService } from "@/lib/wasm-core";
import type { APIKeyData, CreateAPIKeyRequest, UpdateAPIKeyRequest } from "@/lib/api/apikeyTypes";
export type { APIKeyData, CreateAPIKeyRequest, UpdateAPIKeyRequest };

function fromProto(k: ProtoApiKey): APIKeyData {
  return {
    id: Number(k.id),
    organization_id: Number(k.organizationId),
    name: k.name,
    description: k.description,
    key_prefix: k.keyPrefix,
    scopes: k.scopes,
    is_enabled: k.isEnabled,
    expires_at: k.expiresAt,
    last_used_at: k.lastUsedAt,
    created_by: Number(k.createdBy),
    created_at: k.createdAt,
    updated_at: k.updatedAt,
  };
}

export async function listApiKeys(
  orgSlug: string,
  opts: { offset?: number; limit?: number } = {},
): Promise<{ items: APIKeyData[]; total: number; limit: number; offset: number }> {
  const req = create(ListApiKeysRequestSchema, {
    orgSlug,
    offset: opts.offset,
    limit: opts.limit,
  });
  const bytes = toBinary(ListApiKeysRequestSchema, req);
  const respBytes = await getApiKeyService().listApiKeysConnect(bytes);
  const resp = fromBinary(ListApiKeysResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProto),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function getApiKey(orgSlug: string, id: number): Promise<APIKeyData> {
  const req = create(GetApiKeyRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetApiKeyRequestSchema, req);
  const respBytes = await getApiKeyService().getApiKeyConnect(bytes);
  return fromProto(fromBinary(ApiKeySchema, new Uint8Array(respBytes)));
}

export interface CreateApiKeyResult {
  api_key: APIKeyData;
  raw_key: string;
}

export async function createApiKey(
  orgSlug: string,
  data: CreateAPIKeyRequest,
): Promise<CreateApiKeyResult> {
  const req = create(CreateApiKeyRequestSchema, {
    orgSlug,
    name: data.name,
    description: data.description,
    scopes: data.scopes,
    expiresIn: data.expires_in !== undefined ? BigInt(data.expires_in) : undefined,
  });
  const bytes = toBinary(CreateApiKeyRequestSchema, req);
  const respBytes = await getApiKeyService().createApiKeyConnect(bytes);
  const resp = fromBinary(CreateApiKeyResponseSchema, new Uint8Array(respBytes));
  // PR #345 contract: both api_key and raw_key MUST be present.
  if (!resp.apiKey) {
    throw new Error("createApiKey response missing api_key (proto tag 1)");
  }
  return {
    api_key: fromProto(resp.apiKey),
    raw_key: resp.rawKey,
  };
}

export async function updateApiKey(
  orgSlug: string,
  id: number,
  data: UpdateAPIKeyRequest,
): Promise<APIKeyData> {
  const req = create(UpdateApiKeyRequestSchema, {
    orgSlug,
    id: BigInt(id),
    name: data.name,
    description: data.description,
    scopes: data.scopes ?? [],
    isEnabled: data.is_enabled,
  });
  const bytes = toBinary(UpdateApiKeyRequestSchema, req);
  const respBytes = await getApiKeyService().updateApiKeyConnect(bytes);
  return fromProto(fromBinary(ApiKeySchema, new Uint8Array(respBytes)));
}

export async function revokeApiKey(orgSlug: string, id: number): Promise<void> {
  const req = create(RevokeApiKeyRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(RevokeApiKeyRequestSchema, req);
  const respBytes = await getApiKeyService().revokeApiKeyConnect(bytes);
  fromBinary(RevokeApiKeyResponseSchema, new Uint8Array(respBytes));
}

export async function deleteApiKey(orgSlug: string, id: number): Promise<void> {
  const req = create(DeleteApiKeyRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(DeleteApiKeyRequestSchema, req);
  const respBytes = await getApiKeyService().deleteApiKeyConnect(bytes);
  fromBinary(DeleteApiKeyResponseSchema, new Uint8Array(respBytes));
}

// Legacy `apiKeyApi` shape — preserved so existing call sites keep working
// during dual-track. Each call now goes through the Connect path under the
// hood; `orgSlug` is sourced from the page params at the call site.
export const apiKeyApi = {
  list: async (orgSlug: string) => {
    const resp = await listApiKeys(orgSlug);
    return { api_keys: resp.items, total: resp.total };
  },
  create: async (orgSlug: string, data: CreateAPIKeyRequest) => {
    return createApiKey(orgSlug, data);
  },
  update: async (orgSlug: string, id: number, data: UpdateAPIKeyRequest) => {
    return updateApiKey(orgSlug, id, data);
  },
  revoke: async (orgSlug: string, id: number) => {
    await revokeApiKey(orgSlug, id);
  },
  remove: async (orgSlug: string, id: number) => {
    await deleteApiKey(orgSlug, id);
  },
};
