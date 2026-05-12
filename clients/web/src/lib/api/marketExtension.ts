// Connect-RPC adapter for proto.extension.v1.MarketService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the
// Uint8Array to the wasm bridge (binary in / binary out — conventions
// §2.5), and decodes responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing web SkillMarketItem / McpMarketItem shape (the
// snake_case TS interfaces in lib/api/extensionTypes.ts) so call sites
// don't have to convert. The wire is camelCase + BigInt-typed — diverging
// the public API surface across all 26 services in one PR is out of scope.

import {
  ListMarketMcpServersRequestSchema,
  ListMarketMcpServersResponseSchema,
  ListMarketSkillsRequestSchema,
  ListMarketSkillsResponseSchema,
  type McpMarketItem as ProtoMcpMarketItem,
  type SkillMarketItem as ProtoSkillMarketItem,
} from "@proto/extension/v1/market_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getExtensionService } from "@/lib/wasm-core";
import type {
  EnvVarSchemaEntry,
  McpMarketItem,
  SkillMarketItem,
} from "@/lib/api/extensionTypes";

function fromProtoSkill(s: ProtoSkillMarketItem): SkillMarketItem {
  return {
    id: Number(s.id),
    registry_id: Number(s.registryId),
    slug: s.slug,
    display_name: s.displayName,
    description: s.description,
    license: s.license,
    category: s.category,
    content_sha: s.contentSha,
    version: s.version,
    is_active: s.isActive,
  };
}

function fromProtoMcp(m: ProtoMcpMarketItem): McpMarketItem {
  const defaultArgs = parseJsonArrayOfStrings(m.defaultArgs);
  const envVarSchema: EnvVarSchemaEntry[] = m.envVarSchema.map((e) => ({
    name: e.name,
    label: e.label,
    required: e.required,
    sensitive: e.sensitive,
    placeholder: e.placeholder || undefined,
  }));
  return {
    id: Number(m.id),
    slug: m.slug,
    name: m.name,
    description: m.description,
    icon: m.icon,
    transport_type: m.transportType,
    command: m.command,
    default_args: defaultArgs,
    default_http_url: m.defaultHttpUrl,
    env_var_schema: envVarSchema,
    category: m.category,
    source: m.source || undefined,
    registry_name: m.registryName || undefined,
    version: m.version || undefined,
    repository_url: m.repositoryUrl || undefined,
  };
}

function parseJsonArrayOfStrings(raw: string): string[] | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((s): s is string => typeof s === "string") : null;
  } catch {
    return null;
  }
}

export async function listMarketSkills(
  orgSlug: string,
  opts: { query?: string; category?: string } = {},
): Promise<{ items: SkillMarketItem[]; total: number; limit: number; offset: number }> {
  const req = create(ListMarketSkillsRequestSchema, {
    orgSlug,
    query: opts.query,
    category: opts.category,
  });
  const bytes = toBinary(ListMarketSkillsRequestSchema, req);
  const respBytes = await getExtensionService().listMarketSkillsConnect(bytes);
  const resp = fromBinary(ListMarketSkillsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoSkill),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function listMarketMcpServers(
  orgSlug: string,
  opts: { query?: string; category?: string; offset?: number; limit?: number } = {},
): Promise<{ items: McpMarketItem[]; total: number; limit: number; offset: number }> {
  const req = create(ListMarketMcpServersRequestSchema, {
    orgSlug,
    query: opts.query,
    category: opts.category,
    offset: opts.offset,
    limit: opts.limit,
  });
  const bytes = toBinary(ListMarketMcpServersRequestSchema, req);
  const respBytes = await getExtensionService().listMarketMcpServersConnect(bytes);
  const resp = fromBinary(ListMarketMcpServersResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoMcp),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}
