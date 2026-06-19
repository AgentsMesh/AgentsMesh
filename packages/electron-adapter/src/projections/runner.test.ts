import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { RunnerSchema } from "@agentsmesh/proto/runner_api/v1/runner_pb";
import { runnerToCache } from "./runner";

// Guards the shared runner projection against proto-schema drift.

describe("runnerToCache", () => {
  it("maps every scalar field the runner UI reads", () => {
    const c = runnerToCache(create(RunnerSchema, {
      id: 9n, nodeId: "node-1", description: "dev box", status: "online",
      lastHeartbeat: "hb1", currentPods: 2, maxConcurrentPods: 8,
      runnerVersion: "1.2.3", isEnabled: true, visibility: "organization",
      registeredByUserId: 7n, availableAgents: ["claude-code"], tags: ["mac"],
      createdAt: "c0", updatedAt: "u0",
    }));
    expect(c.id).toBe(9);
    expect(c.node_id).toBe("node-1");
    expect(c.status).toBe("online");
    expect(c.current_pods).toBe(2);
    expect(c.max_concurrent_pods).toBe(8);
    expect(c.runner_version).toBe("1.2.3");
    expect(c.is_enabled).toBe(true);
    expect(c.visibility).toBe("organization");
    expect(c.registered_by_user_id).toBe(7);
    expect(c.available_agents).toEqual(["claude-code"]);
    expect(c.tags).toEqual(["mac"]);
  });

  it("parses host_info JSON and normalizes empty lists/strings", () => {
    const c = runnerToCache(create(RunnerSchema, {
      id: 1n, nodeId: "n", status: "offline",
      hostInfoJson: '{"os":"darwin","arch":"arm64","cpu_cores":10}',
      availableAgents: [], tags: [], description: "",
    }));
    expect(c.host_info).toEqual({ os: "darwin", arch: "arm64", cpu_cores: 10 });
    expect(c.available_agents).toBeUndefined();
    expect(c.tags).toBeUndefined();
    expect(c.description).toBeUndefined();
  });

  it("defaults malformed host_info to undefined", () => {
    const c = runnerToCache(create(RunnerSchema, {
      id: 1n, nodeId: "n", status: "offline", hostInfoJson: "{not json",
    }));
    expect(c.host_info).toBeUndefined();
  });
});
