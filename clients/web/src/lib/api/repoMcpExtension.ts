// Connect-RPC adapter for proto.extension.v1.RepoMcpService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the
// Uint8Array to the wasm bridge (binary in / binary out — conventions
// §2.5), and decodes responses via .fromBinary(). No JSON intermediate.
//
// args / http_headers / env_vars round-trip as JSON strings on the wire —
// the renderer parses them locally because their schema is user-defined.

import {
  InstallCustomMcpServerRequestSchema,
  InstallMcpFromMarketRequestSchema,
  InstalledMcpServerSchema,
  ListRepoMcpServersRequestSchema,
  ListRepoMcpServersResponseSchema,
  type InstalledMcpServer as ProtoInstalledMcpServer,
  UninstallMcpServerRequestSchema,
  UninstallMcpServerResponseSchema,
  UpdateMcpServerRequestSchema,
} from "@proto/extension/v1/repo_mcp_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getExtensionService } from "@/lib/wasm-core";
import type { InstalledMcpServer } from "@/lib/api/extensionTypes";

function parseArgsArray(raw: string): string[] | null {
  if (!raw) return null;
  try {
    const arr = JSON.parse(raw);
    return Array.isArray(arr) ? arr.filter((v): v is string => typeof v === "string") : null;
  } catch {
    return null;
  }
}

function parseHeaders(raw: string): Record<string, string> | null {
  if (!raw) return null;
  try {
    const obj = JSON.parse(raw);
    if (obj && typeof obj === "object" && !Array.isArray(obj)) {
      const out: Record<string, string> = {};
      for (const [k, v] of Object.entries(obj)) {
        if (typeof v === "string") out[k] = v;
      }
      return out;
    }
    return null;
  } catch {
    return null;
  }
}

function parseEnvVars(raw: string): Record<string, string> {
  if (!raw) return {};
  try {
    const obj = JSON.parse(raw);
    if (obj && typeof obj === "object" && !Array.isArray(obj)) {
      const out: Record<string, string> = {};
      for (const [k, v] of Object.entries(obj)) {
        if (typeof v === "string") out[k] = v;
      }
      return out;
    }
    return {};
  } catch {
    return {};
  }
}

function fromProto(s: ProtoInstalledMcpServer): InstalledMcpServer {
  return {
    id: Number(s.id),
    organization_id: Number(s.organizationId),
    repository_id: Number(s.repositoryId),
    market_item_id: s.marketItemId === undefined ? null : Number(s.marketItemId),
    scope: s.scope as "org" | "user",
    installed_by: s.installedBy === undefined ? 0 : Number(s.installedBy),
    name: s.name,
    slug: s.slug,
    transport_type: s.transportType,
    command: s.command,
    args: parseArgsArray(s.args),
    http_url: s.httpUrl,
    http_headers: parseHeaders(s.httpHeaders),
    env_vars: parseEnvVars(s.envVars),
    is_enabled: s.isEnabled,
  };
}

function serializeArgs(args?: string[]): string | undefined {
  return args === undefined ? undefined : JSON.stringify(args);
}

function serializeJsonObject(obj?: Record<string, unknown>): string | undefined {
  return obj === undefined ? undefined : JSON.stringify(obj);
}

export async function listRepoMcpServers(
  orgSlug: string,
  repositoryId: number,
  opts: { scope?: string } = {},
): Promise<{ items: InstalledMcpServer[]; total: number; limit: number; offset: number }> {
  const req = create(ListRepoMcpServersRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    scope: opts.scope,
  });
  const bytes = toBinary(ListRepoMcpServersRequestSchema, req);
  const respBytes = await getExtensionService().listRepoMcpServersConnect(bytes);
  const resp = fromBinary(ListRepoMcpServersResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProto),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function installMcpFromMarket(
  orgSlug: string,
  repositoryId: number,
  data: { marketItemId: number; scope: string; envVars?: Record<string, string> },
): Promise<InstalledMcpServer> {
  const req = create(InstallMcpFromMarketRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    marketItemId: BigInt(data.marketItemId),
    scope: data.scope,
    envVars: serializeJsonObject(data.envVars),
  });
  const bytes = toBinary(InstallMcpFromMarketRequestSchema, req);
  const respBytes = await getExtensionService().installMcpFromMarketConnect(bytes);
  return fromProto(fromBinary(InstalledMcpServerSchema, new Uint8Array(respBytes)));
}

export async function installCustomMcpServer(
  orgSlug: string,
  repositoryId: number,
  data: {
    name: string;
    slug: string;
    transportType: string;
    command?: string;
    args?: string[];
    httpUrl?: string;
    httpHeaders?: Record<string, string>;
    scope: string;
    envVars?: Record<string, string>;
  },
): Promise<InstalledMcpServer> {
  const req = create(InstallCustomMcpServerRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    name: data.name,
    slug: data.slug,
    transportType: data.transportType,
    command: data.command,
    args: serializeArgs(data.args),
    httpUrl: data.httpUrl,
    httpHeaders: serializeJsonObject(data.httpHeaders),
    scope: data.scope,
    envVars: serializeJsonObject(data.envVars),
  });
  const bytes = toBinary(InstallCustomMcpServerRequestSchema, req);
  const respBytes = await getExtensionService().installCustomMcpServerConnect(bytes);
  return fromProto(fromBinary(InstalledMcpServerSchema, new Uint8Array(respBytes)));
}

export async function updateMcpServer(
  orgSlug: string,
  repositoryId: number,
  installId: number,
  data: { isEnabled?: boolean; envVars?: Record<string, string> },
): Promise<InstalledMcpServer> {
  const req = create(UpdateMcpServerRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    installId: BigInt(installId),
    isEnabled: data.isEnabled,
    envVars: serializeJsonObject(data.envVars),
  });
  const bytes = toBinary(UpdateMcpServerRequestSchema, req);
  const respBytes = await getExtensionService().updateMcpServerConnect(bytes);
  return fromProto(fromBinary(InstalledMcpServerSchema, new Uint8Array(respBytes)));
}

export async function uninstallMcpServer(
  orgSlug: string,
  repositoryId: number,
  installId: number,
): Promise<void> {
  const req = create(UninstallMcpServerRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    installId: BigInt(installId),
  });
  const bytes = toBinary(UninstallMcpServerRequestSchema, req);
  const respBytes = await getExtensionService().uninstallMcpServerConnect(bytes);
  fromBinary(UninstallMcpServerResponseSchema, new Uint8Array(respBytes));
}
