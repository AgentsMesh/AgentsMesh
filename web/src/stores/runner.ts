import { create } from "zustand";
import { useMemo } from "react";
import type { RunnerData } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getRunnerService } from "@/lib/wasm-core";

export type RunnerStatus = "online" | "offline" | "maintenance" | "busy";
export type Runner = RunnerData;

interface RunnerState {
  _tick: number; loading: boolean; error: string | null;
  fetchRunners: (status?: RunnerStatus) => Promise<void>;
  fetchAvailableRunners: () => Promise<void>;
  fetchRunner: (id: number) => Promise<void>;
  updateRunner: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean; tags?: string[] }) => Promise<Runner>;
  deleteRunner: (id: number) => Promise<void>;
  createToken: (data?: { name?: string; labels?: string[]; max_uses?: number; expires_in_days?: number }) => Promise<string>;
  setCurrentRunner: (runner: Runner | null) => void;
  updateRunnerStatus: (runnerId: number, status: RunnerStatus) => void;
  clearError: () => void;
}

const svc = () => getRunnerService();
const bump = () => useRunnerStore.setState((s) => ({ _tick: s._tick + 1 }));

export function useRunners(): Runner[] {
  const tick = useRunnerStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => JSON.parse(svc().runners_json()), [tick]);
}

export function useAvailableRunners(): Runner[] {
  const tick = useRunnerStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => JSON.parse(svc().available_runners_json()), [tick]);
}

export function useCurrentRunner(): Runner | null {
  const tick = useRunnerStore((s) => s._tick);
  return useMemo(() => {
    const raw = svc().current_runner_json();
    return raw ? (typeof raw === "string" ? JSON.parse(raw) : raw) : null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
}

export const useRunnerStore = create<RunnerState>((set, get) => ({
  _tick: 0, loading: false, error: null,

  fetchRunners: async (status) => {
    set({ loading: true, error: null });
    try {
      await svc().fetch_runners(status ?? null);
      set({ loading: false, _tick: get()._tick + 1 });
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch runners"), loading: false }); }
  },

  fetchAvailableRunners: async () => {
    try {
      await svc().fetch_available_runners();
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch available runners") }); }
  },

  fetchRunner: async (id) => {
    try {
      await svc().fetch_runner(BigInt(id));
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch runner") }); }
  },

  updateRunner: async (id, data) => {
    try {
      const json = await svc().update_runner(BigInt(id), JSON.stringify(data));
      const runner: Runner = JSON.parse(json);
      bump();
      return runner;
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update runner") }); throw e; }
  },

  deleteRunner: async (id) => {
    try {
      await svc().delete_runner(BigInt(id));
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to delete runner") }); throw e; }
  },

  createToken: async (data) => {
    try {
      const json = await svc().create_token(JSON.stringify(data || {}));
      const resp: { token: string } = JSON.parse(json);
      return resp.token;
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create token") }); throw e; }
  },

  setCurrentRunner: (runner) => {
    svc().set_current_runner(runner ? JSON.stringify(runner) : "");
    bump();
  },

  updateRunnerStatus: (runnerId, status) => {
    svc().update_runner_status(BigInt(runnerId), status);
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

export const canAcceptPods = (runner: Runner): boolean =>
  runner.status === "online" && runner.current_pods < runner.max_concurrent_pods;

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
