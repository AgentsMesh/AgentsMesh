import { create } from "zustand";
import { useMemo } from "react";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { ApiError } from "@/lib/api/api-types";
import { reconnectRegistry } from "@/lib/realtime";
import { readCurrentOrg, readCurrentUser } from "@/stores/auth";
import { getErrorMessage } from "@/lib/utils";
import { initWasmCore, getPodState } from "@/lib/wasm-core";
import {
  listPods as listPodsConnect,
  getPod as getPodConnect,
  terminatePod as terminatePodConnect,
  updatePodAlias as updatePodAliasConnect,
  updatePodPerpetual as updatePodPerpetualConnect,
} from "@/lib/api/facade/podConnect";
import { fromProtoPod, podToProtoPod } from "@/lib/api/podProtoMap";
import {
  ReplaceCachedPodsRequestSchema, AppendCachedPodsRequestSchema,
  InsertCreatedPodRequestSchema, PatchPodPerpetualRequestSchema,
  MarkPodTerminatedRequestSchema,
} from "@proto/pod_state/v1/pod_state_pb";
import type { PodState, Pod } from "./podTypes";
import { SIDEBAR_STATUS_MAP, SIDEBAR_PAGE_SIZE } from "./podTypes";

export type { Pod } from "./podTypes";
export { SIDEBAR_STATUS_MAP } from "./podTypes";

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

function sidebarStatusParam(filter: string): string | undefined {
  return SIDEBAR_STATUS_MAP[filter];
}

export function usePods(): Pod[] {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => JSON.parse(getPodState().pods_json()) as Pod[], [tick]);
}

export function usePod(podKey: string | undefined): Pod | undefined {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return undefined;
    const json = getPodState().get_pod_json(podKey);
    if (!json) return undefined;
    return JSON.parse(json as string) as Pod;
  }, [tick, podKey]);
}

export function useCurrentPod(): Pod | null {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => {
    const json = getPodState().current_pod_json();
    if (!json) return null;
    return JSON.parse(json as string) as Pod;
  }, [tick]);
}

const fetchPodInflight = new Map<string, Promise<void>>();
const bump = () => usePodStore.setState((s) => ({ _tick: s._tick + 1 }));

// Monotonic id so an out-of-order sidebar fetch can't clobber a newer one. Both
// the cold-load (non-silent) and reconnect/manual (silent) paths bump it on
// entry; a response whose id is no longer the latest is discarded before it
// writes the cache/total. sidebarLoadingSeq tracks the latest NON-silent call
// (the spinner's owner) separately, so a superseding silent refresh — which
// never touches loading — can't strand a stale non-silent call's spinner.
let sidebarRequestSeq = 0;
let sidebarLoadingSeq = 0;

