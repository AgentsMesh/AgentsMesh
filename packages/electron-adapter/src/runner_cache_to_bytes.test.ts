import { describe, it, expect, beforeEach } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  RunnerSchema, ListRunnersResponseSchema, ListAvailableRunnersResponseSchema,
} from "@agentsmesh/proto/runner_api/v1/runner_pb";
import {
  ReplaceCachedRunnersRequestSchema, ReplaceAvailableRunnersRequestSchema,
} from "@agentsmesh/proto/runner_state/v1/runner_state_pb";
import { runnerToCache } from "./projections/runner";
import { runnersBytes, availableRunnersBytes, currentRunnerBytes } from "./runner_cache_to_bytes";
import { ElectronRunnerService } from "./runner";

const wireRunner = (id: bigint, nodeId: string) => create(RunnerSchema, {
  id, nodeId, status: "online", isEnabled: true, currentPods: 1, maxConcurrentPods: 5,
  lastHeartbeat: "2026-01-01", runnerVersion: "1.2.3", description: "d",
  hostInfoJson: '{"os":"linux"}', availableAgents: ["claude"], tags: ["t1"],
  createdAt: "2026-01-01", updatedAt: "2026-01-02",
});

// cache→bytes must round-trip every field runnerToCache reads, or desktop
// diverges from web (which decodes the same bytes through runnerToCache).
describe("runner cache→bytes round-trip", () => {
  it("preserves runner fields through cache → bytes → state", () => {
    const cache = runnerToCache(wireRunner(1n, "node-a"));
    const decoded = fromBinary(ReplaceCachedRunnersRequestSchema, runnersBytes(JSON.stringify([cache])));
    const back = runnerToCache(decoded.runners[0]);
    expect(back).toEqual(cache);
  });

  it("preserves available + current", () => {
    const cache = runnerToCache(wireRunner(2n, "node-b"));
    expect(fromBinary(ReplaceAvailableRunnersRequestSchema, availableRunnersBytes(JSON.stringify([cache]))).runners[0].nodeId).toBe("node-b");
    expect(runnerToCache(fromBinary(RunnerSchema, currentRunnerBytes(JSON.stringify(cache)))).id).toBe(2);
    expect(currentRunnerBytes(null).length).toBe(0);
  });
});

describe("ElectronRunnerService fetch→state", () => {
  let invokes: string[];
  beforeEach(() => {
    invokes = [];
    (globalThis as { window?: unknown }).window = {
      electronAPI: { invoke: async (ch: string) => { invokes.push(ch); return undefined; } },
    };
  });

  it("apply_fetched_runners caches + fans to main + reads back via bytes", () => {
    const svc = new ElectronRunnerService();
    const bytes = toBinary(ListRunnersResponseSchema, create(ListRunnersResponseSchema, {
      items: [wireRunner(1n, "node-a"), wireRunner(2n, "node-b")],
    }));
    svc.apply_fetched_runners(bytes);
    expect(invokes).toContain("appRunnerApplyFetched");
    const decoded = fromBinary(ReplaceCachedRunnersRequestSchema, svc.runners_bytes());
    expect(decoded.runners.map((r) => r.nodeId)).toEqual(["node-a", "node-b"]);
  });

  it("apply_fetched_available_runners populates the available cache", () => {
    const svc = new ElectronRunnerService();
    const bytes = toBinary(ListAvailableRunnersResponseSchema, create(ListAvailableRunnersResponseSchema, {
      items: [wireRunner(3n, "node-c")],
    }));
    svc.apply_fetched_available_runners(bytes);
    expect(invokes).toContain("appRunnerApplyFetchedAvailable");
    expect(fromBinary(ReplaceAvailableRunnersRequestSchema, svc.available_runners_bytes()).runners[0].nodeId).toBe("node-c");
  });
});
