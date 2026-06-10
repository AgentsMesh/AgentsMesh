import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { ListPodsRequestSchema, ListPodsResponseSchema, PodSchema } from "@proto/pod/v1/pod_pb";
import { usePodStore, SIDEBAR_STATUS_MAP, Pod } from "../pod";
import { getAuthManager, getPodService } from "@/lib/wasm-core";
import {
  mockPod,
  mockPod2,
  resetPodStore,
  seedPods,
  readPods,
  podStateMock,
  lastAppendCachedPods,
} from "./pod-test-utils";

interface MockService {
  list_pods_connect: ReturnType<typeof vi.fn>;
}

function svc(): MockService {
  return getPodService() as unknown as MockService;
}

function encodePods(pods: unknown[], total: number) {
  const items = pods.map((p) =>
    create(PodSchema, {
      id: BigInt((p as { id: number }).id),
      podKey: (p as { pod_key: string }).pod_key,
      status: (p as { status: string }).status,
      agentStatus: (p as { agent_status?: string }).agent_status ?? "",
      createdAt: (p as { created_at?: string }).created_at ?? "",
    }),
  );
  const resp = create(ListPodsResponseSchema, { items, total: BigInt(total), limit: 0, offset: 0 });
  return toBinary(ListPodsResponseSchema, resp);
}

function mockSidebar(pods: unknown[], total: number) {
  vi.mocked(svc().list_pods_connect).mockResolvedValue(encodePods(pods, total));
}

function mockLoadMore(newPods: unknown[], total: number) {
  vi.mocked(svc().list_pods_connect).mockResolvedValue(encodePods(newPods, total));
}

describe("Pod Store — defaults", () => {
  it("should default currentSidebarFilter to mine", () => {
    expect(SIDEBAR_STATUS_MAP).toHaveProperty("mine");
    expect(SIDEBAR_STATUS_MAP).not.toHaveProperty("all");
  });

  it("should have mine as default currentSidebarFilter after reset", () => {
    resetPodStore();
    expect(usePodStore.getState().currentSidebarFilter).toBe("mine");
  });
});

describe("Pod Store — SIDEBAR_STATUS_MAP client-side guard", () => {
  function applyClientFilter(pods: Pod[], filter: string, userId?: number): Pod[] {
    const allowedStatuses = SIDEBAR_STATUS_MAP[filter];
    const statusSet = allowedStatuses
      ? new Set(allowedStatuses.split(","))
      : null;

    return pods.filter((pod) => {
      if (statusSet && !statusSet.has(pod.status)) return false;
      if (filter === "mine" && userId && pod.created_by?.id !== userId) return false;
      return true;
    });
  }

  const myPod: Pod = { ...mockPod, created_by: { id: 42, username: "me" } };
  const otherPod: Pod = { ...mockPod2, created_by: { id: 99, username: "other" } };
  const noPod: Pod = { ...mockPod, pod_key: "pod-no-creator" };

  it("mine filter should only show pods created by the current user", () => {
    const result = applyClientFilter([myPod, otherPod], "mine", 42);
    expect(result).toHaveLength(1);
    expect(result[0].pod_key).toBe(myPod.pod_key);
  });

  it("mine filter should exclude pods without created_by", () => {
    const result = applyClientFilter([myPod, noPod], "mine", 42);
    expect(result).toHaveLength(1);
    expect(result[0].pod_key).toBe(myPod.pod_key);
  });

  it("mine filter should show all pods when userId is undefined (not logged in)", () => {
    const result = applyClientFilter([myPod, otherPod], "mine", undefined);
    expect(result).toHaveLength(2);
  });

  it("org filter should only show running/initializing pods regardless of creator", () => {
    const runningPod: Pod = { ...otherPod, status: "running" };
    const terminatedPod: Pod = { ...myPod, status: "terminated" };
    const result = applyClientFilter([runningPod, terminatedPod], "org", 42);
    expect(result).toHaveLength(1);
    expect(result[0].pod_key).toBe(runningPod.pod_key);
  });

  it("completed filter should only show terminal status pods", () => {
    const runningPod: Pod = { ...myPod, status: "running" };
    const terminatedPod: Pod = { ...otherPod, status: "terminated" };
    const failedPod: Pod = { ...mockPod, pod_key: "pod-failed", status: "failed", agent_status: "idle", created_at: "2024-01-03T00:00:00Z" };
    const result = applyClientFilter([runningPod, terminatedPod, failedPod], "completed", 42);
    expect(result).toHaveLength(2);
  });
});

