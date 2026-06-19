// Connect-RPC adapter for proto.autopilot.v1.AutopilotControllerService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing AutopilotControllerData shape (legacy autopilot.rs DTO)
// so call-sites in the autopilot store don't need to flip off camelCase + BigInt.

import {
  ActionRequestSchema,
  ActionResponseSchema,
  ApproveRequestSchema,
  CreateAutopilotControllerRequestSchema,
  GetAutopilotControllerRequestSchema,
  GetIterationsRequestSchema,
  GetIterationsResponseSchema,
  ListAutopilotControllersRequestSchema,
  ListAutopilotControllersResponseSchema,
  AutopilotControllerSchema,
  type AutopilotController as ProtoController,
  type AutopilotIteration as ProtoIteration,
} from "@proto/autopilot/v1/autopilot_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getAutopilotService } from "@/lib/wasm-core";

export interface AutopilotControllerWire {
  id: number;
  autopilot_controller_key: string;
  pod_key: string;
  phase: string;
  current_iteration: number;
  max_iterations: number;
  circuit_breaker: { state: string; reason?: string };
  user_takeover: boolean;
  prompt?: string;
  started_at?: string;
  last_iteration_at?: string;
  created_at: string;
}

export interface AutopilotIterationWire {
  id: number;
  iteration_number: number;
  status: string;
  result?: string;
  started_at?: string;
  completed_at?: string;
}

function fromProtoController(p: ProtoController): AutopilotControllerWire {
  return {
    id: Number(p.id),
    autopilot_controller_key: p.autopilotControllerKey,
    pod_key: p.podKey,
    phase: p.phase,
    current_iteration: p.currentIteration,
    max_iterations: p.maxIterations,
    circuit_breaker: {
      state: p.circuitBreaker?.state ?? "",
      reason: p.circuitBreaker?.reason || undefined,
    },
    user_takeover: p.userTakeover,
    prompt: p.prompt || undefined,
    started_at: p.startedAt,
    last_iteration_at: p.lastIterationAt,
    created_at: p.createdAt,
  };
}

function fromProtoIteration(p: ProtoIteration): AutopilotIterationWire {
  return {
    id: Number(p.id),
    iteration_number: Number(p.iterationNumber),
    status: p.status,
    result: p.result || undefined,
    started_at: p.startedAt,
    completed_at: p.completedAt,
  };
}

export interface CreateAutopilotParams {
  orgSlug: string;
  podKey: string;
  prompt?: string;
  maxIterations?: number;
  iterationTimeoutSec?: number;
  noProgressThreshold?: number;
  sameErrorThreshold?: number;
  approvalTimeoutMin?: number;
  controlAgentSlug?: string;
  controlPromptTemplate?: string;
  mcpConfigJson?: string;
}

export async function listAutopilots(orgSlug: string): Promise<AutopilotControllerWire[]> {
  const req = create(ListAutopilotControllersRequestSchema, { orgSlug });
  const bytes = toBinary(ListAutopilotControllersRequestSchema, req);
  const respBytes = await getAutopilotService().listAutopilotsConnect(bytes);
  const resp = fromBinary(ListAutopilotControllersResponseSchema, new Uint8Array(respBytes));
  return resp.items.map(fromProtoController);
}

// Raw wire bytes for the fetch→state path: the response goes straight to Rust
// apply_fetched_controllers (no TS fromProtoController + controllerToProto).
export async function listAutopilotsRaw(orgSlug: string): Promise<Uint8Array> {
  const req = create(ListAutopilotControllersRequestSchema, { orgSlug });
  const bytes = toBinary(ListAutopilotControllersRequestSchema, req);
  return new Uint8Array(await getAutopilotService().listAutopilotsConnect(bytes));
}

export async function getAutopilot(orgSlug: string, key: string): Promise<AutopilotControllerWire> {
  const req = create(GetAutopilotControllerRequestSchema, { orgSlug, key });
  const bytes = toBinary(GetAutopilotControllerRequestSchema, req);
  const respBytes = await getAutopilotService().getAutopilotConnect(bytes);
  const resp = fromBinary(AutopilotControllerSchema, new Uint8Array(respBytes));
  return fromProtoController(resp);
}

