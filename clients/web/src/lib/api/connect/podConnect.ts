// Connect-RPC adapter for proto.pod.v1.PodService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns snake_case web shapes (PodData, PodConnectionInfo) so call sites
// don't have to switch wire-camelCase off the proto generated types — same
// pattern as runnerConnect.ts during the dual-track migration window.

import {
  CreatePodRequestSchema,
  CreatePodResponseSchema,
  GetPodConnectionRequestSchema,
  GetPodRequestSchema,
  ListPodsByTicketRequestSchema,
  ListPodsByTicketResponseSchema,
  ListPodsRequestSchema,
  ListPodsResponseSchema,
  PodConnectionInfoSchema,
  PodSchema,
  SendPodPromptRequestSchema,
  SendPodPromptResponseSchema,
  TerminatePodRequestSchema,
  TerminatePodResponseSchema,
  UpdatePodAliasRequestSchema,
  UpdatePodAliasResponseSchema,
  UpdatePodPerpetualRequestSchema,
  UpdatePodPerpetualResponseSchema,
} from "@proto/pod/v1/pod_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
// Shared proto→PodData projection — single SSOT, also consumed by the desktop
// electron-adapter. Aliased to the historical fromProtoPod name and re-exported
// for cross-file use (stores/pod, podProtoMap, facade).
import { podToCache as fromProtoPod } from "@agentsmesh/electron-adapter/projections";
import { getPodService } from "@/lib/wasm-core";
import type { PodData } from "@/lib/api/facade/pod";

export { fromProtoPod };

export interface PodConnectionInfo {
  relay_url: string;
  token: string;
  pod_key: string;
  local_relay_url?: string;
  local_token?: string;
  local_relay_node_id?: string;
}

// ============== Pod CRUD ==============

