import { describe, it, expect, vi, beforeEach } from "vitest";
import { __resetChannelPodsCacheForTests } from "../useChannelPods";
import { renderHook, waitFor } from "@testing-library/react";
import { useMentionCandidates } from "../useMentionCandidates";

// Mock stores
vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    currentOrg: { slug: "test-org" },
    user: { id: 1 },
  }),
  useCurrentUser: () => ({ id: 1, email: "u@e.com", username: "u" }),
  useCurrentOrg: () => ({ id: 1, name: "TestOrg", slug: "test-org" }),
  useAuthOrganizations: () => [],
  readCurrentUser: () => ({ id: 1, email: "u@e.com", username: "u" }),
  readCurrentOrg: () => ({ id: 1, name: "TestOrg", slug: "test-org" }),
  readOrganizations: () => [],
}));

const mockPods = [
  { pod_key: "pk-bot", alias: "MyBot", agent: { name: "Claude" } },
];
vi.mock("@/stores/pod", () => ({
  usePodStore: (selector: (s: { pods: typeof mockPods }) => unknown) =>
    selector({ pods: mockPods }),
  usePods: () => mockPods,
}));

vi.mock("@/lib/pod-display-name", () => ({
  getPodDisplayName: (pod: { alias?: string; pod_key: string }) => pod.alias || pod.pod_key,
  getMentionSafeName: (pod: { alias?: string; pod_key: string }) => (pod.alias || pod.pod_key).replace(/\s+/g, "_"),
  getShortPodKey: (key: string) => key.substring(0, 8),
}));

vi.mock("@/lib/api/organization", () => ({
  organizationApi: {
    listMembers: vi.fn().mockResolvedValue({
      members: [
        { user: { id: 2, username: "alice", name: "Alice", email: "alice@test.com", avatar_url: null } },
        { user: { id: 1, username: "self", name: "Self", email: "self@test.com" } }, // current user, should be excluded
      ],
    }),
  },
}));

// Override wasm-core so Rust ChannelService (SSOT) reflects fetch outcomes:
// `get_channel_pods` seeds cache on success; `channel_pods_json` reads it.
const CHANNEL_PODS_FIXTURE = [
  { id: 1, pod_key: "pk-bot", alias: "MyBot", status: "running", agent_status: "idle" },
  { id: 2, pod_key: "pk-stopped", status: "terminated", agent_status: "idle" },
  { id: 3, pod_key: "pk-init", alias: null, status: "initializing", agent_status: "idle" },
];
const channelPodsCache = new Map<number, typeof CHANNEL_PODS_FIXTURE>();
vi.mock("@/lib/wasm-core", async () => {
  const actual = await vi.importActual<typeof import("@/lib/wasm-core")>("@/lib/wasm-core");
  return {
    ...actual,
    getChannelService: () => ({
      get_channel_pods: async (id: bigint) => {
        const num = Number(id);
        channelPodsCache.set(num, CHANNEL_PODS_FIXTURE);
        return JSON.stringify({ pods: CHANNEL_PODS_FIXTURE });
      },
      channel_pods_json: (id: bigint) => JSON.stringify(channelPodsCache.get(Number(id)) ?? []),
    }),
  };
});

vi.mock("@/lib/api/channel", () => ({
  channelApi: {
    getPods: vi.fn(async (id: number) => {
      // Simulate the real path: channelApi.getPods goes through WASM which
      // caches the response in Rust ChannelState. Mirror that here.
      channelPodsCache.set(id, CHANNEL_PODS_FIXTURE);
      return { pods: CHANNEL_PODS_FIXTURE, total: CHANNEL_PODS_FIXTURE.length };
    }),
  },
}));

describe("useMentionCandidates", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    __resetChannelPodsCacheForTests();
    channelPodsCache.clear();
  });

  it("returns empty when disabled", () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1, enabled: false })
    );
    expect(result.current.candidates).toEqual([]);
    expect(result.current.loading).toBe(false);
  });

  it("returns empty when channelId is null", () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: null })
    );
    expect(result.current.pods).toEqual([]);
  });

  it("fetches and excludes current user from members", async () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.members.length).toBe(1);
    });

    expect(result.current.members[0].id).toBe("user:2");
    expect(result.current.members[0].mentionText).toBe("alice");
    expect(result.current.members[0].displayName).toBe("Alice");
    expect(result.current.members[0].type).toBe("user");
  });

  it("fetches running/initializing pods and excludes terminated", async () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.pods.length).toBe(2);
    });

    const podKeys = result.current.pods.map((p) => p.id);
    expect(podKeys).toContain("pod:pk-bot");
    expect(podKeys).toContain("pod:pk-init");
    expect(podKeys).not.toContain("pod:pk-stopped");
  });

  it("resolves pod display name from store", async () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.pods.length).toBe(2);
    });

    const bot = result.current.pods.find((p) => p.id === "pod:pk-bot");
    expect(bot?.displayName).toBe("MyBot");
    expect(bot?.mentionText).toBe("MyBot");
  });

  it("falls back to short pod key when no alias or store entry", async () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.pods.length).toBe(2);
    });

    const initPod = result.current.pods.find((p) => p.id === "pod:pk-init");
    expect(initPod?.displayName).toBe("pk-init");
    expect(initPod?.mentionText).toBe("pk-init");
  });

  it("merges members and pods into candidates", async () => {
    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.candidates.length).toBe(3); // 1 member + 2 pods
    });

    const types = result.current.candidates.map((c) => c.type);
    expect(types).toContain("user");
    expect(types).toContain("pod");
  });

  it("handles member fetch error gracefully", async () => {
    const { organizationApi } = await import("@/lib/api/organization");
    vi.mocked(organizationApi.listMembers).mockRejectedValueOnce(new Error("network error"));

    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    // Should not throw — error logged internally
    await waitFor(() => {
      expect(result.current.pods.length).toBeGreaterThan(0);
    });
    expect(result.current.members).toEqual([]);
  });

  it("handles pod fetch error gracefully", async () => {
    const { channelApi } = await import("@/lib/api/channel");
    vi.mocked(channelApi.getPods).mockRejectedValueOnce(new Error("pod fetch failed"));

    const { result } = renderHook(() =>
      useMentionCandidates({ channelId: 1 })
    );

    await waitFor(() => {
      expect(result.current.members.length).toBeGreaterThan(0);
    });
    expect(result.current.pods).toEqual([]);
  });

  it("clears pods when channelId changes to null", async () => {
    const { result, rerender } = renderHook(
      ({ channelId }: { channelId: number | null }) =>
        useMentionCandidates({ channelId }),
      { initialProps: { channelId: 1 as number | null } }
    );

    await waitFor(() => {
      expect(result.current.pods.length).toBeGreaterThan(0);
    });

    rerender({ channelId: null });

    await waitFor(() => {
      expect(result.current.pods).toEqual([]);
    });
  });
});
