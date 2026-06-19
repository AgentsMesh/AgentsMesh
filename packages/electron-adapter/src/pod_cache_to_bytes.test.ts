import { describe, it, expect } from "vitest";
import { create, fromBinary } from "@bufbuild/protobuf";
import {
  PodSchema, PodRunnerInfoSchema, PodRepositoryInfoSchema, PodTicketInfoSchema,
  PodLoopInfoSchema, PodCreatedByInfoSchema, PodAgentInfoSchema,
} from "@agentsmesh/proto/pod/v1/pod_pb";
import { ReplaceCachedPodsRequestSchema } from "@agentsmesh/proto/pod_state/v1/pod_state_pb";
import { podToCache } from "./projections/pod";
import { podsBytes, podBytes, currentPodBytes } from "./pod_cache_to_bytes";

// The desktop renderer round-trips fetched wire pods through a snake_case cache
// and back into state proto bytes. The shared web selectors then decode those
// bytes — so this round-trip MUST preserve every field podToCache reads, or
// desktop silently diverges from web. wire Pod → cache → bytes → state Pod.
describe("pod cache→bytes round-trip", () => {
  const fullWire = () =>
    create(PodSchema, {
      id: 5n, podKey: "pk-1", status: "running", agentStatus: "executing",
      alias: "my-pod", title: "Task", prompt: "do it", branchName: "feat/x",
      sandboxPath: "/tmp/x", interactionMode: "pty", perpetual: true,
      restartCount: 2, lastRestartAt: "2026-01-03", startedAt: "2026-01-01",
      finishedAt: "2026-01-02", lastActivity: "2026-01-04", createdAt: "2026-01-01",
      errorCode: "E1", errorMessage: "boom", resumedByPodKey: "pk-2",
      runner: create(PodRunnerInfoSchema, { id: 9n, nodeId: "node-9", status: "online" }),
      agent: create(PodAgentInfoSchema, { name: "Claude", slug: "claude" }),
      repository: create(PodRepositoryInfoSchema, { id: 3n, name: "repo", slug: "org/repo", providerType: "github" }),
      ticket: create(PodTicketInfoSchema, { id: 7n, slug: "tk-7", title: "Ticket" }),
      loop: create(PodLoopInfoSchema, { id: 4n, name: "Loop", slug: "loop-4" }),
      createdBy: create(PodCreatedByInfoSchema, { id: 2n, username: "alice", name: "Alice" }),
    });

  it("preserves all scalar + nested fields via ReplaceCachedPodsRequest", () => {
    const cacheJson = JSON.stringify([podToCache(fullWire())]);
    const decoded = fromBinary(ReplaceCachedPodsRequestSchema, podsBytes(cacheJson));
    const p = decoded.pods[0];
    expect(p.id).toBe(5n);
    expect(p.podKey).toBe("pk-1");
    expect(p.status).toBe("running");
    expect(p.agentStatus).toBe("executing");
    expect(p.alias).toBe("my-pod");
    expect(p.perpetual).toBe(true);
    expect(p.restartCount).toBe(2);
    expect(p.interactionMode).toBe("pty");
    expect(p.resumedByPodKey).toBe("pk-2");
    expect(p.runner?.nodeId).toBe("node-9");
    expect(p.runner?.id).toBe(9n);
    expect(p.agent?.slug).toBe("claude");
    expect(p.repository?.slug).toBe("org/repo");
    expect(p.repository?.providerType).toBe("github");
    expect(p.ticket?.slug).toBe("tk-7");
    expect(p.loop?.slug).toBe("loop-4");
    expect(p.createdBy?.username).toBe("alice");
  });

  it("preserves single pod via get/current bytes; empty when absent", () => {
    const cacheJson = JSON.stringify([podToCache(fullWire())]);
    expect(fromBinary(PodSchema, podBytes(cacheJson, "pk-1")).alias).toBe("my-pod");
    expect(fromBinary(PodSchema, currentPodBytes(JSON.stringify(podToCache(fullWire())))).podKey).toBe("pk-1");
    expect(podBytes(cacheJson, "missing").length).toBe(0);
    expect(currentPodBytes(null).length).toBe(0);
  });
});
