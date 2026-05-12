// Connect-RPC adapter for proto.extension.v1.SkillRegistryService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (which forwards binary in / binary out — conventions
// §2.5), and decodes responses via .fromBinary(). No JSON intermediate.
//
// Legacy JSON-flavored methods (getExtensionService().list_skill_registries()
// etc.) remain available during dual-track; this file is the new lane.

import {
  CreateSkillRegistryRequestSchema,
  DeleteSkillRegistryRequestSchema,
  DeleteSkillRegistryResponseSchema,
  ListSkillRegistriesRequestSchema,
  ListSkillRegistriesResponseSchema,
  ListSkillRegistryOverridesRequestSchema,
  ListSkillRegistryOverridesResponseSchema,
  SkillRegistrySchema,
  SyncSkillRegistryRequestSchema,
  TogglePlatformRegistryRequestSchema,
  TogglePlatformRegistryResponseSchema,
  type ListSkillRegistriesResponse,
  type ListSkillRegistryOverridesResponse,
  type SkillRegistry,
  type TogglePlatformRegistryResponse,
} from "@proto/extension/v1/skill_registry_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getExtensionService } from "@/lib/wasm-core";

export async function listSkillRegistries(
  orgSlug: string,
  opts: { offset?: number; limit?: number } = {},
): Promise<ListSkillRegistriesResponse> {
  const req = create(ListSkillRegistriesRequestSchema, {
    orgSlug,
    offset: opts.offset,
    limit: opts.limit,
  });
  const bytes = toBinary(ListSkillRegistriesRequestSchema, req);
  const respBytes = await getExtensionService().listSkillRegistriesConnect(bytes);
  return fromBinary(ListSkillRegistriesResponseSchema, new Uint8Array(respBytes));
}

export async function createSkillRegistry(
  orgSlug: string,
  data: {
    repositoryUrl: string;
    branch?: string;
    sourceType?: string;
    compatibleAgents?: string[];
    authType?: string;
    authCredential?: string;
  },
): Promise<SkillRegistry> {
  const req = create(CreateSkillRegistryRequestSchema, {
    orgSlug,
    repositoryUrl: data.repositoryUrl,
    branch: data.branch,
    sourceType: data.sourceType,
    compatibleAgents: data.compatibleAgents ?? [],
    authType: data.authType,
    authCredential: data.authCredential,
  });
  const bytes = toBinary(CreateSkillRegistryRequestSchema, req);
  const respBytes = await getExtensionService().createSkillRegistryConnect(bytes);
  return fromBinary(SkillRegistrySchema, new Uint8Array(respBytes));
}

export async function syncSkillRegistry(orgSlug: string, id: bigint): Promise<SkillRegistry> {
  const req = create(SyncSkillRegistryRequestSchema, { orgSlug, id });
  const bytes = toBinary(SyncSkillRegistryRequestSchema, req);
  const respBytes = await getExtensionService().syncSkillRegistryConnect(bytes);
  return fromBinary(SkillRegistrySchema, new Uint8Array(respBytes));
}

export async function deleteSkillRegistry(orgSlug: string, id: bigint): Promise<void> {
  const req = create(DeleteSkillRegistryRequestSchema, { orgSlug, id });
  const bytes = toBinary(DeleteSkillRegistryRequestSchema, req);
  const respBytes = await getExtensionService().deleteSkillRegistryConnect(bytes);
  fromBinary(DeleteSkillRegistryResponseSchema, new Uint8Array(respBytes));
}

export async function togglePlatformRegistry(
  orgSlug: string,
  id: bigint,
  disabled: boolean,
): Promise<TogglePlatformRegistryResponse> {
  const req = create(TogglePlatformRegistryRequestSchema, { orgSlug, id, disabled });
  const bytes = toBinary(TogglePlatformRegistryRequestSchema, req);
  const respBytes = await getExtensionService().togglePlatformRegistryConnect(bytes);
  return fromBinary(TogglePlatformRegistryResponseSchema, new Uint8Array(respBytes));
}

export async function listSkillRegistryOverrides(
  orgSlug: string,
): Promise<ListSkillRegistryOverridesResponse> {
  const req = create(ListSkillRegistryOverridesRequestSchema, { orgSlug });
  const bytes = toBinary(ListSkillRegistryOverridesRequestSchema, req);
  const respBytes = await getExtensionService().listSkillRegistryOverridesConnect(bytes);
  return fromBinary(ListSkillRegistryOverridesResponseSchema, new Uint8Array(respBytes));
}
