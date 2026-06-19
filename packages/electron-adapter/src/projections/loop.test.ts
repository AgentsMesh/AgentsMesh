import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { LoopSchema, LoopRunSchema } from "@agentsmesh/proto/loop/v1/loop_pb";
import { loopToCache, loopRunToCache } from "./loop";

// Guards the shared loop projection against proto-schema drift. The desktop
// loop-detail crash (undefined.toString() in ConfigPanel) was a projection
// built on a dead proto schema that silently dropped max_concurrent_runs and
// ~20 other fields. These assert every field the UI reads survives.

describe("loopToCache", () => {
  it("maps every scalar field the loop UI reads", () => {
    const c = loopToCache(create(LoopSchema, {
      id: 7n, slug: "nightly-audit", name: "Nightly Audit",
      description: "desc", agentSlug: "claude-code", permissionMode: "bypassPermissions",
      promptTemplate: "Audit deps", executionMode: "autopilot", status: "enabled",
      sandboxStrategy: "persistent", sessionPersistence: true, concurrencyPolicy: "skip",
      maxConcurrentRuns: 5, maxRetainedRuns: 3, timeoutMinutes: 60,
      cronExpression: "0 0 * * *", createdAt: "2026-01-01", updatedAt: "2026-01-02",
    }));

    // The crash field — ConfigPanel calls .toString() on it.
    expect(c.max_concurrent_runs).toBe(5);
    expect(c.max_retained_runs).toBe(3);
    expect(c.timeout_minutes).toBe(60);
    expect(c.execution_mode).toBe("autopilot");
    expect(c.sandbox_strategy).toBe("persistent");
    expect(c.session_persistence).toBe(true);
    expect(c.concurrency_policy).toBe("skip");
    expect(c.prompt_template).toBe("Audit deps");
    expect(c.cron_expression).toBe("0 0 * * *");
    expect(c.status).toBe("enabled");
    expect(c.agent_slug).toBe("claude-code");
    expect(c.permission_mode).toBe("bypassPermissions");
  });

  it("converts bigint counters to number", () => {
    const c = loopToCache(create(LoopSchema, {
      id: 42n, totalRuns: 100n, successfulRuns: 90n, failedRuns: 10n, activeRunCount: 3n,
    }));
    expect(c.id).toBe(42);
    expect(typeof c.id).toBe("number");
    expect(c.total_runs).toBe(100);
    expect(c.successful_runs).toBe(90);
    expect(c.failed_runs).toBe(10);
    expect(c.active_run_count).toBe(3);
  });

  it("parses JSON string fields into objects", () => {
    const c = loopToCache(create(LoopSchema, {
      configOverridesJson: '{"model":"opus"}',
      promptVariablesJson: '{"repo":"demo"}',
      autopilotConfigJson: '{"max_iterations":5}',
    }));
    expect(c.config_overrides).toEqual({ model: "opus" });
    expect(c.prompt_variables).toEqual({ repo: "demo" });
    expect(c.autopilot_config).toEqual({ max_iterations: 5 });
  });

  it("defaults empty JSON / array fields safely", () => {
    const c = loopToCache(create(LoopSchema, {}));
    expect(c.config_overrides).toBeUndefined();
    expect(c.prompt_variables).toBeUndefined();
    expect(c.autopilot_config).toEqual({});
    expect(c.used_env_bundles).toEqual([]);
  });

  it("normalizes empty optional strings to undefined", () => {
    const c = loopToCache(create(LoopSchema, { description: "", cronExpression: "" }));
    expect(c.description).toBeUndefined();
    expect(c.cron_expression).toBeUndefined();
  });
});

describe("loopRunToCache", () => {
  it("maps run fields and flips completedAt → finished_at", () => {
    const c = loopRunToCache(create(LoopRunSchema, {
      id: 1n, loopId: 7n, runNumber: 3n, status: "completed",
      podKey: "pod-x", startedAt: "t1", completedAt: "t2", createdAt: "t0",
    }));
    expect(c.id).toBe(1);
    expect(c.loop_id).toBe(7);
    expect(c.run_number).toBe(3);
    expect(c.status).toBe("completed");
    expect(c.pod_key).toBe("pod-x");
    expect(c.started_at).toBe("t1");
    expect(c.finished_at).toBe("t2");
  });
});
