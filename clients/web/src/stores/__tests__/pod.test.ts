import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { create, toBinary } from "@bufbuild/protobuf";
import { ListPodsResponseSchema, PodSchema } from "@proto/pod/v1/pod_pb";
import { usePodStore } from "../pod";
import { getPodService } from "@/lib/wasm-core";
import { mockPod, mockPod2, resetPodStore, seedPods, readPods, readCurrentPod } from "./pod-test-utils";

interface MockService {
  list_pods_connect: ReturnType<typeof vi.fn>;
  get_pod_connect: ReturnType<typeof vi.fn>;
}

function svc(): MockService {
  return getPodService() as unknown as MockService;
}

function mockListPodsConnect(pods: unknown[], total = pods.length) {
  const items = pods.map((p) =>
    create(PodSchema, {
      id: BigInt((p as { id: number }).id),
      podKey: (p as { pod_key: string }).pod_key,
      status: (p as { status: string }).status,
      agentStatus: (p as { agent_status?: string }).agent_status ?? "",
      createdAt: (p as { created_at?: string }).created_at ?? "",
    }),
  );
  const resp = create(ListPodsResponseSchema, {
    items,
    total: BigInt(total),
    limit: 0,
    offset: 0,
  });
  vi.mocked(svc().list_pods_connect).mockResolvedValue(toBinary(ListPodsResponseSchema, resp));
}

function mockGetPodConnect(pod: unknown) {
  const protoPod = create(PodSchema, {
    id: BigInt((pod as { id: number }).id),
    podKey: (pod as { pod_key: string }).pod_key,
    status: (pod as { status: string }).status,
    agentStatus: (pod as { agent_status?: string }).agent_status ?? "",
    createdAt: (pod as { created_at?: string }).created_at ?? "",
  });
  vi.mocked(svc().get_pod_connect).mockResolvedValue(toBinary(PodSchema, protoPod));
}

describe("Pod Store — basic reads", () => {
  beforeEach(resetPodStore);

  describe("initial state", () => {
    it("should have default values", () => {
      const state = usePodStore.getState();
      expect(readPods()).toEqual([]);
      expect(readCurrentPod()).toBeNull();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe("fetchPods", () => {
    it("should fetch pods successfully", async () => {
      mockListPodsConnect([mockPod, mockPod2], 2);

      await act(async () => { await usePodStore.getState().fetchPods(); });

      const pods = readPods();
      expect(pods).toHaveLength(2);
      expect(pods[0].pod_key).toBe("pod-abc-123");
      expect(usePodStore.getState().loading).toBe(false);
      expect(usePodStore.getState().error).toBeNull();
    });

    it("should call Connect with status filter", async () => {
      mockListPodsConnect([]);

      await act(async () => {
        await usePodStore.getState().fetchPods({ status: "running" });
      });

      expect(svc().list_pods_connect).toHaveBeenCalled();
    });

    it("should handle empty response", async () => {
      mockListPodsConnect([]);
      await act(async () => { await usePodStore.getState().fetchPods(); });
      expect(readPods()).toEqual([]);
    });

    it("should handle fetch error", async () => {
      vi.mocked(svc().list_pods_connect).mockRejectedValue(new Error("Network error"));

      await act(async () => { await usePodStore.getState().fetchPods(); });

      const state = usePodStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should use default error message when no message provided", async () => {
      vi.mocked(svc().list_pods_connect).mockRejectedValue({});

      await act(async () => { await usePodStore.getState().fetchPods(); });
      expect(usePodStore.getState().error).toBe("Failed to fetch pods");
    });
  });

  describe("fetchPod", () => {
    it("should fetch single pod successfully", async () => {
      mockGetPodConnect(mockPod);

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      expect(readCurrentPod()).toBeNull();
      expect(usePodStore.getState().loading).toBe(false);
    });

    it("should add fetched pod to pods array when not present", async () => {
      mockGetPodConnect(mockPod);
      expect(readPods()).toEqual([]);

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      const pods = readPods();
      expect(pods).toHaveLength(1);
      expect(pods[0].pod_key).toBe(mockPod.pod_key);
    });

    it("should update existing pod in pods array when present", async () => {
      const updatedPod = { ...mockPod, status: "terminated" as const };
      seedPods(mockPod, mockPod2);
      mockGetPodConnect(updatedPod);

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      const pods = readPods();
      expect(pods).toHaveLength(2);
      expect(pods.find(p => p.pod_key === "pod-abc-123")?.status).toBe("terminated");
      expect(pods.find(p => p.pod_key === "pod-def-456")).toEqual(mockPod2);
    });

    it("should handle fetch error", async () => {
      vi.mocked(svc().get_pod_connect).mockRejectedValue({ message: "Pod not found" });

      await act(async () => {
        await usePodStore.getState().fetchPod("non-existent").catch(() => {});
      });

      const state = usePodStore.getState();
      expect(state.error).toBeNull();
      expect(state.loading).toBe(false);
    });
  });

  describe("setCurrentPod", () => {
    it("should set current pod", () => {
      act(() => { usePodStore.getState().setCurrentPod(mockPod); });
      expect(readCurrentPod()).toEqual(mockPod);
    });

    it("should set to null", () => {
      usePodStore.getState().setCurrentPod(mockPod);
      act(() => { usePodStore.getState().setCurrentPod(null); });
      expect(readCurrentPod()).toBeNull();
    });
  });

  describe("clearError", () => {
    it("should clear error", () => {
      usePodStore.setState({ error: "Some error" });
      act(() => { usePodStore.getState().clearError(); });
      expect(usePodStore.getState().error).toBeNull();
    });
  });
});
