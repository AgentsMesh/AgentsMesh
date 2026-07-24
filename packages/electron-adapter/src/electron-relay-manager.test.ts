import { describe, expect, it, vi } from "vitest";
import { ElectronRelayManager } from "./electron-relay-manager";
import { installElectronRelayManagerFixture } from "./electron-relay-manager.fixture.test";

describe("ElectronRelayManager subscription coordination", () => {
  const { emitDisconnected, emitFromSubscribe, invoke } =
    installElectronRelayManagerFixture();

  it("delivers output only to the addressed subscription", async () => {
    const manager = new ElectronRelayManager();
    const first = vi.fn();
    const second = vi.fn();
    await manager.subscribe("pod-1", "sub-1", "wss://relay", "token", first);
    await manager.subscribe("pod-1", "sub-2", "wss://relay", "token", second);

    emitFromSubscribe(1, Uint8Array.of(2, 3));

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(2, 3));
  });

  it("ignores stale output after the target subscription is removed", async () => {
    const manager = new ElectronRelayManager();
    const callback = vi.fn();
    await manager.subscribe("pod-1", "sub-1", "wss://relay", "token", callback);
    await manager.unsubscribe("pod-1", "sub-1");

    emitFromSubscribe(0, Uint8Array.of(9));

    expect(callback).not.toHaveBeenCalled();
    expect(invoke).toHaveBeenLastCalledWith("relay:unsubscribe", "pod-1", "sub-1");
  });

  it("does not cross pods when subscription ids match", async () => {
    const manager = new ElectronRelayManager();
    const first = vi.fn();
    const second = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", first);
    await manager.subscribe("pod-2", "terminal", "wss://relay", "token", second);

    emitFromSubscribe(1, Uint8Array.of(7));

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(7));
  });

  it("restores the previous callback when replacement subscribe fails", async () => {
    const manager = new ElectronRelayManager();
    const previous = vi.fn();
    const replacement = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", previous);
    invoke.mockRejectedValueOnce(new Error("subscribe failed"));

    await expect(
      manager.subscribe("pod-1", "terminal", "wss://relay", "token", replacement),
    ).rejects.toThrow("subscribe failed");
    emitFromSubscribe(0, Uint8Array.of(5));

    expect(previous).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(5));
    expect(replacement).not.toHaveBeenCalled();
  });

  it("does not restore a superseded candidate when a newer replacement fails", async () => {
    const manager = new ElectronRelayManager();
    const active = vi.fn();
    const superseded = vi.fn();
    const failed = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", active);
    let resolveSuperseded!: (committed: boolean) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((resolve) => { resolveSuperseded = resolve; }),
    );

    const older = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", superseded,
    );
    invoke.mockRejectedValueOnce(new Error("newer subscribe failed"));
    await expect(
      manager.subscribe("pod-1", "terminal", "wss://relay", "token", failed),
    ).rejects.toThrow("newer subscribe failed");
    resolveSuperseded(false);
    await older;

    emitFromSubscribe(0, Uint8Array.of(6));
    expect(active).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(6));
    expect(superseded).not.toHaveBeenCalled();
    expect(failed).not.toHaveBeenCalled();
  });

  it("promotes an older committed candidate while a newer replacement is pending", async () => {
    const manager = new ElectronRelayManager();
    const initial = vi.fn();
    const committed = vi.fn();
    const failed = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", initial);
    let resolveCommitted!: (value: boolean) => void;
    let rejectNewer!: (error: Error) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((resolve) => { resolveCommitted = resolve; }),
    );
    const older = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", committed,
    );
    invoke.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectNewer = reject; }),
    );
    const newer = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", failed,
    );

    resolveCommitted(true);
    emitFromSubscribe(1, Uint8Array.of(7));
    await older;
    rejectNewer(new Error("newer subscribe failed"));
    await expect(newer).rejects.toThrow("newer subscribe failed");

    expect(committed).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(7));
    expect(initial).not.toHaveBeenCalled();
    expect(failed).not.toHaveBeenCalled();
  });

  it("buffers a candidate baseline until the matching subscribe commits", async () => {
    const manager = new ElectronRelayManager();
    const callback = vi.fn();
    let resolve!: (committed: boolean) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((done) => { resolve = done; }),
    );

    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", callback,
    );
    emitFromSubscribe(0, Uint8Array.of(1, 2, 3));
    expect(callback).not.toHaveBeenCalled();

    resolve(true);
    await pending;
    expect(callback).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(1, 2, 3));
  });

  it("waits for matching renderer output after the subscribe invoke resolves", async () => {
    const manager = new ElectronRelayManager();
    const callback = vi.fn();
    invoke.mockResolvedValueOnce(true);
    let settled = false;

    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", callback,
    ).then(() => { settled = true; });
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));
    await Promise.resolve();
    await Promise.resolve();
    expect(settled, "main-process commit alone must not publish readiness").toBe(false);

    emitFromSubscribe(0, new Uint8Array());
    await pending;
    expect(settled).toBe(true);
    expect(callback, "the empty baseline marker is not terminal output").not.toHaveBeenCalled();
  });

  it("settles a barrier wait when the renderer unsubscribes", async () => {
    const manager = new ElectronRelayManager();
    const callback = vi.fn();
    invoke.mockResolvedValueOnce(true);
    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", callback,
    );
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));
    await Promise.resolve();

    await manager.unsubscribe("pod-1", "terminal");
    await expect(pending).resolves.toBeUndefined();
    emitFromSubscribe(0, Uint8Array.of(9));
    expect(callback).not.toHaveBeenCalled();
  });

  it("settles the previous barrier wait when a replacement commits", async () => {
    const manager = new ElectronRelayManager();
    const previous = vi.fn();
    const replacement = vi.fn();
    invoke.mockResolvedValueOnce(true);
    const older = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", previous,
    );
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));
    await Promise.resolve();

    const newer = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", replacement,
    );
    await expect(newer).resolves.toBeUndefined();
    await expect(older).resolves.toBeUndefined();
    expect(previous).not.toHaveBeenCalled();
    expect(replacement).not.toHaveBeenCalled();
  });

  it("settles a superseded invoke when the newer replacement commits", async () => {
    const manager = new ElectronRelayManager();
    let resolveOlder!: (committed: boolean) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((resolve) => { resolveOlder = resolve; }),
    );
    const older = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", vi.fn(),
    );

    const newer = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", vi.fn(),
    );
    await expect(newer).resolves.toBeUndefined();
    await expect(older).resolves.toBeUndefined();

    // The in-flight invoke is still observed even though its caller was
    // settled locally, so a late response cannot become an unhandled promise.
    resolveOlder(false);
    await Promise.resolve();
  });

  it("ignores a late old-driver disconnect when a new output route exists", async () => {
    const manager = new ElectronRelayManager();
    const callback = vi.fn();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);
    let resolve!: (committed: boolean) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((done) => { resolve = done; }),
    );
    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", callback,
    );
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));

    emitDisconnected({ podKey: "pod-1", generation: 1 });
    expect(onDisconnected).not.toHaveBeenCalled();
    resolve(true);
    emitFromSubscribe(0, Uint8Array.of(9));
    await expect(pending).resolves.toBeUndefined();
    expect(callback).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(9));
    expect(onDisconnected).not.toHaveBeenCalled();
  });

  it("applies a deferred disconnect when the pending candidate fails", async () => {
    const manager = new ElectronRelayManager();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);
    let reject!: (error: Error) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((_resolve, fail) => { reject = fail; }),
    );
    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", vi.fn(),
    );
    const rejected = expect(pending).rejects.toThrow("subscribe failed");
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));

    emitDisconnected({ podKey: "pod-1", generation: 1 });
    expect(onDisconnected).not.toHaveBeenCalled();
    reject(new Error("subscribe failed"));

    await rejected;
    expect(onDisconnected).toHaveBeenCalledExactlyOnceWith("pod-1");
  });

  it("keeps the newest disconnect while a candidate is pending", async () => {
    const manager = new ElectronRelayManager();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);
    let reject!: (error: Error) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((_resolve, fail) => { reject = fail; }),
    );
    const pending = manager.subscribe(
      "pod-1", "terminal", "wss://relay", "token", vi.fn(),
    );
    const rejected = expect(pending).rejects.toThrow("subscribe failed");
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(1));

    emitDisconnected({ podKey: "pod-1", generation: 2 });
    emitDisconnected({ podKey: "pod-1", generation: 4 });
    emitDisconnected({ podKey: "pod-1", generation: 3 });
    reject(new Error("subscribe failed"));

    await rejected;
    expect(onDisconnected).toHaveBeenCalledExactlyOnceWith("pod-1");
  });

  it("defers disconnect until every pending route for the pod settles", async () => {
    const manager = new ElectronRelayManager();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);
    let rejectFirst!: (error: Error) => void;
    let rejectSecond!: (error: Error) => void;
    invoke
      .mockImplementationOnce(
        () => new Promise<boolean>((_resolve, fail) => { rejectFirst = fail; }),
      )
      .mockImplementationOnce(
        () => new Promise<boolean>((_resolve, fail) => { rejectSecond = fail; }),
      );
    const first = manager.subscribe("pod-1", "terminal", "wss://relay", "token", vi.fn());
    const second = manager.subscribe("pod-1", "acp-pane", "wss://relay", "token", vi.fn());
    const firstRejected = expect(first).rejects.toThrow("first failed");
    const secondRejected = expect(second).rejects.toThrow("second failed");
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(2));

    emitDisconnected({ podKey: "pod-1", generation: 2 });
    rejectFirst(new Error("first failed"));
    await firstRejected;
    expect(onDisconnected).not.toHaveBeenCalled();

    rejectSecond(new Error("second failed"));
    await secondRejected;
    expect(onDisconnected).toHaveBeenCalledExactlyOnceWith("pod-1");
  });

  it("discards a disconnect when another route for the pod is active", async () => {
    const manager = new ElectronRelayManager();
    const active = vi.fn();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", active);
    let reject!: (error: Error) => void;
    invoke.mockImplementationOnce(
      () => new Promise<boolean>((_resolve, fail) => { reject = fail; }),
    );
    const pending = manager.subscribe(
      "pod-1", "acp-pane", "wss://relay", "token", vi.fn(),
    );
    const rejected = expect(pending).rejects.toThrow("subscribe failed");
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(2));

    emitDisconnected({ podKey: "pod-1", generation: 1 });
    reject(new Error("subscribe failed"));
    await rejected;
    emitFromSubscribe(0, Uint8Array.of(7));

    expect(active).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(7));
    expect(onDisconnected).not.toHaveBeenCalled();
  });

  it("notifies when a driver disconnect arrives without an output route", async () => {
    const manager = new ElectronRelayManager();
    const onDisconnected = vi.fn();
    await manager.on_pod_disconnected(onDisconnected);

    emitDisconnected({ podKey: "pod-1", generation: 1 });

    expect(onDisconnected).toHaveBeenCalledExactlyOnceWith("pod-1");
  });

});