export async function listPods(
  orgSlug: string,
  opts: {
    status?: string;
    created_by_id?: number;
    runner_id?: number;
    limit?: number;
    offset?: number;
  } = {},
): Promise<{ items: PodData[]; total: number; limit: number; offset: number }> {
  const req = create(ListPodsRequestSchema, {
    orgSlug,
    status: opts.status,
    createdById: opts.created_by_id === undefined ? undefined : BigInt(opts.created_by_id),
    runnerId: opts.runner_id === undefined ? undefined : BigInt(opts.runner_id),
    limit: opts.limit,
    offset: opts.offset,
  });
  const bytes = toBinary(ListPodsRequestSchema, req);
  const respBytes = await getPodService().list_pods_connect(bytes);
  const resp = fromBinary(ListPodsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoPod),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

// Raw wire bytes for the fetch→state path: the ListPodsResponse goes straight
// to Rust apply_fetched_pods / apply_appended_pods (no TS fromProtoPod +
// podToProtoPod). Same opts surface as listPods so the sidebar's status/limit/
// offset/created_by/runner filters carry through unchanged.
export async function listPodsRaw(
  orgSlug: string,
  opts: {
    status?: string;
    created_by_id?: number;
    runner_id?: number;
    limit?: number;
    offset?: number;
  } = {},
): Promise<Uint8Array> {
  const req = create(ListPodsRequestSchema, {
    orgSlug,
    status: opts.status,
    createdById: opts.created_by_id === undefined ? undefined : BigInt(opts.created_by_id),
    runnerId: opts.runner_id === undefined ? undefined : BigInt(opts.runner_id),
    limit: opts.limit,
    offset: opts.offset,
  });
  const bytes = toBinary(ListPodsRequestSchema, req);
  return new Uint8Array(await getPodService().list_pods_connect(bytes));
}

export async function getPod(orgSlug: string, podKey: string): Promise<PodData> {
  const req = create(GetPodRequestSchema, { orgSlug, podKey });
  const bytes = toBinary(GetPodRequestSchema, req);
  const respBytes = await getPodService().get_pod_connect(bytes);
  return fromProtoPod(fromBinary(PodSchema, new Uint8Array(respBytes)));
}

export interface CreatePodInput {
  agent_slug: string;
  runner_id?: number;
  ticket_slug?: string;
  alias?: string;
  agentfile_layer?: string;
  repository_id?: number;
  credential_profile_id?: number;
  cols?: number;
  rows?: number;
  source_pod_key?: string;
  resume_agent_session?: boolean;
  perpetual?: boolean;
}

export async function createPod(
  orgSlug: string,
  input: CreatePodInput,
): Promise<{ pod: PodData; warning?: string }> {
  const req = create(CreatePodRequestSchema, {
    orgSlug,
    agentSlug: input.agent_slug,
    runnerId: input.runner_id === undefined ? undefined : BigInt(input.runner_id),
    ticketSlug: input.ticket_slug,
    alias: input.alias,
    agentfileLayer: input.agentfile_layer,
    repositoryId: input.repository_id === undefined ? undefined : BigInt(input.repository_id),
    credentialProfileId:
      input.credential_profile_id === undefined ? undefined : BigInt(input.credential_profile_id),
    cols: input.cols ?? 0,
    rows: input.rows ?? 0,
    sourcePodKey: input.source_pod_key,
    resumeAgentSession: input.resume_agent_session,
    perpetual: input.perpetual,
  });
  const bytes = toBinary(CreatePodRequestSchema, req);
  const respBytes = await getPodService().create_pod_connect(bytes);
  const resp = fromBinary(CreatePodResponseSchema, new Uint8Array(respBytes));
  return { pod: fromProtoPod(resp.pod!), warning: resp.warning };
}

export async function terminatePod(orgSlug: string, podKey: string): Promise<string> {
  const req = create(TerminatePodRequestSchema, { orgSlug, podKey });
  const bytes = toBinary(TerminatePodRequestSchema, req);
  const respBytes = await getPodService().terminate_pod_connect(bytes);
  return fromBinary(TerminatePodResponseSchema, new Uint8Array(respBytes)).message;
}

export async function updatePodAlias(
  orgSlug: string,
  podKey: string,
  alias: string | null,
): Promise<string> {
  const req = create(UpdatePodAliasRequestSchema, {
    orgSlug,
    podKey,
    // alias is `optional string` — undefined = no change, "" = clear.
    alias: alias === null ? "" : alias,
  });
  const bytes = toBinary(UpdatePodAliasRequestSchema, req);
  const respBytes = await getPodService().update_pod_alias_connect(bytes);
  return fromBinary(UpdatePodAliasResponseSchema, new Uint8Array(respBytes)).message;
}

export async function updatePodPerpetual(
  orgSlug: string,
  podKey: string,
  perpetual: boolean,
): Promise<string> {
  const req = create(UpdatePodPerpetualRequestSchema, { orgSlug, podKey, perpetual });
  const bytes = toBinary(UpdatePodPerpetualRequestSchema, req);
  const respBytes = await getPodService().update_pod_perpetual_connect(bytes);
  return fromBinary(UpdatePodPerpetualResponseSchema, new Uint8Array(respBytes)).message;
}

export async function getPodConnection(
  orgSlug: string,
  podKey: string,
): Promise<PodConnectionInfo> {
  const req = create(GetPodConnectionRequestSchema, { orgSlug, podKey });
  const bytes = toBinary(GetPodConnectionRequestSchema, req);
  const respBytes = await getPodService().get_pod_connection_connect(bytes);
  const c = fromBinary(PodConnectionInfoSchema, new Uint8Array(respBytes));
  return {
    relay_url: c.relayUrl,
    token: c.token,
    pod_key: c.podKey,
    local_relay_url: c.localRelayUrl || undefined,
    local_token: c.localToken || undefined,
    local_relay_node_id: c.localRelayNodeId || undefined,
  };
}

export async function sendPodPrompt(
  orgSlug: string,
  podKey: string,
  prompt: string,
): Promise<string> {
  const req = create(SendPodPromptRequestSchema, { orgSlug, podKey, prompt });
  const bytes = toBinary(SendPodPromptRequestSchema, req);
  const respBytes = await getPodService().send_pod_prompt_connect(bytes);
  return fromBinary(SendPodPromptResponseSchema, new Uint8Array(respBytes)).status;
}

export async function listPodsByTicket(
  orgSlug: string,
  ticketId: number,
): Promise<{ items: PodData[]; total: number }> {
  const req = create(ListPodsByTicketRequestSchema, { orgSlug, ticketId: BigInt(ticketId) });
  const bytes = toBinary(ListPodsByTicketRequestSchema, req);
  const respBytes = await getPodService().list_pods_by_ticket_connect(bytes);
  const resp = fromBinary(ListPodsByTicketResponseSchema, new Uint8Array(respBytes));
  return { items: resp.items.map(fromProtoPod), total: Number(resp.total) };
}
