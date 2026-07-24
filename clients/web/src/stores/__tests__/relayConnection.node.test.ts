import { describe, expect, it, vi } from "vitest";

const manager = vi.hoisted(() => ({
  disconnect_all: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@agentsmesh/service-runtime", () => ({ getRelayManager: () => manager }));
vi.mock("../relayEndpointSelection", () => ({
  selectRelayEndpoint: vi.fn().mockResolvedValue({ url: "wss://relay", token: "token" }),
}));

describe("relayConnection outside a browser", () => {
  it("constructs without installing the browser unload hook", async () => {
    const browserWindow = window;
    vi.stubGlobal("window", undefined);
    delete (globalThis as Record<string, unknown>).__relayPool;
    vi.resetModules();

    const { relayPool } = await import("../relayConnection");

    expect(relayPool.getStatus("unknown")).toBe("none");
    vi.stubGlobal("window", browserWindow);
    vi.unstubAllGlobals();
  });
});
