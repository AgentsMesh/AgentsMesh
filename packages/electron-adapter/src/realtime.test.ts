import { describe, it, expect, vi, afterEach } from "vitest";
import { ElectronEventsManager } from "./realtime";

type Handler = (s: string) => void;
const flush = () => new Promise((r) => setTimeout(r, 0));

let stateHandler: Handler | undefined;

function stubApi(getStateValue: unknown = "disconnected") {
  stateHandler = undefined;
  const invoke = vi.fn((channel: string) =>
    channel === "realtime:getState" ? Promise.resolve(getStateValue) : Promise.resolve(undefined),
  );
  (globalThis as { window?: unknown }).window = {
    electronAPI: {
      invoke,
      onRealtimeEvent: () => () => {},
      onRealtimeState: (h: Handler) => {
        stateHandler = h;
        return () => {};
      },
    },
  };
  return invoke;
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window;
  vi.restoreAllMocks();
});

describe("ElectronEventsManager · onRealtimeState guard (#4)", () => {
  it("ignores empty/non-string frames so they don't clobber a good badge", async () => {
    stubApi();
    const mgr = new ElectronEventsManager();
    const seen: string[] = [];
    await mgr.on_connection_state_change((s) => seen.push(s));
    await flush(); // drain the initial 'disconnected' replay + getState seed
    seen.length = 0;

    stateHandler!("connected");
    expect(seen).toEqual(["connected"]);

    stateHandler!(""); // empty frame ignored
    expect(seen).toEqual(["connected"]);
  });
});

describe("ElectronEventsManager · getState seed (#3)", () => {
  it("seeds a window opened after the stream already connected", async () => {
    stubApi("connected");
    const mgr = new ElectronEventsManager();
    const seen: string[] = [];
    await mgr.on_connection_state_change((s) => seen.push(s));
    await flush();
    expect(seen).toContain("connected");
  });

  it("does NOT regress state once a real broadcast already landed", async () => {
    stubApi("connecting"); // getState would resolve a now-stale value
    const mgr = new ElectronEventsManager();
    const seen: string[] = [];
    await mgr.on_connection_state_change((s) => seen.push(s));

    stateHandler!("connected"); // authoritative broadcast lands first
    await flush(); // getState seed resolves now

    expect(seen).toContain("connected");
    expect(seen).not.toContain("connecting");
  });
});