// Raw wire bytes for the fetch→state path: response (AutopilotController) →
// apply_fetched_current_controller (Rust upsert + set_current via the shared
// wire→state converter), no TS controllerSnapshotToCache round-trip.
export async function getAutopilotRaw(orgSlug: string, key: string): Promise<Uint8Array> {
  const req = create(GetAutopilotControllerRequestSchema, { orgSlug, key });
  return new Uint8Array(
    await getAutopilotService().getAutopilotConnect(toBinary(GetAutopilotControllerRequestSchema, req)),
  );
}

export async function createAutopilot(params: CreateAutopilotParams): Promise<AutopilotControllerWire> {
  const req = create(CreateAutopilotControllerRequestSchema, {
    orgSlug: params.orgSlug,
    podKey: params.podKey,
    prompt: params.prompt ?? "",
    maxIterations: params.maxIterations ?? 0,
    iterationTimeoutSec: params.iterationTimeoutSec ?? 0,
    noProgressThreshold: params.noProgressThreshold ?? 0,
    sameErrorThreshold: params.sameErrorThreshold ?? 0,
    approvalTimeoutMin: params.approvalTimeoutMin ?? 0,
    controlAgentSlug: params.controlAgentSlug ?? "",
    controlPromptTemplate: params.controlPromptTemplate ?? "",
    mcpConfigJson: params.mcpConfigJson ?? "",
  });
  const bytes = toBinary(CreateAutopilotControllerRequestSchema, req);
  const respBytes = await getAutopilotService().createAutopilotConnect(bytes);
  const resp = fromBinary(AutopilotControllerSchema, new Uint8Array(respBytes));
  return fromProtoController(resp);
}

async function sendAction(
  caller: (b: Uint8Array) => Promise<Uint8Array>,
  orgSlug: string,
  key: string,
): Promise<{ status: string; action: string }> {
  const req = create(ActionRequestSchema, { orgSlug, key });
  const bytes = toBinary(ActionRequestSchema, req);
  const respBytes = await caller(bytes);
  const resp = fromBinary(ActionResponseSchema, new Uint8Array(respBytes));
  return { status: resp.status, action: resp.action };
}

export async function pauseAutopilot(orgSlug: string, key: string) {
  return sendAction((b) => getAutopilotService().pauseAutopilotConnect(b), orgSlug, key);
}
export async function resumeAutopilot(orgSlug: string, key: string) {
  return sendAction((b) => getAutopilotService().resumeAutopilotConnect(b), orgSlug, key);
}
export async function stopAutopilot(orgSlug: string, key: string) {
  return sendAction((b) => getAutopilotService().stopAutopilotConnect(b), orgSlug, key);
}
export async function takeoverAutopilot(orgSlug: string, key: string) {
  return sendAction((b) => getAutopilotService().takeoverAutopilotConnect(b), orgSlug, key);
}
export async function handbackAutopilot(orgSlug: string, key: string) {
  return sendAction((b) => getAutopilotService().handbackAutopilotConnect(b), orgSlug, key);
}

export async function approveAutopilot(
  orgSlug: string,
  key: string,
  continueExecution?: boolean,
  additionalIterations?: number,
): Promise<{ status: string; action: string }> {
  const req = create(ApproveRequestSchema, {
    orgSlug,
    key,
    continueExecution,
    additionalIterations: additionalIterations ?? 0,
  });
  const bytes = toBinary(ApproveRequestSchema, req);
  const respBytes = await getAutopilotService().approveAutopilotConnect(bytes);
  const resp = fromBinary(ActionResponseSchema, new Uint8Array(respBytes));
  return { status: resp.status, action: resp.action };
}

export async function getAutopilotIterations(
  orgSlug: string,
  key: string,
): Promise<AutopilotIterationWire[]> {
  const req = create(GetIterationsRequestSchema, { orgSlug, key });
  const bytes = toBinary(GetIterationsRequestSchema, req);
  const respBytes = await getAutopilotService().getIterationsConnect(bytes);
  const resp = fromBinary(GetIterationsResponseSchema, new Uint8Array(respBytes));
  return resp.items.map(fromProtoIteration);
}

// Raw wire bytes for the fetch→state path → Rust apply_fetched_iterations.
export async function getAutopilotIterationsRaw(orgSlug: string, key: string): Promise<Uint8Array> {
  const req = create(GetIterationsRequestSchema, { orgSlug, key });
  const bytes = toBinary(GetIterationsRequestSchema, req);
  return new Uint8Array(await getAutopilotService().getIterationsConnect(bytes));
}
