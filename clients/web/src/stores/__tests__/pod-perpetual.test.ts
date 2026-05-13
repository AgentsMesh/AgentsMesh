import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { create, toBinary } from "@bufbuild/protobuf";
import { UpdatePodPerpetualResponseSchema } from "@proto/pod/v1/pod_pb";
import { usePodStore } from "../pod";
import { getPodService } from "@/lib/wasm-core";
import { mockPod, resetPodStore } from "./pod-test-utils";

interface MockService {
  get_pod_json: ReturnType<typeof vi.fn>;
  upsert_pod: ReturnType<typeof vi.fn>;
  pods_json: ReturnType<typeof vi.fn>;
  update_pod_perpetual_connect: ReturnType<typeof vi.fn>;
}

function svc(): MockService {
  return getPodService() as unknown as MockService;
}

const okBytes = () =>
  toBinary(
    UpdatePodPerpetualResponseSchema,
    create(UpdatePodPerpetualResponseSchema, { message: "ok" }),
  );

beforeEach(() => {
  resetPodStore();
  Object.assign(getPodService(), {
    get_pod_json: vi.fn((key: string) =>
      key === mockPod.pod_key ? JSON.stringify({ ...mockPod, perpetual: false }) : null,
    ),
    upsert_pod: vi.fn(),
    pods_json: vi.fn().mockReturnValue("[]"),
    update_pod_perpetual_connect: vi.fn().mockResolvedValue(okBytes()),
  });
});

describe("Pod Store — updatePodPerpetual", () => {
  it("calls Connect adapter and upserts the pod with new perpetual", async () => {
    await act(async () => {
      await usePodStore.getState().updatePodPerpetual(mockPod.pod_key, true);
    });

    expect(svc().update_pod_perpetual_connect).toHaveBeenCalledTimes(1);
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
    svc().update_pod_perpetual_connect.mockRejectedValue(new Error("Server error"));

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

    expect(svc().update_pod_perpetual_connect).toHaveBeenCalledTimes(1);
    expect(svc().upsert_pod).not.toHaveBeenCalled();
  });
});
