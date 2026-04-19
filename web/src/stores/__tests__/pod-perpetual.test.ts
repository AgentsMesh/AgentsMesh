import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { usePodStore } from "../pod";
import { getApiClient, getPodService } from "@/lib/wasm-core";
import { mockPod, resetPodStore } from "./pod-test-utils";

interface MockClient {
  org_path: ReturnType<typeof vi.fn>;
  patch: ReturnType<typeof vi.fn>;
}

interface MockService {
  get_pod_json: ReturnType<typeof vi.fn>;
  upsert_pod: ReturnType<typeof vi.fn>;
  pods_json: ReturnType<typeof vi.fn>;
}

function client(): MockClient {
  return getApiClient() as unknown as MockClient;
}

function svc(): MockService {
  return getPodService() as unknown as MockService;
}

beforeEach(() => {
  resetPodStore();
  // apiClient stubs
  Object.assign(getApiClient(), {
    org_path: vi.fn().mockImplementation((p: string) => `/api/v1/orgs/test/${p.replace(/^\//, "")}`),
    patch: vi.fn().mockResolvedValue(JSON.stringify({ message: "ok" })),
  });
  // podService stubs — return target pod as JSON for get_pod_json
  Object.assign(getPodService(), {
    get_pod_json: vi.fn((key: string) =>
      key === mockPod.pod_key ? JSON.stringify({ ...mockPod, perpetual: false }) : null,
    ),
    upsert_pod: vi.fn(),
    pods_json: vi.fn().mockReturnValue("[]"),
  });
});

describe("Pod Store — updatePodPerpetual", () => {
  it("calls ApiClient.patch and upserts the pod with new perpetual", async () => {
    await act(async () => {
      await usePodStore.getState().updatePodPerpetual(mockPod.pod_key, true);
    });

    expect(client().org_path).toHaveBeenCalledWith(`/pods/${mockPod.pod_key}/perpetual`);
    expect(client().patch).toHaveBeenCalledTimes(1);
    const [, body] = client().patch.mock.calls[0];
    expect(JSON.parse(body)).toEqual({ perpetual: true });

    expect(svc().upsert_pod).toHaveBeenCalledTimes(1);
    const [upsertBody] = svc().upsert_pod.mock.calls[0];
    expect(JSON.parse(upsertBody).perpetual).toBe(true);
  });

  it("flips perpetual to false", async () => {
    svc().get_pod_json.mockReturnValue(JSON.stringify({ ...mockPod, perpetual: true }));

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual(mockPod.pod_key, false);
    });

    const [upsertBody] = svc().upsert_pod.mock.calls[0];
    expect(JSON.parse(upsertBody).perpetual).toBe(false);
  });

  it("records the error and rethrows when the API call fails", async () => {
    client().patch.mockRejectedValue(new Error("Server error"));

    await act(async () => {
      await expect(
        usePodStore.getState().updatePodPerpetual(mockPod.pod_key, true),
      ).rejects.toThrow("Server error");
    });

    expect(svc().upsert_pod).not.toHaveBeenCalled();
    expect(usePodStore.getState().error).toContain("Server error");
  });

  it("does nothing when the pod is not in WASM state", async () => {
    svc().get_pod_json.mockReturnValue(null);

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual("missing-pod", true);
    });

    // API was still called (the server is authoritative), but no local upsert.
    expect(client().patch).toHaveBeenCalledTimes(1);
    expect(svc().upsert_pod).not.toHaveBeenCalled();
  });
});
