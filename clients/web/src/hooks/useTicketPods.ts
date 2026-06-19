import { useEffect, useReducer } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import { ReplaceCachedPodsRequestSchema } from "@proto/pod_state/v1/pod_state_pb";
import { podToCache } from "@agentsmesh/electron-adapter/projections";
import { getTicketService, getTicketState } from "@/lib/wasm-core";

export interface TicketPodSummary {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  started_at?: string;
  runner_id: number;
  created_by_id: number;
}

const inflight = new Map<string, Promise<TicketPodSummary[]>>();
const listeners = new Map<string, Set<() => void>>();
const svc = () => getTicketService();

function notify(slug: string): void {
  listeners.get(slug)?.forEach((fn) => fn());
}

function subscribe(slug: string | null, cb: () => void): () => void {
  if (!slug) return () => undefined;
  const set = listeners.get(slug) ?? new Set<() => void>();
  set.add(cb);
  listeners.set(slug, set);
  return () => {
    const s = listeners.get(slug);
    if (!s) return;
    s.delete(cb);
    if (s.size === 0) listeners.delete(slug);
  };
}

async function fetchTicketPods(slug: string): Promise<TicketPodSummary[]> {
  const pending = inflight.get(slug);
  if (pending) return pending;
  const p = svc()
    .get_ticket_pods(slug, true)
    .then((json: string) => {
      const parsed = JSON.parse(json) as { pods?: TicketPodSummary[] };
      const pods = parsed.pods ?? [];
      // Mirror the fetched pods into runtime.state (the SSOT) so the
      // synchronous readPodsFromRust reflects them.
      try {
        getTicketState().set_ticket_pods(slug, JSON.stringify(pods));
      } catch {
        /* state mirror is best-effort; the returned pods still drive this call */
      }
      inflight.delete(slug);
      notify(slug);
      return pods;
    })
    .catch((err: unknown) => {
      inflight.delete(slug);
      notify(slug);
      throw err;
    });
  inflight.set(slug, p);
  return p;
}

// Read side (B, zero-JSON): decode the ticket→pods state proto bytes via the
// shared podToCache projection (TicketPodSummary is a subset of PodData — the
// UI reads pod_key/status/agent_status only).
function readPodsFromRust(slug: string | null): TicketPodSummary[] {
  if (!slug) return [];
  try {
    const req = fromBinary(ReplaceCachedPodsRequestSchema, getTicketState().ticket_pods_bytes(slug));
    return req.pods.map(podToCache) as unknown as TicketPodSummary[];
  } catch {
    return [];
  }
}

export interface UseTicketPodsResult {
  pods: TicketPodSummary[];
  loading: boolean;
  ready: boolean;
  error: string | null;
  refresh: () => Promise<TicketPodSummary[]>;
}

export function useTicketPods(ticketSlug: string | null): UseTicketPodsResult {
  const [, force] = useReducer((n) => n + 1, 0);

  useEffect(() => {
    if (!ticketSlug) return;
    const unsub = subscribe(ticketSlug, force);
    void fetchTicketPods(ticketSlug).catch(() => undefined);
    return unsub;
  }, [ticketSlug]);

  const pods = readPodsFromRust(ticketSlug);
  const loading = !!ticketSlug && inflight.has(ticketSlug);
  const ready = !!ticketSlug && !loading;

  return {
    pods,
    loading,
    ready,
    error: null,
    refresh: () => (ticketSlug ? fetchTicketPods(ticketSlug) : Promise.resolve([])),
  };
}

export function invalidateTicketPods(ticketSlug: string): void {
  inflight.delete(ticketSlug);
  notify(ticketSlug);
}

export function __resetTicketPodsCacheForTests(): void {
  inflight.clear();
  listeners.clear();
}
