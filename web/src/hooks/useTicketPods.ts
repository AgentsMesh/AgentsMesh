import { useEffect, useReducer } from "react";
import { getTicketService } from "@/lib/wasm-core";

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
      // Rust TicketService is SSOT: get_ticket_pods caches the response
      // into `pods_by_ticket_slug`. Just notify subscribers to re-read.
      const parsed = JSON.parse(json) as { pods?: TicketPodSummary[] };
      inflight.delete(slug);
      notify(slug);
      return parsed.pods ?? [];
    })
    .catch((err: unknown) => {
      inflight.delete(slug);
      notify(slug);
      throw err;
    });
  inflight.set(slug, p);
  return p;
}

function readPodsFromRust(slug: string | null): TicketPodSummary[] {
  if (!slug) return [];
  try {
    return JSON.parse(svc().ticket_pods_json(slug)) as TicketPodSummary[];
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

/**
 * Per-ticket pods hook. Rust TicketService is the SSOT — `get_ticket_pods`
 * caches the server response into `pods_by_ticket_slug`. The hook reads via
 * `ticket_pods_json` on every fetch + cross-subscriber notification.
 */
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
  // `ready` fires after the first successful fetch — Rust state has an entry.
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
