// Connect-RPC adapter for proto.extension.v1.RepoSkillService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the
// Uint8Array to the wasm bridge (binary in / binary out — conventions
// §2.5), and decodes responses via .fromBinary(). No JSON intermediate.
//
// DEVIATION: install-from-upload (multipart) stays REST. Use the legacy
// `extensionApi.installSkillFromUpload` for that one path.

import {
  InstallSkillFromGitHubRequestSchema,
  InstallSkillFromMarketRequestSchema,
  InstalledSkillSchema,
  ListRepoSkillsRequestSchema,
  ListRepoSkillsResponseSchema,
  type InstalledSkill as ProtoInstalledSkill,
  UninstallSkillRequestSchema,
  UninstallSkillResponseSchema,
  UpdateSkillRequestSchema,
} from "@proto/extension/v1/repo_skill_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getExtensionService } from "@/lib/wasm-core";
import type { InstalledSkill } from "@/lib/api/extensionTypes";

function fromProto(s: ProtoInstalledSkill): InstalledSkill {
  return {
    id: Number(s.id),
    organization_id: Number(s.organizationId),
    repository_id: Number(s.repositoryId),
    market_item_id: s.marketItemId === undefined ? null : Number(s.marketItemId),
    scope: s.scope as "org" | "user",
    installed_by: s.installedBy === undefined ? 0 : Number(s.installedBy),
    slug: s.slug,
    install_source: s.installSource as "market" | "github" | "upload",
    source_url: s.sourceUrl,
    content_sha: s.contentSha,
    package_size: Number(s.packageSize),
    pinned_version: s.pinnedVersion === undefined ? null : s.pinnedVersion,
    is_enabled: s.isEnabled,
  };
}

export async function listRepoSkills(
  orgSlug: string,
  repositoryId: number,
  opts: { scope?: string } = {},
): Promise<{ items: InstalledSkill[]; total: number; limit: number; offset: number }> {
  const req = create(ListRepoSkillsRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    scope: opts.scope,
  });
  const bytes = toBinary(ListRepoSkillsRequestSchema, req);
  const respBytes = await getExtensionService().listRepoSkillsConnect(bytes);
  const resp = fromBinary(ListRepoSkillsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProto),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function installSkillFromMarket(
  orgSlug: string,
  repositoryId: number,
  data: { marketItemId: number; scope: string },
): Promise<InstalledSkill> {
  const req = create(InstallSkillFromMarketRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    marketItemId: BigInt(data.marketItemId),
    scope: data.scope,
  });
  const bytes = toBinary(InstallSkillFromMarketRequestSchema, req);
  const respBytes = await getExtensionService().installSkillFromMarketConnect(bytes);
  return fromProto(fromBinary(InstalledSkillSchema, new Uint8Array(respBytes)));
}

export async function installSkillFromGitHub(
  orgSlug: string,
  repositoryId: number,
  data: { url: string; branch?: string; path?: string; scope: string },
): Promise<InstalledSkill> {
  const req = create(InstallSkillFromGitHubRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    url: data.url,
    branch: data.branch,
    path: data.path,
    scope: data.scope,
  });
  const bytes = toBinary(InstallSkillFromGitHubRequestSchema, req);
  const respBytes = await getExtensionService().installSkillFromGithubConnect(bytes);
  return fromProto(fromBinary(InstalledSkillSchema, new Uint8Array(respBytes)));
}

export async function updateSkill(
  orgSlug: string,
  repositoryId: number,
  installId: number,
  data: { isEnabled?: boolean; pinnedVersion?: number },
): Promise<InstalledSkill> {
  const req = create(UpdateSkillRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    installId: BigInt(installId),
    isEnabled: data.isEnabled,
    pinnedVersion: data.pinnedVersion,
  });
  const bytes = toBinary(UpdateSkillRequestSchema, req);
  const respBytes = await getExtensionService().updateSkillConnect(bytes);
  return fromProto(fromBinary(InstalledSkillSchema, new Uint8Array(respBytes)));
}

export async function uninstallSkill(
  orgSlug: string,
  repositoryId: number,
  installId: number,
): Promise<void> {
  const req = create(UninstallSkillRequestSchema, {
    orgSlug,
    repositoryId: BigInt(repositoryId),
    installId: BigInt(installId),
  });
  const bytes = toBinary(UninstallSkillRequestSchema, req);
  const respBytes = await getExtensionService().uninstallSkillConnect(bytes);
  fromBinary(UninstallSkillResponseSchema, new Uint8Array(respBytes));
}