export const usePodStore = create<PodState>((set, get) => ({
  _tick: 0, loading: false, error: null, initProgress: {},
  podTotal: 0, podHasMore: false, loadingMore: false, currentSidebarFilter: "mine", sidebarLoadedCount: 0,

  fetchPods: async (filters) => {
    await initWasmCore();
    set({ error: null });
    try {
      const { items } = await listPodsConnect(orgSlug(), { status: filters?.status });
      const req = protoCreate(ReplaceCachedPodsRequestSchema, { pods: items.map(podToProtoPod) });
      getPodState().replace_cached_pods(toBinary(ReplaceCachedPodsRequestSchema, req));
      bump();
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch pods") });
    }
  },

  fetchPod: async (podKey) => {
    const inflight = fetchPodInflight.get(podKey);
    if (inflight) return inflight;
    const promise = (async () => {
      await initWasmCore();
      try {
        const pod = await getPodConnect(orgSlug(), podKey);
        const req = protoCreate(InsertCreatedPodRequestSchema, {
          pod: podToProtoPod(pod), clientTimestampMs: BigInt(Date.now()),
        });
        getPodState().insert_created_pod(toBinary(InsertCreatedPodRequestSchema, req));
        bump();
      } catch (error: unknown) {
        console.warn("[PodStore] fetchPod failed for", podKey, error);
        throw error;
      } finally { fetchPodInflight.delete(podKey); }
    })();
    fetchPodInflight.set(podKey, promise);
    return promise;
  },

  fetchSidebarPods: async (statusFilter, opts) => {
    const silent = opts?.silent ?? false;
    const mySeq = ++sidebarRequestSeq;
    // Only a non-silent call owns the spinner; record it so the seq guard below
    // discarding a superseded response (esp. when the superseder is a SILENT
    // refresh that never touches loading) can't leave the spinner stuck.
    if (!silent) sidebarLoadingSeq = mySeq;
    await initWasmCore();
    if (!silent) set({ error: null, currentSidebarFilter: statusFilter, loading: true });
    try {
      const uid = statusFilter === "mine" ? readCurrentUser()?.id ?? null : null;
      const { items, total } = await listPodsConnect(orgSlug(), {
        status: sidebarStatusParam(statusFilter),
        created_by_id: uid ?? undefined,
        limit: SIDEBAR_PAGE_SIZE, offset: 0,
      });
      // Write the cache + counters only if this is still the latest request for
      // the active filter: a concurrent tab switch, or a newer fetch (silent
      // reconnect vs cold load) superseding this one, must not clobber fresher
      // data. loadMorePods guards the same way before it appends.
      if (get().currentSidebarFilter === statusFilter && sidebarRequestSeq === mySeq) {
        const req = protoCreate(ReplaceCachedPodsRequestSchema, { pods: items.map(podToProtoPod) });
        getPodState().replace_cached_pods(toBinary(ReplaceCachedPodsRequestSchema, req));
        set({
          podTotal: total, podHasMore: items.length < total,
          sidebarLoadedCount: items.length, _tick: get()._tick + 1,
        });
      }
    } catch (error: unknown) {
      if (!silent) set({ error: getErrorMessage(error, "Failed to fetch pods") });
    } finally {
      // Clear the spinner iff this call still owns it (no newer non-silent call
      // took over). Decoupled from the seq/data guard above so a superseding
      // SILENT refresh can't leave loading stuck true.
      if (!silent && sidebarLoadingSeq === mySeq) set({ loading: false });
    }
  },

  loadMorePods: async () => {
    const { podHasMore, loadingMore, currentSidebarFilter, sidebarLoadedCount } = get();
    if (!podHasMore || loadingMore) return;
    // Baseline seq (loadMore appends, it doesn't bump): if a fetchSidebarPods
    // replaces the cache + resets the offset mid-flight, our page would append at
    // a stale offset (gap/duplicate rows), so discard it then.
    const mySeq = sidebarRequestSeq;
    set({ loadingMore: true });
    await initWasmCore();
    try {
      const uid = currentSidebarFilter === "mine" ? readCurrentUser()?.id ?? null : null;
      // Page from how many we've actually pulled for THIS filter, not the cache
      // length: realtime insert_created_pod upserts org-wide pods (incl. ones the
      // active filter hides) into the shared cache, so cache length drifts from
      // the server's filtered offset and would skip or duplicate rows.
      const { items: newPods, total } = await listPodsConnect(orgSlug(), {
        status: sidebarStatusParam(currentSidebarFilter),
        created_by_id: uid ?? undefined,
        limit: SIDEBAR_PAGE_SIZE, offset: sidebarLoadedCount,
      });
      // Discard if superseded: filter changed, a newer fetch bumped the seq, or
      // a fetchSidebarPods already in flight when we started reset the offset
      // (same seq baseline, so the seq check alone misses it — catch it via the
      // loaded-count change).
      if (get().currentSidebarFilter !== currentSidebarFilter
          || sidebarRequestSeq !== mySeq
          || get().sidebarLoadedCount !== sidebarLoadedCount) {
        set({ loadingMore: false });
        return;
      }
      const req = protoCreate(AppendCachedPodsRequestSchema, { pods: newPods.map(podToProtoPod) });
      getPodState().append_cached_pods(toBinary(AppendCachedPodsRequestSchema, req));
      const loaded = sidebarLoadedCount + newPods.length;
      set({ podTotal: total, podHasMore: loaded < total, loadingMore: false, sidebarLoadedCount: loaded, _tick: get()._tick + 1 });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to load more pods"), loadingMore: false });
    }
  },

  terminatePod: async (podKey) => {
    try {
      await terminatePodConnect(orgSlug(), podKey);
      const req = protoCreate(MarkPodTerminatedRequestSchema, { podKey });
      getPodState().mark_pod_terminated(toBinary(MarkPodTerminatedRequestSchema, req));
    } catch (error: unknown) {
      const msg = error instanceof Error ? error.message : String(error);
      const isNotFound = (error instanceof ApiError && error.status === 404) || msg.includes("404");
      if (!isNotFound) { set({ error: getErrorMessage(error, "Failed to terminate pod") }); throw error; }
    }
    bump();
  },

  upsertPod: (pod) => {
    const req = protoCreate(InsertCreatedPodRequestSchema, {
      pod: podToProtoPod(pod), clientTimestampMs: BigInt(Date.now()),
    });
    getPodState().insert_created_pod(toBinary(InsertCreatedPodRequestSchema, req));
    bump();
  },

  // Note: set_current_pod removed — no production caller. Method kept on
  // PodState interface for now to satisfy the typed registry shape.
  setCurrentPod: (_pod) => { bump(); },

  updatePodStatus: (podKey, status, agentStatus, errorCode, errorMessage) => {
    void applyPodStatusEvent(podKey, status, agentStatus, errorCode, errorMessage);
    bump();
  },

  updateAgentStatus: (podKey, agentStatus) => {
    void applyAgentStatusEvent(podKey, agentStatus);
    bump();
  },

  updatePodTitle: (podKey, title) => {
    void applyPodTitleEvent(podKey, title);
    bump();
  },

  updatePodAliasFromEvent: (podKey, alias) => {
    void applyPodAliasEvent(podKey, alias);
    bump();
  },

  updatePodAlias: async (podKey, alias) => {
    await initWasmCore();
    try {
      await updatePodAliasConnect(orgSlug(), podKey, alias);
      applyPodAliasEvent(podKey, alias);
      bump();
    } catch (error: unknown) {
      console.warn("[PodStore] updatePodAlias failed, reverting", error);
      bump();
      throw error;
    }
  },

  updatePodPerpetualFromEvent: (podKey, perpetual) => {
    const req = protoCreate(PatchPodPerpetualRequestSchema, { podKey, perpetual });
    getPodState().patch_pod_perpetual(toBinary(PatchPodPerpetualRequestSchema, req));
    bump();
  },

  updatePodPerpetual: async (podKey, perpetual) => {
    await initWasmCore();
    try {
      await updatePodPerpetualConnect(orgSlug(), podKey, perpetual);
      const req = protoCreate(PatchPodPerpetualRequestSchema, { podKey, perpetual });
      getPodState().patch_pod_perpetual(toBinary(PatchPodPerpetualRequestSchema, req));
      bump();
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update perpetual") });
      throw error;
    }
  },

  updatePodInitProgress: (podKey, phase, progress, message) => {
    set((state) => ({ initProgress: { ...state.initProgress, [podKey]: { phase, progress, message } } }));
  },

  clearInitProgress: (podKey) => {
    set((state) => {
      const { [podKey]: _removed, ...rest } = state.initProgress;
      return { initProgress: rest };
    });
  },

  clearError: () => set({ error: null }),
}));

