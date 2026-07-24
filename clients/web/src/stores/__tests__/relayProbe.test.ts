import { afterEach, describe, expect, it, vi } from "vitest";

import { probeRelayOpen } from "../relayProbe";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  readonly url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  close = vi.fn();

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }
}

function socket(): FakeWebSocket {
  return FakeWebSocket.instances.at(-1)!;
}

describe("probeRelayOpen", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    FakeWebSocket.instances = [];
  });

  it("returns false when WebSocket is unavailable", async () => {
    vi.stubGlobal("WebSocket", undefined);

    await expect(probeRelayOpen("ws://relay", "token", 100)).resolves.toBe(false);
  });

  it("encodes the token, resolves true on open, and ignores later events", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const pending = probeRelayOpen("ws://relay", "a token&scope=admin", 100);

    expect(socket().url).toBe(
      "ws://relay/browser/relay?token=a%20token%26scope%3Dadmin",
    );
    socket().onopen?.();
    socket().onerror?.();
    socket().onclose?.();

    await expect(pending).resolves.toBe(true);
    expect(socket().close).toHaveBeenCalledOnce();
  });

  it.each(["error", "close"] as const)("resolves false on %s", async (event) => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const pending = probeRelayOpen("ws://relay", "token", 100);

    if (event === "error") socket().onerror?.();
    else socket().onclose?.();

    await expect(pending).resolves.toBe(false);
    expect(socket().close).toHaveBeenCalledOnce();
  });

  it("times out and closes a relay that never settles", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const pending = probeRelayOpen("ws://relay", "token", 25);

    await vi.advanceTimersByTimeAsync(25);

    await expect(pending).resolves.toBe(false);
    expect(socket().close).toHaveBeenCalledOnce();
  });

  it("still resolves when closing the probe socket throws", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const pending = probeRelayOpen("ws://relay", "token", 100);
    socket().close.mockImplementation(() => {
      throw new Error("already closed");
    });

    socket().onopen?.();

    await expect(pending).resolves.toBe(true);
  });
});
