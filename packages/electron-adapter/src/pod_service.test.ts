import { describe, it, expect, beforeEach } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { PodSchema, ListPodsResponseSchema } from "@agentsmesh/proto/pod/v1/pod_pb";
import { ReplaceCachedPodsRequestSchema } from "@agentsmesh/proto/pod_state/v1/pod_state_pb";
import { ElectronPodService } from "./pod";

// Desktop fetch→state runtime contract: the shared store calls apply_fetched_pods
// with wire bytes, and reads back via *_bytes. This verifies (1) the renderer
// cache updates synchronously, (2) the SAME wire bytes are fanned out to main
// (runtime.state SSOT), (3) the read path re-encodes the cache into state proto
// the web selectors decode — all WITHOUT a backend. The wire Pod IS the cache
// Pod, so the round-trip is identity.
describe("ElectronPodService fetch→state", () => {
  let invokes: Array<{ channel: string; args: unknown[] }>;

  beforeEach(() => {
    invokes = [];
    (globalThis as { window?: unknown }).window = {
      electronAPI: {
        invoke: async (channel: string, ...args: unknown[]) => {
          invokes.push({ channel, args });
          return undefined;
        },
      },
    };
  });

  const wirePodsBytes = (...keys: string[]) =>
    toBinary(ListPodsResponseSchema, create(ListPodsResponseSchema, {
      items: keys.map((k, i) => create(PodSchema, {
        id: BigInt(i + 1), podKey: k, status: "running", agentStatus: "executing",
        createdAt: "2026-01-01",
      })),
      total: BigInt(keys.length),
    }));

  it("apply_fetched_pods updates cache + fans wire bytes to main", () => {
    const svc = new ElectronPodService();
    const bytes = wirePodsBytes("pk-a", "pk-b");
    svc.apply_fetched_pods(bytes);

    const cached = JSON.parse(svc.pods_json()) as Array<{ pod_key: string }>;
    expect(cached.map((p) => p.pod_key)).toEqual(["pk-a", "pk-b"]);

    const fan = invokes.find((i) => i.channel === "appPodApplyFetchedPods");
    expect(fan).toBeDefined();
    expect(Array.from(fan!.args[0] as number[])).toEqual(Array.from(bytes));

    const decoded = fromBinary(ReplaceCachedPodsRequestSchema, svc.pods_bytes());
    expect(decoded.pods.map((p) => p.podKey)).toEqual(["pk-a", "pk-b"]);
    expect(decoded.pods[0].agentStatus).toBe("executing");
  });

  it("apply_appended_pods dedups against existing cache + fans to main", () => {
    const svc = new ElectronPodService();
    svc.apply_fetched_pods(wirePodsBytes("pk-a", "pk-b"));
    // Append a page that re-includes pk-b — must not duplicate.
    svc.apply_appended_pods(wirePodsBytes("pk-b", "pk-c"));

    const cached = JSON.parse(svc.pods_json()) as Array<{ pod_key: string }>;
    expect(cached.map((p) => p.pod_key)).toEqual(["pk-a", "pk-b", "pk-c"]);
    expect(invokes.some((i) => i.channel === "appPodApplyAppendedPods")).toBe(true);
  });

  it("get_pod_bytes / current_pod_bytes read back the cached pod", () => {
    const svc = new ElectronPodService();
    svc.apply_fetched_pods(wirePodsBytes("pk-a"));
    expect(fromBinary(PodSchema, svc.get_pod_bytes("pk-a")).podKey).toBe("pk-a");
    expect(svc.get_pod_bytes("missing").length).toBe(0);
    expect(svc.current_pod_bytes().length).toBe(0);
  });
});