describe("Pod Store — fetchSidebarPods", () => {
  beforeEach(resetPodStore);

  it("should call list_pods_connect for org filter", async () => {
    mockSidebar([mockPod], 1);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org");
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
    expect(usePodStore.getState().currentSidebarFilter).toBe("org");
  });

  it("should call list_pods_connect for completed filter", async () => {
    mockSidebar([], 0);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("completed");
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
  });

  it("should call list_pods_connect with mine filter when user is logged in", async () => {
    getAuthManager().apply_session(JSON.stringify({ token: "t", refresh_token: "r", user: { id: 42, email: "test@test.com", username: "test" } }));
    mockSidebar([mockPod], 1);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("mine");
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
    expect(usePodStore.getState().currentSidebarFilter).toBe("mine");
  });

  it("should call list_pods_connect with mine filter when not logged in", async () => {
    (getAuthManager() as unknown as { _reset: () => void })._reset();
    mockSidebar([], 0);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("mine");
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
  });

  it("should set loading during fetch and clear after", async () => {
    let loadingDuringFetch = false;
    vi.mocked(svc().list_pods_connect).mockImplementation(async () => {
      loadingDuringFetch = usePodStore.getState().loading;
      return encodePods([], 0);
    });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org");
    });

    expect(loadingDuringFetch).toBe(true);
    expect(usePodStore.getState().loading).toBe(false);
  });

  it("should NOT flip loading during a silent refresh but still hit the network", async () => {
    usePodStore.setState({ currentSidebarFilter: "org" }); // silent refreshes the current filter
    let loadingDuringFetch = false;
    vi.mocked(svc().list_pods_connect).mockImplementation(async () => {
      loadingDuringFetch = usePodStore.getState().loading;
      return encodePods([mockPod], 1);
    });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org", { silent: true });
    });

    // Realtime reconnect / manual-refresh path: the list stays visible, no spinner.
    expect(loadingDuringFetch).toBe(false);
    expect(svc().list_pods_connect).toHaveBeenCalled();
    expect(usePodStore.getState().loading).toBe(false);
  });

  it("silent refresh updates data without flipping loading or filter", async () => {
    usePodStore.setState({ currentSidebarFilter: "org" });
    mockSidebar([mockPod], 5);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org", { silent: true });
    });

    expect(usePodStore.getState().currentSidebarFilter).toBe("org");
    expect(usePodStore.getState().podTotal).toBe(5);
    expect(usePodStore.getState().podHasMore).toBe(true);
    expect(usePodStore.getState().loading).toBe(false);
  });

  it("should discard a stale page when the filter changes mid-flight", async () => {
    usePodStore.setState({ currentSidebarFilter: "mine" });
    vi.mocked(svc().list_pods_connect).mockImplementation(async () => {
      // User switches tabs while this request is in flight.
      usePodStore.setState({ currentSidebarFilter: "org" });
      return encodePods([mockPod], 5);
    });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("mine", { silent: true });
    });

    // The stale "mine" page must not clobber the now-active "org" cache/total.
    expect(podStateMock().replace_cached_pods).not.toHaveBeenCalled();
    expect(usePodStore.getState().podTotal).not.toBe(5);
  });

  it("silent refresh failure should preserve existing error, loading, and list", async () => {
    usePodStore.setState({ currentSidebarFilter: "org" });
    seedPods(mockPod);
    usePodStore.setState({ error: "stale error", loading: false });
    vi.mocked(svc().list_pods_connect).mockRejectedValue(new Error("network down"));

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org", { silent: true });
    });

    // Silent failure is swallowed and never touches the cache or visible state.
    expect(usePodStore.getState().error).toBe("stale error");
    expect(usePodStore.getState().loading).toBe(false);
    expect(podStateMock().replace_cached_pods).not.toHaveBeenCalled();
    expect(readPods()).toHaveLength(1);
  });

  it("should compute podHasMore correctly", async () => {
    mockSidebar([mockPod], 5);

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org");
    });

    expect(usePodStore.getState().podHasMore).toBe(true);
    expect(usePodStore.getState().podTotal).toBe(5);
  });

  it("should handle error and clear loading", async () => {
    vi.mocked(svc().list_pods_connect).mockRejectedValue(new Error("Network error"));

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("org");
    });

    expect(usePodStore.getState().error).toBe("Network error");
    expect(usePodStore.getState().loading).toBe(false);
  });
});

