import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { PodSchema } from "@agentsmesh/proto/pod/v1/pod_pb";
import { podToCache } from "./pod";

// Guards the shared pod projection against proto-schema drift. The old desktop
// podToCache used stale field names (branch/initial_prompt) and dropped
// sandbox_path/started_at/finished_at — these assert the correct PodData shape.

describe("podToCache", () => {
  it("maps scalar + correctly-named fields the pod UI reads", () => {
    const c = podToCache(create(PodSchema, {
      id: 9n, podKey: "pod-x", status: "running", agentStatus: "idle",
      alias: "my-pod", title: "Fix bug", prompt: "do it",
      branchName: "feature/x", sandboxPath: "/tmp/wt", interactionMode: "acp",
      perpetual: true, restartCount: 2, startedAt: "s1", finishedAt: "f1",
      lastActivity: "a1", createdAt: "c0", errorCode: "E1", errorMessage: "boom",
      resumedByPodKey: "pod-y",
    }));
    expect(c.pod_key).toBe("pod-x");
    expect(c.status).toBe("running");
    // Field names that the stale desktop projection got wrong:
    expect(c.branch_name).toBe("feature/x");
    expect(c.prompt).toBe("do it");
    expect(c.sandbox_path).toBe("/tmp/wt");
    expect(c.started_at).toBe("s1");
    expect(c.finished_at).toBe("f1");
    expect(c.interaction_mode).toBe("acp");
    expect(c.restart_count).toBe(2);
    expect(c.resumed_by_pod_key).toBe("pod-y");
  });

  it("projects nested runner/agent/repository/ticket/loop/created_by", () => {
    const c = podToCache(create(PodSchema, {
      id: 1n, podKey: "p", status: "running",
      runner: { id: 3n, nodeId: "node-1", status: "online" },
      agent: { name: "Claude", slug: "claude-code" },
      repository: { id: 4n, name: "demo", slug: "org/demo", providerType: "github" },
      ticket: { id: 5n, slug: "DEV-5", title: "t" },
      loop: { id: 6n, name: "nightly", slug: "nightly" },
      createdBy: { id: 7n, username: "stone", name: "Stone" },
    }));
    expect(c.runner).toEqual({ id: 3, node_id: "node-1", status: "online" });
    expect(c.agent).toEqual({ name: "Claude", slug: "claude-code" });
    expect(c.repository?.provider_type).toBe("github");
    expect(c.ticket?.slug).toBe("DEV-5");
    expect(c.loop?.name).toBe("nightly");
    expect(c.created_by?.username).toBe("stone");
  });

  it("leaves absent nested objects undefined", () => {
    const c = podToCache(create(PodSchema, { id: 1n, podKey: "p", status: "running" }));
    expect(c.runner).toBeUndefined();
    expect(c.ticket).toBeUndefined();
    expect(c.loop).toBeUndefined();
  });
});
