import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  manager: {
    subscribe: vi.fn(),
    unsubscribe: vi.fn().mockResolvedValue(undefined),
    disconnect: vi.fn().mockResolvedValue(undefined),
    get_status: vi.fn().mockResolvedValue("disconnected"),
  },
}));

vi.mock("@agentsmesh/service-runtime", () => ({
  getRelayManager: () => mocks.manager,
}));

import { installRelayReadinessProbe } from "./relayReadinessProbe";

describe("relay readiness E2E probe", () => {
  beforeEach(() => {
    mocks.manager.subscribe.mockReset();
    mocks.manager.unsubscribe.mockReset();
    mocks.manager.disconnect.mockReset();
    mocks.manager.get_status.mockReset();
    mocks.manager.unsubscribe.mockResolvedValue(undefined);
    mocks.manager.disconnect.mockResolvedValue(undefined);
    mocks.manager.get_status.mockResolvedValue("disconnected");
    installRelayReadinessProbe(window);
  });

  afterEach(() => {
    delete window.__agentsmeshE2ERelayReadiness;
    vi.restoreAllMocks();
  });

  it("surfaces the subscription cancellation error and always disconnects", async () => {
    let rejectSubscribe!: (error: Error) => void;
    mocks.manager.subscribe.mockImplementationOnce(
      (
        _podKey: string,
        _subscriptionId: string,
        _relayUrl: string,
        _token: string,
        onOutput: (data: Uint8Array) => void,
      ) => {
        onOutput(new Uint8Array([1, 2, 3]));
        return new Promise<void>((_resolve, reject) => { rejectSubscribe = reject; });
      },
    );
    mocks.manager.unsubscribe.mockImplementationOnce(async () => {
      rejectSubscribe(new Error("readiness candidate removed"));
    });

    const probe = window.__agentsmeshE2ERelayReadiness!;
    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    await expect(probe.cancelPendingSubscribe()).rejects.toThrow("readiness candidate removed");

    expect(mocks.manager.subscribe).toHaveBeenCalledWith(
      expect.stringMatching(/^e2e-wasm-readiness-/),
      expect.stringMatching(/^cancel-/),
      "ws://relay-blackhole.test",
      "e2e-invalid-token",
      expect.any(Function),
    );
    expect(mocks.manager.unsubscribe).toHaveBeenCalledTimes(1);
    expect(mocks.manager.disconnect).toHaveBeenCalledTimes(1);
    expect(mocks.manager.get_status).toHaveBeenCalledWith(
      expect.stringMatching(/^e2e-wasm-readiness-/),
    );
    expect(probe.managerConstructorName).toBe("Object");
  });

  it("waits for the Rust driver handle to disappear before surfacing cancellation", async () => {
    let rejectSubscribe!: (error: Error) => void;
    mocks.manager.subscribe.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectSubscribe = reject; }),
    );
    mocks.manager.unsubscribe.mockImplementationOnce(async () => {
      rejectSubscribe(new Error("readiness candidate removed"));
    });
    mocks.manager.get_status
      .mockResolvedValueOnce("connected")
      .mockResolvedValueOnce("connecting")
      .mockResolvedValueOnce("disconnected");

    const probe = window.__agentsmeshE2ERelayReadiness!;
    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    await expect(probe.cancelPendingSubscribe()).rejects.toThrow("readiness candidate removed");

    expect(mocks.manager.get_status).toHaveBeenCalledTimes(3);
  });

  it("fails closed when the Rust driver handle does not disappear", async () => {
    let rejectSubscribe!: (error: Error) => void;
    mocks.manager.subscribe.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectSubscribe = reject; }),
    );
    mocks.manager.unsubscribe.mockImplementationOnce(async () => {
      rejectSubscribe(new Error("readiness candidate removed"));
    });
    mocks.manager.get_status.mockResolvedValue("connected");
    vi.spyOn(performance, "now")
      .mockReturnValueOnce(0)
      .mockReturnValue(5_000);

    const probe = window.__agentsmeshE2ERelayReadiness!;
    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    await expect(probe.cancelPendingSubscribe()).rejects.toThrow(
      /relay driver did not tear down/,
    );
  });

  it("fails closed if a subscription resolves without a baseline cancellation", async () => {
    mocks.manager.subscribe.mockResolvedValueOnce(undefined);

    const probe = window.__agentsmeshE2ERelayReadiness!;
    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    await expect(probe.cancelPendingSubscribe()).rejects.toThrow(
      "relay readiness unexpectedly resolved without a baseline",
    );

    expect(mocks.manager.disconnect).toHaveBeenCalledTimes(1);
  });

  it("rejects cancel-before-begin and overlapping begin or cancel calls", async () => {
    const probe = window.__agentsmeshE2ERelayReadiness!;
    await expect(probe.cancelPendingSubscribe()).rejects.toMatchObject({
      name: "InvalidStateError",
    });

    let rejectSubscribe!: (error: Error) => void;
    mocks.manager.subscribe.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectSubscribe = reject; }),
    );
    let releaseUnsubscribe!: () => void;
    mocks.manager.unsubscribe.mockImplementationOnce(
      () => new Promise<void>((resolve) => { releaseUnsubscribe = resolve; }),
    );

    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    expect(() => probe.beginPendingSubscribe("ws://second.test")).toThrowError(
      expect.objectContaining({ name: "InvalidStateError" }),
    );

    const cancelling = probe.cancelPendingSubscribe();
    await expect(probe.cancelPendingSubscribe()).rejects.toMatchObject({
      name: "InvalidStateError",
    });
    expect(() => probe.beginPendingSubscribe("ws://second.test")).toThrowError(
      expect.objectContaining({ name: "InvalidStateError" }),
    );
    rejectSubscribe(new Error("readiness candidate removed"));
    releaseUnsubscribe();
    await expect(cancelling).rejects.toThrow("readiness candidate removed");
  });

  it("surfaces cleanup failure and releases probe ownership", async () => {
    const probe = window.__agentsmeshE2ERelayReadiness!;
    mocks.manager.subscribe
      .mockRejectedValueOnce(new Error("readiness candidate removed"))
      .mockResolvedValueOnce(undefined);
    mocks.manager.disconnect.mockRejectedValueOnce(new Error("disconnect cleanup failed"));

    probe.beginPendingSubscribe("ws://relay-blackhole.test");
    await expect(probe.cancelPendingSubscribe()).rejects.toThrow("disconnect cleanup failed");

    expect(() => probe.beginPendingSubscribe("ws://replacement.test")).not.toThrow();
  });
});
