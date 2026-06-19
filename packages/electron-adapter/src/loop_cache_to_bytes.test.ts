import { describe, it, expect } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  LoopSchema, LoopRunSchema, ListLoopsResponseSchema, ListRunsResponseSchema,
} from "@agentsmesh/proto/loop/v1/loop_pb";
import {
  ReplaceCachedLoopsRequestSchema, ReplaceCachedRunsRequestSchema,
} from "@agentsmesh/proto/loop_state/v1/loop_state_pb";
import { loopToCache, loopRunToCache } from "./projections/loop";
import { loopsBytes, runsBytes, currentLoopBytes } from "./loop_cache_to_bytes";
import { ElectronLoopService } from "./loop";

const wireLoop = (id: bigint, slug: string) => create(LoopSchema, {
  id, slug, name: "n", description: "d", agentSlug: "claude", permissionMode: "default",
  promptTemplate: "p", configOverridesJson: '{"a":1}', promptVariablesJson: '{"b":2}',
  executionMode: "direct", cronExpression: "* * * * *", autopilotConfigJson: "{}",
  callbackUrl: "http://x", repositoryId: 5n, runnerId: 6n, branchName: "main", ticketId: 7n,
  credentialProfileId: 8n, status: "enabled", sandboxStrategy: "fresh", sessionPersistence: true,
  concurrencyPolicy: "queue", maxConcurrentRuns: 2, maxRetainedRuns: 10, timeoutMinutes: 30,
  totalRuns: 3n, successfulRuns: 2n, failedRuns: 1n, activeRunCount: 0n, avgDurationSec: 4.5,
  lastRunAt: "2026-01-02", createdAt: "2026-01-01", updatedAt: "2026-01-03", usedEnvBundles: ["e1"],
});
const wireRun = (id: bigint) => create(LoopRunSchema, {
  id, loopId: 1n, runNumber: 4n, status: "completed", podKey: "pk", startedAt: "2026-01-01",
  completedAt: "2026-01-02", errorMessage: "", createdAt: "2026-01-01",
});

// cache→bytes must round-trip every field loopToCache reads, or desktop diverges
// from web (which decodes the same bytes through loopToCache).
describe("loop cache→bytes round-trip", () => {
  it("preserves loop fields through cache → bytes → state", () => {
    const cache = loopToCache(wireLoop(1n, "L-1"));
    const decoded = fromBinary(ReplaceCachedLoopsRequestSchema, loopsBytes(JSON.stringify([cache])));
    expect(loopToCache(decoded.loops[0])).toEqual(cache);
  });

  it("preserves run fields + current loop", () => {
    const run = loopRunToCache(wireRun(2n));
    expect(loopRunToCache(fromBinary(ReplaceCachedRunsRequestSchema, runsBytes(JSON.stringify([run]))).runs[0])).toEqual(run);
    const cache = loopToCache(wireLoop(3n, "L-3"));
    expect(currentLoopBytes(JSON.stringify(cache)).length).toBeGreaterThan(0);
    expect(currentLoopBytes(null).length).toBe(0);
  });
});

describe("ElectronLoopService fetch→state", () => {
  it("apply_fetched_loops/runs cache + read back via bytes", () => {
    const svc = new ElectronLoopService();
    svc.apply_fetched_loops(toBinary(ListLoopsResponseSchema,
      create(ListLoopsResponseSchema, { items: [wireLoop(1n, "L-1"), wireLoop(2n, "L-2")] })));
    expect(fromBinary(ReplaceCachedLoopsRequestSchema, svc.loops_bytes()).loops.map((l) => l.slug)).toEqual(["L-1", "L-2"]);

    svc.apply_fetched_runs(toBinary(ListRunsResponseSchema, create(ListRunsResponseSchema, { items: [wireRun(1n)] })));
    svc.apply_appended_runs(toBinary(ListRunsResponseSchema, create(ListRunsResponseSchema, { items: [wireRun(2n)] })));
    expect(fromBinary(ReplaceCachedRunsRequestSchema, svc.runs_bytes()).runs.map((r) => Number(r.id))).toEqual([1, 2]);
  });
});
