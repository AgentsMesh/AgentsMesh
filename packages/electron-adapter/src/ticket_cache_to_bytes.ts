// renderer cache (snake_case TicketData JSON) → state proto bytes. Reuses
// projections/ticket.cacheTicketToProto (inverse of ticketToCache) and wraps it
// in the ReplaceCachedTicketsRequest the wasm tickets_bytes() readers decode, so
// the shared web selector decodes desktop and web identically.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedTicketsRequestSchema, SetCurrentTicketRequestSchema,
  ReplaceBoardColumnsRequestSchema, ReplaceCachedLabelsRequestSchema,
} from "@agentsmesh/proto/ticket_state/v1/ticket_state_pb";
import { BoardColumnSchema, LabelSchema } from "@agentsmesh/proto/ticket/v1/ticket_pb";
import { ReplaceCachedPodsRequestSchema } from "@agentsmesh/proto/pod_state/v1/pod_state_pb";
import { PodSchema } from "@agentsmesh/proto/pod/v1/pod_pb";
import { cacheTicketToProto, type CachedBoardColumn } from "./projections/ticket";
import type { TicketData } from "@agentsmesh/service-interface";

interface PodSummaryJson {
  pod_key?: string; status?: string; agent_status?: string;
  started_at?: string; runner_id?: number;
}

// useTicketPods mirrors the fetched summaries (snake_case) under set_ticket_pods;
// re-encode them into ReplaceCachedPodsRequest so the hook decodes via podToCache.
// (proto.pod.v1.Pod has no `model`, matching the wasm set_ticket_pods round-trip.)
export function ticketPodsBytes(podsJson: string): Uint8Array {
  const pods = JSON.parse(podsJson) as PodSummaryJson[];
  return toBinary(ReplaceCachedPodsRequestSchema, create(ReplaceCachedPodsRequestSchema, {
    pods: pods.map((p) => create(PodSchema, {
      podKey: p.pod_key ?? "", status: p.status ?? "", agentStatus: p.agent_status ?? "",
      startedAt: p.started_at, runnerId: p.runner_id ? BigInt(p.runner_id) : undefined,
    })),
  }));
}

export function ticketsBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as TicketData[];
  return toBinary(ReplaceCachedTicketsRequestSchema,
    create(ReplaceCachedTicketsRequestSchema, { tickets: list.map(cacheTicketToProto) }));
}

export function currentTicketBytes(cacheJson: string | null): Uint8Array {
  if (!cacheJson) return new Uint8Array();
  const t = JSON.parse(cacheJson) as TicketData;
  return toBinary(SetCurrentTicketRequestSchema,
    create(SetCurrentTicketRequestSchema, { ticket: cacheTicketToProto(t) }));
}

export function boardColumnsBytes(cacheJson: string): Uint8Array {
  const cols = JSON.parse(cacheJson) as CachedBoardColumn[];
  return toBinary(ReplaceBoardColumnsRequestSchema, create(ReplaceBoardColumnsRequestSchema, {
    columns: cols.map((c) => create(BoardColumnSchema, {
      status: c.status, totalCount: BigInt(c.total_count), tickets: c.tickets.map(cacheTicketToProto),
    })),
  }));
}

export function labelsBytes(cacheJson: string): Uint8Array {
  const labels = JSON.parse(cacheJson) as { id: number; name: string; color: string }[];
  return toBinary(ReplaceCachedLabelsRequestSchema, create(ReplaceCachedLabelsRequestSchema, {
    labels: labels.map((l) => create(LabelSchema, { id: BigInt(l.id), name: l.name, color: l.color })),
  }));
}
