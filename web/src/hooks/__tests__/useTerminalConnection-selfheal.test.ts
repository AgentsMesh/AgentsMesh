import { describe, it, expect, vi, beforeEach } from "vitest";

const mocks = vi.hoisted(() => ({
  subscribe: vi.fn(),
  onStatusChange: vi.fn(() => () => undefined),
  removePaneByPodKey: vi.fn(),
}));

vi.mock("@/stores/workspace", () => ({
  relayPool: {
    subscribe: mocks.subscribe,
    onStatusChange: mocks.onStatusChange,
  },
  useWorkspaceStore: {
    getState: () => ({ removePaneByPodKey: mocks.removePaneByPodKey }),
  },
}));

import { setupConnection } from "../useTerminalConnection";

const scheduler = { schedule: vi.fn() } as unknown as import("@/lib/terminalScheduler").TerminalWriteScheduler;

describe("useTerminalConnection · 404 self-heal", () => {
  beforeEach(() => {
    mocks.subscribe.mockReset();
    mocks.removePaneByPodKey.mockReset();
  });

  // Regression: 226 stale panes persisted in localStorage all tried to
  // connect, each producing an error-status spam. Now a definitive "pod
  // gone" response (HTTP 404 / RESOURCE_NOT_FOUND) removes the dead pane
  // from the store so subsequent renders stay clean.
  it("drops the pane when server returns Pod not found 404", async () => {
    mocks.subscribe.mockRejectedValueOnce(
      new Error("Error invoking remote method 'podGetPodConnection': Error: HTTP 404: Pod not found [RESOURCE_NOT_FOUND]"),
    );

    const setConnectionStatus = vi.fn();
    setupConnection("1-404-gone", scheduler, { current: null }, setConnectionStatus, vi.fn());
    await new Promise((r) => setTimeout(r, 0));

    expect(mocks.removePaneByPodKey).toHaveBeenCalledWith("1-404-gone");
    expect(setConnectionStatus).not.toHaveBeenCalledWith("error");
  });

  it("surfaces other errors as connection status 'error'", async () => {
    mocks.subscribe.mockRejectedValueOnce(new Error("network flaked"));

    const setConnectionStatus = vi.fn();
    setupConnection("1-live-abc", scheduler, { current: null }, setConnectionStatus, vi.fn());
    await new Promise((r) => setTimeout(r, 0));

    expect(mocks.removePaneByPodKey).not.toHaveBeenCalled();
    expect(setConnectionStatus).toHaveBeenCalledWith("error");
  });
});
