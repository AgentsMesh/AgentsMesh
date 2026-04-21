import { useEffect, useReducer } from "react";
import { channelApi } from "@/lib/api/channel";
import { getChannelService } from "@/lib/wasm-core";

export interface ChannelPodSummary {
  pod_key: string;
  alias?: string;
  status: string;
  agent_status?: string;
}

const inflight = new Map<number, Promise<ChannelPodSummary[]>>();
const listeners = new Map<number, Set<() => void>>();
const svc = () => getChannelService();

function notify(channelId: number): void {
  listeners.get(channelId)?.forEach((fn) => fn());
}

function subscribe(channelId: number | null, cb: () => void): () => void {
  if (channelId == null) return () => undefined;
  const set = listeners.get(channelId) ?? new Set<() => void>();
  set.add(cb);
  listeners.set(channelId, set);
  return () => {
    const s = listeners.get(channelId);
    if (!s) return;
    s.delete(cb);
    if (s.size === 0) listeners.delete(channelId);
  };
}

async function fetchPods(channelId: number): Promise<ChannelPodSummary[]> {
  const pending = inflight.get(channelId);
  if (pending) return pending;
  const p = channelApi
    .getPods(channelId)
    .then((res) => {
      // Rust ChannelService is SSOT: getPods() now goes through WASM and caches
      // server response into `pods_by_channel`. Just trigger subscriber reread.
      inflight.delete(channelId);
      notify(channelId);
      return (res.pods ?? []) as ChannelPodSummary[];
    })
    .catch((err) => {
      inflight.delete(channelId);
      notify(channelId);
      throw err;
    });
  inflight.set(channelId, p);
  return p;
}

function readPodsFromRust(channelId: number | null): ChannelPodSummary[] {
  if (channelId == null) return [];
  try {
    return JSON.parse(svc().channel_pods_json(BigInt(channelId))) as ChannelPodSummary[];
  } catch {
    return [];
  }
}

export interface UseChannelPodsResult {
  pods: ChannelPodSummary[];
  loading: boolean;
  refresh: () => Promise<ChannelPodSummary[]>;
}

/**
 * Per-channel pods hook. Rust ChannelService is the SSOT — it caches the
 * server-returned pods list in `pods_by_channel`. The hook reads through
 * `channel_pods_json` on every fetch + cross-subscriber notification. Multiple
 * components subscribing to the same channel share one in-flight request.
 */
export function useChannelPods(channelId: number | null): UseChannelPodsResult {
  const [, force] = useReducer((n) => n + 1, 0);

  useEffect(() => {
    if (channelId == null) return;
    const unsub = subscribe(channelId, force);
    void fetchPods(channelId).catch(() => undefined);
    return unsub;
  }, [channelId]);

  const pods = readPodsFromRust(channelId);
  const loading = channelId != null && inflight.has(channelId);

  return {
    pods,
    loading,
    refresh: () => (channelId != null ? fetchPods(channelId) : Promise.resolve([])),
  };
}

/** Test utility — clears in-flight promises and listeners. */
export function __resetChannelPodsCacheForTests(): void {
  inflight.clear();
  listeners.clear();
}

/** Invalidate — kept for API compatibility; forces a re-fetch on next subscriber. */
export function invalidateChannelPods(channelId: number): void {
  inflight.delete(channelId);
  notify(channelId);
}
