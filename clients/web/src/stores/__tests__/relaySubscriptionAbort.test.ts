import { describe, expect, it, vi } from "vitest";
import { waitForRelayWithAbort } from "../relaySubscriptionAbort";

describe("waitForRelayWithAbort", () => {
  it("rejects when the signal was aborted before listener registration", async () => {
    const controller = new AbortController();
    controller.abort();
    let rejectPending!: (reason: unknown) => void;
    const pending = new Promise<never>((_resolve, reject) => { rejectPending = reject; });

    await expect(waitForRelayWithAbort(pending, controller.signal))
      .rejects.toMatchObject({ name: "AbortError" });

    rejectPending(new Error("late endpoint failure"));
    await Promise.resolve();
  });

  it("rejects immediately when abort follows registration", async () => {
    const controller = new AbortController();
    const pending = new Promise<never>(() => {});
    const waiting = waitForRelayWithAbort(pending, controller.signal);

    controller.abort();

    await expect(waiting).rejects.toMatchObject({ name: "AbortError" });
  });

  it("resolves with the endpoint and removes the abort listener", async () => {
    const controller = new AbortController();
    const remove = vi.spyOn(controller.signal, "removeEventListener");

    await expect(
      waitForRelayWithAbort(Promise.resolve({ url: "wss://relay" }), controller.signal),
    ).resolves.toEqual({ url: "wss://relay" });
    expect(remove).toHaveBeenCalledWith("abort", expect.any(Function));
  });

  it("propagates an endpoint rejection before abort", async () => {
    const controller = new AbortController();
    const error = new Error("endpoint unavailable");

    await expect(
      waitForRelayWithAbort(Promise.reject(error), controller.signal),
    ).rejects.toBe(error);
  });

  it("defends against an abort racing listener registration", async () => {
    let abortedReads = 0;
    const signal = {
      get aborted() {
        abortedReads += 1;
        return abortedReads > 1;
      },
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    } as unknown as AbortSignal;

    await expect(
      waitForRelayWithAbort(Promise.resolve("late"), signal),
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(signal.addEventListener).toHaveBeenCalledWith(
      "abort",
      expect.any(Function),
      { once: true },
    );
  });
});
