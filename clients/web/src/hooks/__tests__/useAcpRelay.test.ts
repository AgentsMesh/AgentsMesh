import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  subscribe: vi.fn(),
  unsubscribe: vi.fn(),
  onAcpMessage: vi.fn(() => vi.fn()),
  dispatchAcpRelayEvent: vi.fn(),
  dispatchLoopalRelayEvent: vi.fn(() => false),
  isResourceNotFound: vi.fn(() => false),
  isPodNotConnectable: vi.fn(() => false),
}));

vi.mock("@/stores/relayConnection", () => ({ relayPool: mocks }));
vi.mock("@/stores/acpEventDispatcher", () => ({
  dispatchAcpRelayEvent: mocks.dispatchAcpRelayEvent,
}));
vi.mock("@/stores/loopalDispatcher", () => ({
  dispatchLoopalRelayEvent: mocks.dispatchLoopalRelayEvent,
}));
vi.mock("@/lib/errors/serviceError", () => ({
  isResourceNotFound: mocks.isResourceNotFound,
  isPodNotConnectable: mocks.isPodNotConnectable,
}));

import { useAcpRelay } from "../useAcpRelay";

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => { resolve = done; });
  return { promise, resolve };
}

describe("useAcpRelay subscription lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.subscribe.mockResolvedValue(undefined);
    mocks.onAcpMessage.mockReturnValue(vi.fn());
    mocks.dispatchLoopalRelayEvent.mockReturnValue(false);
    mocks.isResourceNotFound.mockReturnValue(false);
    mocks.isPodNotConnectable.mockReturnValue(false);
  });

  it("does nothing while inactive", () => {
    const { unmount } = renderHook(() => useAcpRelay("pod-1", "pane-1", false));

    expect(mocks.subscribe).not.toHaveBeenCalled();
    expect(mocks.onAcpMessage).not.toHaveBeenCalled();
    unmount();
    expect(mocks.unsubscribe).not.toHaveBeenCalled();
  });

  it("uses a unique subscription generation and tears down every owner", () => {
    const offFirst = vi.fn();
    const offSecond = vi.fn();
    mocks.onAcpMessage.mockReturnValueOnce(offFirst).mockReturnValueOnce(offSecond);

    const first = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    const second = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    const firstId = mocks.subscribe.mock.calls[0]![1] as string;
    const secondId = mocks.subscribe.mock.calls[1]![1] as string;

    expect(firstId).toMatch(/^acp-pane-1-\d+$/);
    expect(secondId).toMatch(/^acp-pane-1-\d+$/);
    expect(secondId).not.toBe(firstId);
    expect(mocks.subscribe.mock.calls[0]![3]).toBeInstanceOf(AbortSignal);

    first.unmount();
    second.unmount();
    expect(mocks.unsubscribe).toHaveBeenCalledWith("pod-1", firstId);
    expect(mocks.unsubscribe).toHaveBeenCalledWith("pod-1", secondId);
    expect(offFirst).toHaveBeenCalledOnce();
    expect(offSecond).toHaveBeenCalledOnce();
  });

  it("lets Loopal consume ACP messages before the generic dispatcher", () => {
    let listener!: (msgType: number, payload: unknown) => void;
    mocks.subscribe.mockImplementation((_podKey, _subscriptionId, onMessage) => {
      onMessage();
      return Promise.resolve();
    });
    mocks.onAcpMessage.mockImplementation((_podKey, callback) => {
      listener = callback;
      return vi.fn();
    });
    const hook = renderHook(() => useAcpRelay("pod-1", "pane-1", true));

    listener(11, { type: "first" });
    expect(mocks.dispatchLoopalRelayEvent).toHaveBeenCalledWith(
      "pod-1",
      11,
      { type: "first" },
    );
    expect(mocks.dispatchAcpRelayEvent).toHaveBeenCalledWith(
      "pod-1",
      11,
      { type: "first" },
    );

    mocks.dispatchLoopalRelayEvent.mockReturnValue(true);
    mocks.dispatchAcpRelayEvent.mockClear();
    listener(12, { type: "loopal" });
    expect(mocks.dispatchAcpRelayEvent).not.toHaveBeenCalled();
    hook.unmount();
  });

  it("prevents a manager subscription when endpoint selection finishes after unmount", async () => {
    const endpoint = deferred();
    let endpointFinished = false;
    let managerStarted = false;
    let signal!: AbortSignal;
    mocks.subscribe.mockImplementationOnce(
      async (_podKey, _subId, _callback, subscriptionSignal: AbortSignal) => {
        signal = subscriptionSignal;
        await endpoint.promise;
        endpointFinished = true;
        if (signal.aborted) {
          throw new DOMException("Relay subscription aborted", "AbortError");
        }
        managerStarted = true;
      },
    );

    const { unmount } = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    unmount();
    endpoint.resolve();
    await vi.waitFor(() => expect(endpointFinished).toBe(true));

    expect(signal.aborted).toBe(true);
    expect(managerStarted).toBe(false);
  });

  it("aborts a subscription that is still waiting for its baseline", async () => {
    let signal!: AbortSignal;
    let baselinePending = false;
    mocks.subscribe.mockImplementationOnce(
      (_podKey, _subId, _callback, subscriptionSignal: AbortSignal) => {
        signal = subscriptionSignal;
        baselinePending = true;
        return new Promise<void>((_resolve, reject) => {
          signal.addEventListener("abort", () => {
            baselinePending = false;
            reject(new DOMException("Relay subscription aborted", "AbortError"));
          }, { once: true });
        });
      },
    );

    const { unmount } = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    expect(baselinePending).toBe(true);
    unmount();
    await vi.waitFor(() => expect(signal.aborted).toBe(true));

    expect(baselinePending).toBe(false);
    expect(mocks.unsubscribe).toHaveBeenCalledWith(
      "pod-1",
      expect.stringMatching(/^acp-pane-1-\d+$/),
    );
  });

  it.each([
    ["resource-not-found", "isResourceNotFound"],
    ["not-connectable", "isPodNotConnectable"],
  ] as const)("silences the expected %s lifecycle error", async (_name, predicate) => {
    const error = new Error("pod lifecycle race");
    mocks[predicate].mockReturnValue(true);
    mocks.subscribe.mockRejectedValue(error);
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    const hook = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    await vi.waitFor(() => expect(mocks[predicate]).toHaveBeenCalledWith(error));

    expect(consoleError).not.toHaveBeenCalled();
    hook.unmount();
    consoleError.mockRestore();
  });

  it("reports a genuine connection failure", async () => {
    const error = new Error("relay unavailable");
    mocks.subscribe.mockRejectedValue(error);
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    const hook = renderHook(() => useAcpRelay("pod-1", "pane-1", true));
    await vi.waitFor(() => {
      expect(consoleError).toHaveBeenCalledWith("ACP relay subscribe failed:", error);
    });

    hook.unmount();
    consoleError.mockRestore();
  });

  it("silences a late failure after cleanup aborted the attempt", async () => {
    let reject!: (error: unknown) => void;
    mocks.subscribe.mockReturnValue(new Promise<void>((_resolve, fail) => { reject = fail; }));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    const hook = renderHook(() => useAcpRelay("pod-1", "pane-1", true));

    hook.unmount();
    reject(new Error("late failure"));
    await Promise.resolve();
    await Promise.resolve();

    expect(consoleError).not.toHaveBeenCalled();
    expect(mocks.isResourceNotFound).not.toHaveBeenCalled();
    consoleError.mockRestore();
  });
});
