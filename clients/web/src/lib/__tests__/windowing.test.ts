import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  openTerminalWindow,
  openChannelWindow,
  queryIsPrimaryWindow,
  isOutsideCurrentWindow,
} from "../windowing";

type WinWithApi = { electronAPI?: { invoke?: (c: string, ...a: unknown[]) => Promise<unknown> } };
const setApi = (api: WinWithApi["electronAPI"] | undefined) => {
  (window as unknown as WinWithApi).electronAPI = api;
};

afterEach(() => {
  setApi(undefined);
  vi.restoreAllMocks();
});

describe("windowing · electron path (electronAPI.invoke present)", () => {
  let invoke: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    invoke = vi.fn().mockResolvedValue(undefined);
    setApi({ invoke });
  });

  it("openTerminalWindow routes through window:openTerminal IPC", () => {
    openTerminalWindow("pod-1");
    expect(invoke).toHaveBeenCalledWith("window:openTerminal", "pod-1");
  });

  it("openChannelWindow routes through window:open with a stringified id", () => {
    openChannelWindow(42);
    expect(invoke).toHaveBeenCalledWith("window:open", "channel", { id: "42" });
  });

  it("openChannelWindow ignores a non-positive / non-finite id", () => {
    openChannelWindow(0);
    openChannelWindow(-1);
    openChannelWindow(Number.NaN);
    expect(invoke).not.toHaveBeenCalled();
  });

  it("queryIsPrimaryWindow asks main and coerces the result", async () => {
    invoke.mockResolvedValue(true);
    await expect(queryIsPrimaryWindow()).resolves.toBe(true);
    expect(invoke).toHaveBeenCalledWith("window:isPrimary");
  });

  it("queryIsPrimaryWindow resolves false when the invoke rejects", async () => {
    invoke.mockRejectedValue(new Error("ipc down"));
    await expect(queryIsPrimaryWindow()).resolves.toBe(false);
  });
});

describe("windowing · web fallback (no electronAPI)", () => {
  let open: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    setApi(undefined);
    open = vi.fn();
    Object.defineProperty(window, "open", { value: open, configurable: true, writable: true });
  });

  it("openTerminalWindow opens a browser popout", () => {
    openTerminalWindow("pod-1");
    expect(open).toHaveBeenCalledWith("/popout/terminal/pod-1", "terminal-pod-1", expect.any(String));
  });

  it("openChannelWindow opens a browser popout", () => {
    openChannelWindow(7);
    expect(open).toHaveBeenCalledWith("/popout/channel/7", "channel-7", expect.any(String));
  });

  it("queryIsPrimaryWindow resolves true (the sole tab owns notifications)", async () => {
    await expect(queryIsPrimaryWindow()).resolves.toBe(true);
  });
});

describe("isOutsideCurrentWindow (content rect = origin 100,100 size 800x600)", () => {
  beforeEach(() => {
    Object.defineProperty(window, "screenX", { value: 100, configurable: true });
    Object.defineProperty(window, "screenY", { value: 100, configurable: true });
    Object.defineProperty(window, "innerWidth", { value: 800, configurable: true });
    Object.defineProperty(window, "innerHeight", { value: 600, configurable: true });
  });

  it("a pointer inside the rect → false", () => {
    expect(isOutsideCurrentWindow(400, 400)).toBe(false);
  });

  it("left of / above the origin → true", () => {
    expect(isOutsideCurrentWindow(50, 400)).toBe(true);
    expect(isOutsideCurrentWindow(400, 50)).toBe(true);
  });

  it("right of / below the rect → true", () => {
    expect(isOutsideCurrentWindow(950, 400)).toBe(true);
    expect(isOutsideCurrentWindow(400, 750)).toBe(true);
  });
});