describe("Pod Store — loadMorePods", () => {
  beforeEach(resetPodStore);

  it("should page from sidebarLoadedCount, not the realtime-polluted cache length", async () => {
    // Realtime insert_created_pod has bloated the shared cache to 2 pods, but
    // the active filter only ever loaded 1 page-row of its own.
    seedPods(mockPod, mockPod2);
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "org", sidebarLoadedCount: 1 });
    let capturedOffset = -1;
    vi.mocked(svc().list_pods_connect).mockImplementation(async (bytes: unknown) => {
      capturedOffset = Number(fromBinary(ListPodsRequestSchema, bytes as Uint8Array).offset);
      return encodePods([mockPod2], 2);
    });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
    // offset must be sidebarLoadedCount (1), NOT the polluted cache length (2).
    expect(capturedOffset).toBe(1);
    const appended = lastAppendCachedPods();
    expect(appended[0].pod_key).toBe(mockPod2.pod_key);
  });

  it("should skip when no more pods", async () => {
    seedPods(mockPod);
    usePodStore.setState({ podHasMore: false });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(svc().list_pods_connect).not.toHaveBeenCalled();
  });

  it("should skip when already loading more", async () => {
    seedPods(mockPod);
    usePodStore.setState({ podHasMore: true, loadingMore: true });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(svc().list_pods_connect).not.toHaveBeenCalled();
  });

  it("should deduplicate pods already in list (upsert by pod_key)", async () => {
    seedPods(mockPod, mockPod2);
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "org" });
    mockLoadMore([mockPod2], 3);

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    // The bridge handles dedup on the wasm side; the store hands the new
    // page over without checking. Production assertion: append was called.
    expect(svc().list_pods_connect).toHaveBeenCalled();
    const appended = lastAppendCachedPods();
    expect(appended.map((p) => p.pod_key)).toEqual([mockPod2.pod_key]);
  });

  it("should load mine filter when user is logged in", async () => {
    getAuthManager().apply_session(JSON.stringify({ token: "t", refresh_token: "r", user: { id: 42, email: "test@test.com", username: "test" } }));
    seedPods(mockPod);
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "mine" });
    mockLoadMore([mockPod2], 2);

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(svc().list_pods_connect).toHaveBeenCalled();
    const appended = lastAppendCachedPods();
    expect(appended.map((p) => p.pod_key)).toEqual([mockPod2.pod_key]);
  });

  it("should advance sidebarLoadedCount and recompute hasMore from it", async () => {
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "org", sidebarLoadedCount: 20 });
    mockLoadMore([mockPod, mockPod2], 25); // loaded 20 → 22, still < 25

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(usePodStore.getState().sidebarLoadedCount).toBe(22);
    expect(usePodStore.getState().podHasMore).toBe(true);

    mockLoadMore(
      [{ ...mockPod, id: 23, pod_key: "p23" }, { ...mockPod, id: 24, pod_key: "p24" }, { ...mockPod, id: 25, pod_key: "p25" }],
      25,
    ); // loaded 22 → 25, hasMore false

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(usePodStore.getState().sidebarLoadedCount).toBe(25);
    expect(usePodStore.getState().podHasMore).toBe(false);
  });
});

