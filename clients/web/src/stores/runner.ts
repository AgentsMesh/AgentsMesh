import { create } from "zustand";
import { useMemo } from "react";
import { create as protoCreate, toBinary, fromBinary } from "@bufbuild/protobuf";
import type { RunnerData } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getRunnerState } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import {
  listRunnersRaw,
  listAvailableRunnersRaw,
  getRunnerRaw,
  updateRunner as updateRunnerConnect,
  deleteRunner as deleteRunnerConnect,
  createRunnerToken as createRunnerTokenConnect,
} from "@/lib/api/facade/runnerConnect";
import {
  ReplaceCachedRunnersRequestSchema,
  ReplaceAvailableRunnersRequestSchema,
  SetCurrentRunnerRequestSchema,
  PatchCachedRunnerRequestSchema,
  RemoveCachedRunnerRequestSchema,
} from "@proto/runner_state/v1/runner_state_pb";
import { RunnerSchema } from "@proto/runner_api/v1/runner_pb";
import { runnerToProtoRunner } from "@/lib/api/runnerProtoMap";
import { runnerToCache } from "@agentsmesh/electron-adapter/projections";

export type RunnerStatus = "online" | "offline" | "maintenance" | "busy";
export type Runner = RunnerData;

interface RunnerState {
  _tick: number; loading: boolean; fetched: boolean; error: string | null;
  fetchRunners: (status?: RunnerStatus) => Promise<void>;
  fetchAvailableRunners: () => Promise<void>;
  fetchRunner: (id: number) => Promise<void>;
  updateRunner: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean; tags?: string[] }) => Promise<Runner>;
  deleteRunner: (id: number) => Promise<void>;
  createToken: (data?: { name?: string; labels?: string[]; max_uses?: number; expires_in_days?: number }) => Promise<string>;
  setCurrentRunner: (runner: Runner | null) => void;
  clearError: () => void;
}

// Runner state SSOT is the shared AppState (runtime.state) via getRunnerState
// (web: WasmRunnerState; desktop: ElectronRunnerService). This is the SAME
// state the EventBus dispatch + desktop snapshot mirror write, so realtime
// runner changes flow without a JS pure-patch. Connect-RPC stays on the
// runnerConnect facade (which still uses getRunnerService).
const svc = () => getRunnerState();
const bump = () => useRunnerStore.setState((s) => ({ _tick: s._tick + 1 }));

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

function dispatchSetCurrentRunner(runner: Runner | null) {
  const req = protoCreate(SetCurrentRunnerRequestSchema, {
    runner: runner ? runnerToProtoRunner(runner) : undefined,
  });
  svc().set_current_runner_proto(toBinary(SetCurrentRunnerRequestSchema, req));
}

function dispatchPatchCachedRunner(runner: Runner) {
  const req = protoCreate(PatchCachedRunnerRequestSchema, {
    runner: runnerToProtoRunner(runner),
  });
  svc().patch_cached_runner(toBinary(PatchCachedRunnerRequestSchema, req));
}

function dispatchRemoveCachedRunner(id: number) {
  const req = protoCreate(RemoveCachedRunnerRequestSchema, {
    runnerId: BigInt(id),
  });
  svc().remove_cached_runner(toBinary(RemoveCachedRunnerRequestSchema, req));
}

// Read side (B, zero-JSON): decode state proto bytes via fromBinary +
// runnerToCache (shared projection). UI is a projection of state proto bytes.
export function useRunners(): Runner[] {
  const tick = useRunnerStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(
    () => fromBinary(ReplaceCachedRunnersRequestSchema, svc().runners_bytes()).runners.map(runnerToCache) as Runner[],
    [tick],
  );
}

export function useAvailableRunners(): Runner[] {
  const tick = useRunnerStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(
    () => fromBinary(ReplaceAvailableRunnersRequestSchema, svc().available_runners_bytes()).runners.map(runnerToCache) as Runner[],
    [tick],
  );
}

export function useCurrentRunner(): Runner | null {
  const tick = useRunnerStore((s) => s._tick);
  return useMemo(() => {
    const bytes = svc().current_runner_bytes();
    if (bytes.length === 0) return null;
    return runnerToCache(fromBinary(RunnerSchema, bytes)) as Runner;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
}

export const useRunnerStore = create<RunnerState>((set, get) => ({
  _tick: 0, loading: false, fetched: false, error: null,

  fetchRunners: async (status) => {
    set({ loading: true, error: null });
    try {
      const respBytes = await listRunnersRaw(orgSlug(), { status });
      svc().apply_fetched_runners(respBytes);
      set({ loading: false, fetched: true, _tick: get()._tick + 1 });
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch runners"), loading: false }); }
  },

  fetchAvailableRunners: async () => {
    try {
      const respBytes = await listAvailableRunnersRaw(orgSlug());
      svc().apply_fetched_available_runners(respBytes);
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch available runners") }); }
  },

  fetchRunner: async (id) => {
    try {
      const respBytes = await getRunnerRaw(orgSlug(), id);
      svc().apply_fetched_current_runner(respBytes);
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch runner") }); }
  },

  updateRunner: async (id, data) => {
    try {
      const runner = await updateRunnerConnect(orgSlug(), id, data);
      dispatchPatchCachedRunner(runner);
      bump();
      return runner;
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update runner") }); throw e; }
  },

  deleteRunner: async (id) => {
    try {
      await deleteRunnerConnect(orgSlug(), id);
      dispatchRemoveCachedRunner(id);
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete runner") }); throw e; }
  },

  createToken: async (data) => {
    try {
      const resp = await createRunnerTokenConnect(orgSlug(), data ?? {});
      return resp.token ?? "";
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create token") }); throw e; }
  },

  setCurrentRunner: (runner) => {
    dispatchSetCurrentRunner(runner);
    bump();
  },

  clearError: () => set({ error: null }),
}));

export const getRunnerStatusInfo = (status: RunnerStatus) => {
  const m: Record<RunnerStatus, { label: string; color: string; dotColor: string }> = {
    online: { label: "Online", color: "text-green-600 dark:text-green-400", dotColor: "bg-green-500" },
    offline: { label: "Offline", color: "text-gray-500 dark:text-gray-400", dotColor: "bg-gray-400" },
    maintenance: { label: "Maintenance", color: "text-yellow-600 dark:text-yellow-400", dotColor: "bg-yellow-500" },
    busy: { label: "Busy", color: "text-orange-600 dark:text-orange-400", dotColor: "bg-orange-500" },
  };
  return m[status];
};

export const formatHostInfo = (hostInfo?: Runner["host_info"]) => {
  if (!hostInfo) return "Unknown";
  const parts: string[] = [];
  if (hostInfo.os) parts.push(hostInfo.os);
  if (hostInfo.arch) parts.push(hostInfo.arch);
  if (hostInfo.cpu_cores) parts.push(`${hostInfo.cpu_cores} cores`);
  if (hostInfo.memory) parts.push(`${(hostInfo.memory / 1024 / 1024 / 1024).toFixed(1)}GB RAM`);
  return parts.length > 0 ? parts.join(" / ") : "Unknown";
};

reconnectRegistry.register({
  name: "runner:list",
  fn: () => useRunnerStore.getState().fetchRunners?.(),
  priority: "immediate",
});
