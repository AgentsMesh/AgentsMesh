import { vi } from "vitest";
import { fromBinary } from "@bufbuild/protobuf";
import {
  InsertCreatedPodRequestSchema,
  ReplaceCachedPodsRequestSchema,
  AppendCachedPodsRequestSchema,
  PatchPodPerpetualRequestSchema,
  MarkPodTerminatedRequestSchema,
} from "@proto/pod_state/v1/pod_state_pb";
import { usePodStore, usePods, useCurrentPod, Pod } from "../pod";
import { getPodState } from "@/lib/wasm-core";
import { fromProtoPod } from "@/lib/api/facade/podConnect";

export { usePods, useCurrentPod };

export const mockPod: Pod = {
  id: 1,
  pod_key: "pod-abc-123",
  status: "running",
  agent_status: "executing",
  created_at: "2024-01-01T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

export const mockPod2: Pod = {
  id: 2,
  pod_key: "pod-def-456",
  status: "running",
  agent_status: "waiting",
  created_at: "2024-01-02T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

// pods_json / current_pod_json live on `WasmPodState` (cache), not on
// `WasmPodService` (network). The mock at src/test/setup.ts puts the
// JSON reads on the `podState` object; proto-bytes mutators are no-ops
// in the mock (the mock's comment recommends overriding `pods_json` /
// `get_pod_json` directly when the test needs cache contents).
export function readPods(): Pod[] {
  return JSON.parse(getPodState().pods_json()) as Pod[];
}

export function readCurrentPod(): Pod | null {
  const json = getPodState().current_pod_json();
  if (!json) return null;
  return JSON.parse(json as string) as Pod;
}

interface PodStateMock {
  pods_json: ReturnType<typeof vi.fn> & ((..._args: unknown[]) => string);
  current_pod_json: ReturnType<typeof vi.fn> & ((..._args: unknown[]) => string | undefined);
  get_pod_json: ReturnType<typeof vi.fn>;
  insert_created_pod: ReturnType<typeof vi.fn>;
  patch_pod_perpetual: ReturnType<typeof vi.fn>;
  apply_pod_status_event: ReturnType<typeof vi.fn>;
  apply_pod_title_event: ReturnType<typeof vi.fn>;
  apply_pod_alias_event: ReturnType<typeof vi.fn>;
  apply_agent_status_event: ReturnType<typeof vi.fn>;
  replace_cached_pods: ReturnType<typeof vi.fn>;
  append_cached_pods: ReturnType<typeof vi.fn>;
  mark_pod_terminated: ReturnType<typeof vi.fn>;
}

export function podStateMock(): PodStateMock {
  return getPodState() as unknown as PodStateMock;
}

export function resetPodStore() {
  vi.clearAllMocks();
  // After clearAllMocks: re-prime read-side defaults so tests that don't
  // explicitly seed start with empty cache and a working get-by-key.
  podStateMock().pods_json.mockReturnValue("[]");
  podStateMock().current_pod_json.mockReturnValue(undefined);
  podStateMock().get_pod_json.mockImplementation((key: string) => {
    const list = JSON.parse(podStateMock().pods_json()) as { pod_key: string }[];
    const p = list.find((x) => x.pod_key === key);
    return p ? JSON.stringify(p) : undefined;
  });
  usePodStore.setState({
    _tick: 0,
    loading: false,
    error: null,
    initProgress: {},
    podTotal: 0,
    podHasMore: false,
    loadingMore: false,
    currentSidebarFilter: "mine",
    sidebarLoadedCount: 0,
  });
}

// Pre-populate the JSON-read backing fixture. Proto-bytes mutators are
// opaque no-ops, so tests that need readable cache state seed it here.
export function seedPods(...pods: Pod[]) {
  podStateMock().pods_json.mockReturnValue(JSON.stringify(pods));
}

export function seedPodsWithCurrent(current: Pod, ...extra: Pod[]) {
  seedPods(current, ...extra);
  podStateMock().current_pod_json.mockReturnValue(JSON.stringify(current));
}

// Helpers — decode the last proto-bytes call into the snake_case Pod
// shape the store consumes. Tests assert on this instead of the opaque
// Uint8Array bytes.
export function lastReplaceCachedPods(): Pod[] {
  const calls = podStateMock().replace_cached_pods.mock.calls;
  if (calls.length === 0) return [];
  const req = fromBinary(ReplaceCachedPodsRequestSchema, calls[calls.length - 1][0] as Uint8Array);
  return req.pods.map(fromProtoPod);
}

export function lastAppendCachedPods(): Pod[] {
  const calls = podStateMock().append_cached_pods.mock.calls;
  if (calls.length === 0) return [];
  const req = fromBinary(AppendCachedPodsRequestSchema, calls[calls.length - 1][0] as Uint8Array);
  return req.pods.map(fromProtoPod);
}

export function lastInsertCreatedPod(): Pod | null {
  const calls = podStateMock().insert_created_pod.mock.calls;
  if (calls.length === 0) return null;
  const req = fromBinary(InsertCreatedPodRequestSchema, calls[calls.length - 1][0] as Uint8Array);
  return req.pod ? fromProtoPod(req.pod) : null;
}

export function lastPatchPodPerpetual(): { pod_key: string; perpetual: boolean } | null {
  const calls = podStateMock().patch_pod_perpetual.mock.calls;
  if (calls.length === 0) return null;
  const req = fromBinary(PatchPodPerpetualRequestSchema, calls[calls.length - 1][0] as Uint8Array);
  return { pod_key: req.podKey, perpetual: req.perpetual };
}

export function lastMarkPodTerminated(): { pod_key: string } | null {
  const calls = podStateMock().mark_pod_terminated.mock.calls;
  if (calls.length === 0) return null;
  const req = fromBinary(MarkPodTerminatedRequestSchema, calls[calls.length - 1][0] as Uint8Array);
  return { pod_key: req.podKey };
}
