import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  useChannelPods,
  invalidateChannelPods,
  __resetChannelPodsCacheForTests,
} from "../useChannelPods";

// Stateful mock of ChannelService for pod APIs. Rust is SSOT, so the test
// mock stores pods keyed by channel id — get_channel_pods writes, channel_pods_json reads.
const podsByChannel = new Map<number, { pod_key: string; status?: string; alias?: string }[]>();
let pendingFetch: ((_: void) => void) | null = null;

vi.mock("@/lib/api/channel", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api/channel")>("@/lib/api/channel");
  return {
    ...actual,
    channelApi: {
      ...actual.channelApi,
      getPods: vi.fn(async (id: number) => {
        if (pendingFetch) {
          await new Promise<void>((resolve) => {
            pendingFetch = resolve;
          });
        }
        return { pods: (podsByChannel.get(id) ?? []) as never, total: 0 };
      }),
    },
  };
});

vi.mock("@/lib/wasm-core", async () => {
  const actual = await vi.importActual<typeof import("@/lib/wasm-core")>("@/lib/wasm-core");
  return {
    ...actual,
    getChannelService: () => ({
      get_channel_pods: async (id: bigint) => {
        return JSON.stringify({ pods: podsByChannel.get(Number(id)) ?? [] });
      },
      channel_pods_json: (id: bigint) => JSON.stringify(podsByChannel.get(Number(id)) ?? []),
    }),
  };
});

import { channelApi } from "@/lib/api/channel";

function seed(channelId: number, pods: { pod_key: string; status?: string; alias?: string }[]) {
  podsByChannel.set(channelId, pods);
}

describe("useChannelPods", () => {
  beforeEach(() => {
    podsByChannel.clear();
    pendingFetch = null;
    __resetChannelPodsCacheForTests();
    vi.mocked(channelApi.getPods).mockClear();
  });

  afterEach(() => {
    __resetChannelPodsCacheForTests();
    podsByChannel.clear();
  });

  it("fetches once and shares the result across subscribers", async () => {
    seed(1, [{ pod_key: "a", status: "running" }]);

    const a = renderHook(() => useChannelPods(1));
    const b = renderHook(() => useChannelPods(1));

    await waitFor(() => {
      expect(a.result.current.pods).toHaveLength(1);
      expect(b.result.current.pods).toHaveLength(1);
    });

    expect(channelApi.getPods).toHaveBeenCalledTimes(1);
    expect(channelApi.getPods).toHaveBeenCalledWith(1);
  });

  it("deduplicates in-flight calls when the hook is mounted twice rapidly", async () => {
    seed(2, [{ pod_key: "x", status: "running" }]);
    pendingFetch = () => undefined; // gate the first call

    renderHook(() => useChannelPods(2));
    renderHook(() => useChannelPods(2));

    // Both mounts → only one inflight request
    expect(channelApi.getPods).toHaveBeenCalledTimes(1);

    // Release the gate
    const release = pendingFetch;
    pendingFetch = null;
    release?.();
  });

  it("re-reads Rust state after invalidate + refresh", async () => {
    seed(4, [{ pod_key: "a" }]);

    const { result } = renderHook(() => useChannelPods(4));
    await waitFor(() => expect(result.current.pods).toHaveLength(1));

    seed(4, [{ pod_key: "a" }, { pod_key: "b" }]);
    invalidateChannelPods(4);
    await result.current.refresh();
    await waitFor(() => expect(result.current.pods).toHaveLength(2));
    expect(channelApi.getPods).toHaveBeenCalledTimes(2);
  });

  it("returns empty list when channelId is null", () => {
    const { result } = renderHook(() => useChannelPods(null));
    expect(result.current.pods).toEqual([]);
    expect(channelApi.getPods).not.toHaveBeenCalled();
  });
});