describe("Pod Store — sidebar fetch/loadMore out-of-order guards", () => {
  beforeEach(resetPodStore);

  it("a slower stale same-filter fetch must not clobber a newer one's cache + total", async () => {
    usePodStore.setState({ currentSidebarFilter: "org" });
    let resolveSlow: ((b: Uint8Array) => void) | undefined;
    const slow = new Promise<Uint8Array>((r) => { resolveSlow = r; });
    vi.mocked(svc().list_pods_connect)
      .mockReturnValueOnce(slow as unknown as Promise<Uint8Array>) // first (older, slow)
      .mockResolvedValueOnce(encodePods([mockPod2], 99)); // second (newer, fast)

    const pSlow = usePodStore.getState().fetchSidebarPods("org", { silent: true });
    const pFast = usePodStore.getState().fetchSidebarPods("org", { silent: true });
    await act(async () => { await pFast; });

    expect(usePodStore.getState().podTotal).toBe(99);
    const replaceCalls = podStateMock().replace_cached_pods.mock.calls.length;

    resolveSlow!(encodePods([mockPod], 1));
    await act(async () => { await pSlow; });

    // The superseded request's response is dropped by the seq guard — neither
    // the cache nor the total regresses to its stale values.
    expect(usePodStore.getState().podTotal).toBe(99);
    expect(podStateMock().replace_cached_pods.mock.calls.length).toBe(replaceCalls);
  });

  it("a non-silent cold load superseded by a silent refresh still clears loading", async () => {
    let resolveCold: ((b: Uint8Array) => void) | undefined;
    const coldSlow = new Promise<Uint8Array>((r) => { resolveCold = r; });
    vi.mocked(svc().list_pods_connect)
      .mockReturnValueOnce(coldSlow as unknown as Promise<Uint8Array>) // non-silent cold load (slow)
      .mockResolvedValueOnce(encodePods([mockPod], 1)); // silent refresh (fast)

    const pCold = usePodStore.getState().fetchSidebarPods("mine"); // non-silent → loading:true
    const pSilent = usePodStore.getState().fetchSidebarPods("mine", { silent: true }); // bumps seq, never touches loading
    await act(async () => {
      await pSilent;
    });

    // Silent landed; cold load still in flight, loading stays true (silent never clears it).
    expect(usePodStore.getState().loading).toBe(true);

    resolveCold!(encodePods([mockPod2], 2));
    await act(async () => {
      await pCold;
    });

    // Cold load is seq-superseded (skips the data write) but still OWNS the
    // spinner, so its finally clears loading — no permanent spinner.
    expect(usePodStore.getState().loading).toBe(false);
  });

  it("loadMorePods discards its page when a fetchSidebarPods supersedes it mid-flight", async () => {
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "org", sidebarLoadedCount: 20 });
    let resolveLoadMore: ((b: Uint8Array) => void) | undefined;
    const loadMoreSlow = new Promise<Uint8Array>((r) => { resolveLoadMore = r; });
    vi.mocked(svc().list_pods_connect)
      .mockReturnValueOnce(loadMoreSlow as unknown as Promise<Uint8Array>) // loadMore (slow)
      .mockResolvedValueOnce(encodePods([mockPod], 5)); // fetchSidebarPods (fast, bumps seq)

    const pLoadMore = usePodStore.getState().loadMorePods();
    const pFetch = usePodStore.getState().fetchSidebarPods("org", { silent: true });
    await act(async () => {
      await pFetch;
    });

    const appendsBefore = podStateMock().append_cached_pods.mock.calls.length;
    resolveLoadMore!(encodePods([mockPod2], 99));
    await act(async () => {
      await pLoadMore;
    });

    // loadMore was superseded (seq bumped) → must NOT append at the now-stale
    // offset nor write back an inflated count/total.
    expect(podStateMock().append_cached_pods.mock.calls.length).toBe(appendsBefore);
    expect(usePodStore.getState().loadingMore).toBe(false);
    expect(usePodStore.getState().podTotal).not.toBe(99);
  });

  it("loadMorePods discards when a fetchSidebarPods was ALREADY in flight at start (same seq baseline)", async () => {
    usePodStore.setState({ podHasMore: true, currentSidebarFilter: "org", sidebarLoadedCount: 40 });
    let resolveFetch: ((b: Uint8Array) => void) | undefined;
    const fetchSlow = new Promise<Uint8Array>((r) => { resolveFetch = r; });
    let resolveLoadMore: ((b: Uint8Array) => void) | undefined;
    const loadMoreSlow = new Promise<Uint8Array>((r) => { resolveLoadMore = r; });
    vi.mocked(svc().list_pods_connect)
      .mockReturnValueOnce(fetchSlow as unknown as Promise<Uint8Array>) // fetch first (in-flight, bumps seq)
      .mockReturnValueOnce(loadMoreSlow as unknown as Promise<Uint8Array>); // loadMore (same seq baseline)

    const pFetch = usePodStore.getState().fetchSidebarPods("org", { silent: true });
    const pLoadMore = usePodStore.getState().loadMorePods(); // captures count=40, mySeq == fetch's seq

    // fetch resolves first: replaces cache + resets sidebarLoadedCount to the page size.
    resolveFetch!(encodePods([mockPod], 99));
    await act(async () => { await pFetch; });
    const appendsBefore = podStateMock().append_cached_pods.mock.calls.length;

    // loadMore resolves: seq never changed (it never bumped), but the offset was
    // reset — the loaded-count guard must discard it rather than append at 40.
    resolveLoadMore!(encodePods([mockPod2], 99));
    await act(async () => { await pLoadMore; });
    expect(podStateMock().append_cached_pods.mock.calls.length).toBe(appendsBefore);
  });
});
