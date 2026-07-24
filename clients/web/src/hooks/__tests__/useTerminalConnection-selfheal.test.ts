import { describe, it, expect, vi, beforeEach } from "vitest";

const mocks = vi.hoisted(() => ({
  subscribe: vi.fn(),
  forceResize: vi.fn(),
  onStatusChange: vi.fn(() => () => undefined),
  removePaneByPodKey: vi.fn(),
}));

vi.mock("@/stores/workspace", () => ({
  relayPool: {
    subscribe: mocks.subscribe,
    forceResize: mocks.forceResize,
    onStatusChange: mocks.onStatusChange,
  },
  useWorkspaceStore: {
    getState: () => ({ removePaneByPodKey: mocks.removePaneByPodKey }),
  },
}));

import { setupConnection } from "../useTerminalConnection";

const scheduler = { schedule: vi.fn() } as unknown as import("@/lib/terminalScheduler").TerminalWriteScheduler;

describe("useTerminalConnection", () => {
  beforeEach(() => {
    mocks.subscribe.mockReset();
    mocks.forceResize.mockReset();
    mocks.removePaneByPodKey.mockReset();
    mocks.onStatusChange.mockReset();
    mocks.onStatusChange.mockReturnValue(() => undefined);
    scheduler.schedule = vi.fn();
  });

  it("normalizes binary and string relay output through the write scheduler", async () => {
    let onMessage!: (data: Uint8Array | string) => void;
    const handle = { send: vi.fn(), unsubscribe: vi.fn() };
    mocks.subscribe.mockImplementationOnce((_podKey, _subscriptionId, callback) => {
      onMessage = callback;
      return Promise.resolve(handle);
    });
    const connectionRef = { current: null };

    setupConnection("pod-output", scheduler, connectionRef, vi.fn(), vi.fn());
    await vi.waitFor(() => expect(connectionRef.current).toBe(handle));
    const bytes = new Uint8Array([0x41, 0x42]);
    onMessage(bytes);
    onMessage("界");

    expect(scheduler.schedule).toHaveBeenNthCalledWith(1, bytes);
    expect(scheduler.schedule).toHaveBeenNthCalledWith(
      2,
      new TextEncoder().encode("界"),
    );
  });

  it("fans relay status out while preserving the none baseline", () => {
    let onStatus!: (info: { status: string; runnerDisconnected: boolean }) => void;
    const unsubscribeStatus = vi.fn();
    mocks.subscribe.mockReturnValue(new Promise(() => {}));
    mocks.onStatusChange.mockImplementationOnce((_podKey, callback) => {
      onStatus = callback;
      return unsubscribeStatus;
    });
    const setConnectionStatus = vi.fn();
    const setIsRunnerDisconnected = vi.fn();

    const result = setupConnection(
      "pod-status",
      scheduler,
      { current: null },
      setConnectionStatus,
      setIsRunnerDisconnected,
    );
    onStatus({ status: "none", runnerDisconnected: false });
    onStatus({ status: "connecting", runnerDisconnected: true });

    expect(setConnectionStatus).toHaveBeenCalledExactlyOnceWith("connecting");
    expect(setIsRunnerDisconnected).toHaveBeenNthCalledWith(1, false);
    expect(setIsRunnerDisconnected).toHaveBeenNthCalledWith(2, true);
    expect(result.unsubscribeStatus).toBe(unsubscribeStatus);
    result.abort.abort();
  });

  it("does not use a global resize to establish the subscriber baseline", async () => {
    let resolveSubscribe!: (handle: { send: () => void; unsubscribe: () => void }) => void;
    mocks.subscribe.mockReturnValueOnce(new Promise((resolve) => {
      resolveSubscribe = resolve;
    }));

    const connectionRef = { current: null };
    setupConnection("pod-1", scheduler, connectionRef, vi.fn(), vi.fn());
    expect(mocks.forceResize).not.toHaveBeenCalled();

    const handle = { send: vi.fn(), unsubscribe: vi.fn() };
    resolveSubscribe(handle);
    await vi.waitFor(() => expect(connectionRef.current).toBe(handle));

    expect(mocks.forceResize).not.toHaveBeenCalled();
  });

  it("does not let a stale attempt resolving last unsubscribe its replacement", async () => {
    let resolveStale!: (handle: { send: () => void; unsubscribe: () => void }) => void;
    let resolveCurrent!: (handle: { send: () => void; unsubscribe: () => void }) => void;
    mocks.subscribe
      .mockReturnValueOnce(new Promise((resolve) => { resolveStale = resolve; }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveCurrent = resolve; }));

    const connectionRef = { current: null };
    const staleAttempt = setupConnection(
      "pod-1", scheduler, connectionRef, vi.fn(), vi.fn(),
    );
    staleAttempt.abort.abort();
    setupConnection("pod-1", scheduler, connectionRef, vi.fn(), vi.fn());

    const staleId = mocks.subscribe.mock.calls[0][1];
    const currentId = mocks.subscribe.mock.calls[1][1];
    expect(staleId).not.toBe(currentId);

    const currentHandle = { send: vi.fn(), unsubscribe: vi.fn() };
    resolveCurrent(currentHandle);
    await vi.waitFor(() => expect(connectionRef.current).toBe(currentHandle));

    const staleHandle = { send: vi.fn(), unsubscribe: vi.fn() };
    resolveStale(staleHandle);
    await vi.waitFor(() => expect(staleHandle.unsubscribe).toHaveBeenCalledTimes(1));

    expect(connectionRef.current).toBe(currentHandle);
    expect(currentHandle.unsubscribe).not.toHaveBeenCalled();
    expect(mocks.forceResize).not.toHaveBeenCalled();
  });

  it("allows the replacement to establish after the stale attempt resolves and cleans up first", async () => {
    let resolveStale!: (handle: { send: () => void; unsubscribe: () => void }) => void;
    let resolveCurrent!: (handle: { send: () => void; unsubscribe: () => void }) => void;
    mocks.subscribe
      .mockReturnValueOnce(new Promise((resolve) => { resolveStale = resolve; }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveCurrent = resolve; }));

    const connectionRef = { current: null };
    const staleAttempt = setupConnection(
      "pod-1", scheduler, connectionRef, vi.fn(), vi.fn(),
    );
    staleAttempt.abort.abort();
    setupConnection("pod-1", scheduler, connectionRef, vi.fn(), vi.fn());

    expect(mocks.subscribe.mock.calls[0][1]).not.toBe(mocks.subscribe.mock.calls[1][1]);

    const staleHandle = { send: vi.fn(), unsubscribe: vi.fn() };
    resolveStale(staleHandle);
    await vi.waitFor(() => expect(staleHandle.unsubscribe).toHaveBeenCalledTimes(1));
    expect(connectionRef.current).toBeNull();
    expect(mocks.forceResize).not.toHaveBeenCalled();

    const currentHandle = { send: vi.fn(), unsubscribe: vi.fn() };
    resolveCurrent(currentHandle);
    await vi.waitFor(() => expect(connectionRef.current).toBe(currentHandle));

    expect(currentHandle.unsubscribe).not.toHaveBeenCalled();
    expect(mocks.forceResize).not.toHaveBeenCalled();
  });

  // Regression: 226 stale panes persisted in localStorage all tried to
  // connect, each producing an error-status spam. Now a definitive "pod
  // gone" response (HTTP 404 / RESOURCE_NOT_FOUND) removes the dead pane
  // from the store so subsequent renders stay clean.
  it("drops the pane when server returns Pod not found 404 (legacy string)", async () => {
    mocks.subscribe.mockRejectedValueOnce(
      new Error("Error invoking remote method 'podGetPodConnection': Error: HTTP 404: Pod not found [RESOURCE_NOT_FOUND]"),
    );

    const setConnectionStatus = vi.fn();
    setupConnection(
      "1-404-gone", scheduler, { current: null }, setConnectionStatus, vi.fn(),
    );
    await new Promise((r) => setTimeout(r, 0));

    expect(mocks.removePaneByPodKey).toHaveBeenCalledWith("1-404-gone");
    expect(setConnectionStatus).not.toHaveBeenCalledWith("error");
  });

  it("drops the pane when server returns ServiceError resource_not_found JSON", async () => {
    mocks.subscribe.mockRejectedValueOnce(
      new Error('{"kind":"resource_not_found","resource":"Pod","id":"pk_1"}'),
    );

    const setConnectionStatus = vi.fn();
    setupConnection(
      "1-404-json", scheduler, { current: null }, setConnectionStatus, vi.fn(),
    );
    await new Promise((r) => setTimeout(r, 0));

    expect(mocks.removePaneByPodKey).toHaveBeenCalledWith("1-404-json");
    expect(setConnectionStatus).not.toHaveBeenCalledWith("error");
  });

  it("surfaces other errors as connection status 'error'", async () => {
    mocks.subscribe.mockRejectedValueOnce(new Error("network flaked"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    const setConnectionStatus = vi.fn();
    setupConnection(
      "1-live-abc", scheduler, { current: null }, setConnectionStatus, vi.fn(),
    );
    await new Promise((r) => setTimeout(r, 0));

    expect(mocks.removePaneByPodKey).not.toHaveBeenCalled();
    expect(setConnectionStatus).toHaveBeenCalledWith("error");
    expect(consoleError).toHaveBeenCalledWith(
      "Failed to connect terminal:",
      expect.objectContaining({ message: "network flaked" }),
    );
    consoleError.mockRestore();
  });

  it("ignores the benign pod-not-active lifecycle transient", async () => {
    mocks.subscribe.mockRejectedValueOnce(new Error("pod is not active"));
    const setConnectionStatus = vi.fn();
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    setupConnection(
      "pod-starting",
      scheduler,
      { current: null },
      setConnectionStatus,
      vi.fn(),
    );
    await vi.waitFor(() => expect(mocks.subscribe).toHaveBeenCalled());
    await Promise.resolve();

    expect(mocks.removePaneByPodKey).not.toHaveBeenCalled();
    expect(setConnectionStatus).not.toHaveBeenCalledWith("error");
    expect(consoleError).not.toHaveBeenCalled();
    consoleError.mockRestore();
  });

  it("ignores a subscription rejection after the attempt was aborted", async () => {
    let reject!: (error: unknown) => void;
    mocks.subscribe.mockReturnValueOnce(
      new Promise((_resolve, fail) => { reject = fail; }),
    );
    const setConnectionStatus = vi.fn();
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    const attempt = setupConnection(
      "pod-aborted",
      scheduler,
      { current: null },
      setConnectionStatus,
      vi.fn(),
    );

    attempt.abort.abort();
    reject(new Error("late failure"));
    await Promise.resolve();
    await Promise.resolve();

    expect(setConnectionStatus).not.toHaveBeenCalled();
    expect(consoleError).not.toHaveBeenCalled();
    consoleError.mockRestore();
  });
});