// Helpers to encode + dispatch the realtime-event proto messages. Keep
// them as module-scoped functions so realtimePodHandlers.ts can reuse the
// same encoding without going through the zustand store API.
import {
  ApplyPodStatusEventRequestSchema, ApplyPodTitleEventRequestSchema,
  ApplyPodAliasEventRequestSchema, ApplyAgentStatusEventRequestSchema,
} from "@proto/pod_state/v1/pod_state_pb";

export function applyPodStatusEvent(
  podKey: string, status: string,
  agentStatus?: string | null, errorCode?: string | null, errorMessage?: string | null,
) {
  const req = protoCreate(ApplyPodStatusEventRequestSchema, {
    podKey, status,
    agentStatus: agentStatus ?? undefined,
    errorCode: errorCode ?? undefined,
    errorMessage: errorMessage ?? undefined,
  });
  getPodState().apply_pod_status_event(toBinary(ApplyPodStatusEventRequestSchema, req));
}

export function applyPodTitleEvent(podKey: string, title: string) {
  const req = protoCreate(ApplyPodTitleEventRequestSchema, { podKey, title });
  getPodState().apply_pod_title_event(toBinary(ApplyPodTitleEventRequestSchema, req));
}

export function applyPodAliasEvent(podKey: string, alias: string | null) {
  const req = protoCreate(ApplyPodAliasEventRequestSchema, {
    podKey, alias: alias ?? undefined,
  });
  getPodState().apply_pod_alias_event(toBinary(ApplyPodAliasEventRequestSchema, req));
}

export function applyAgentStatusEvent(podKey: string, agentStatus: string) {
  const req = protoCreate(ApplyAgentStatusEventRequestSchema, { podKey, agentStatus });
  getPodState().apply_agent_status_event(toBinary(ApplyAgentStatusEventRequestSchema, req));
}

reconnectRegistry.register({
  name: "pod:sidebar",
  fn: () => {
    const s = usePodStore.getState();
    s.fetchSidebarPods?.(s.currentSidebarFilter, { silent: true });
  },
  priority: "immediate",
});
