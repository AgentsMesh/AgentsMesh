import { describe, expect, it, vi } from "vitest";
import { ElectronRelayManager } from "./electron-relay-manager";
import { installElectronRelayManagerFixture } from "./electron-relay-manager.fixture.test";

describe("ElectronRelayManager events and cleanup", () => {
  const { emitAcp, emitDisconnected, emitFromSubscribe, emitStatus, invoke } =
    installElectronRelayManagerFixture();

  it("does not let a stale committed response unsubscribe a new generation", async () => {
    const manager = new ElectronRelayManager();
    let resolveStale!: (committed: boolean) => void;
    let resolveCurrent!: (committed: boolean) => void;
    invoke
      .mockImplementationOnce(
        () => new Promise<boolean>((resolve) => { resolveStale = resolve; }),
      )
      .mockResolvedValueOnce(undefined)
      .mockImplementationOnce(
        () => new Promise<boolean>((resolve) => { resolveCurrent = resolve; }),
      );

    const stale = manager.subscribe("pod-1", "acp-pane", "wss://relay", "token", vi.fn());
    await manager.unsubscribe("pod-1", "acp-pane");
    const current = manager.subscribe("pod-1", "acp-pane", "wss://relay", "token", vi.fn());

    await expect(stale).resolves.toBeUndefined();
    resolveStale(true);
    expect(invoke).toHaveBeenCalledTimes(3);
    expect(invoke.mock.calls[2][0]).toBe("relay:subscribe");

    resolveCurrent(true);
    emitFromSubscribe(2, new Uint8Array());
    await current;
  });

  it("drops output whose attempt identity is no longer active", async () => {
    const manager = new ElectronRelayManager();
    const first = vi.fn();
    const replacement = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", first);
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", replacement);

    emitFromSubscribe(0, Uint8Array.of(4));
    emitFromSubscribe(1, Uint8Array.of(8));

    expect(first).not.toHaveBeenCalled();
    expect(replacement).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(8));
  });

  it("fans valid status and ACP events out only to callbacks for their pod", async () => {
    const manager = new ElectronRelayManager();
    const status = vi.fn();
    const otherStatus = vi.fn();
    const acp = vi.fn();
    await manager.on_status_change("pod-1", status);
    await manager.on_status_change("pod-1", status);
    await manager.on_status_change("pod-2", otherStatus);
    await manager.on_acp_message("pod-1", acp);
    await manager.on_acp_message("pod-1", acp);

    emitStatus({
      podKey: "pod-1",
      json: '{"status":"connected","runnerDisconnected":false}',
    });
    emitAcp({
      podKey: "pod-1",
      json: '{"msgType":13,"payload":{"session":{"id":"session-1"}}}',
    });

    expect(status).toHaveBeenCalledExactlyOnceWith({
      status: "connected",
      runnerDisconnected: false,
    });
    expect(otherStatus).not.toHaveBeenCalled();
    expect(acp).toHaveBeenCalledExactlyOnceWith(13, { session: { id: "session-1" } });
  });

  it("rejects malformed status and ACP frames without invoking consumers", async () => {
    const manager = new ElectronRelayManager();
    const status = vi.fn();
    const acp = vi.fn();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    await manager.on_status_change("pod-1", status);
    await manager.on_acp_message("pod-1", acp);

    emitStatus({ podKey: "pod-1", json: "not-json" });
    emitAcp({ podKey: "pod-1", json: "{" });

    expect(status).not.toHaveBeenCalled();
    expect(acp).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalledWith("relay: malformed status frame for pod pod-1");
    expect(warn).toHaveBeenCalledWith("relay: malformed acp frame for pod pod-1");
    warn.mockRestore();
  });

  it("clears pod-scoped callbacks after the final driver disconnect", async () => {
    const manager = new ElectronRelayManager();
    const status = vi.fn();
    const acp = vi.fn();
    const disconnected = vi.fn();
    await manager.on_status_change("pod-1", status);
    await manager.on_acp_message("pod-1", acp);
    await manager.on_pod_disconnected(disconnected);

    emitDisconnected({ podKey: "pod-1", generation: 1 });
    emitStatus({
      podKey: "pod-1",
      json: '{"status":"connected","runnerDisconnected":false}',
    });
    emitAcp({ podKey: "pod-1", json: '{"msgType":13,"payload":{}}' });

    expect(disconnected).toHaveBeenCalledExactlyOnceWith("pod-1");
    expect(status).not.toHaveBeenCalled();
    expect(acp).not.toHaveBeenCalled();
  });

  it("restores an active output route when unsubscribe IPC fails", async () => {
    const manager = new ElectronRelayManager();
    const output = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", output);
    invoke.mockRejectedValueOnce(new Error("unsubscribe failed"));

    await expect(manager.unsubscribe("pod-1", "terminal")).rejects.toThrow(
      "unsubscribe failed",
    );
    emitFromSubscribe(0, Uint8Array.of(4, 2));

    expect(output).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(4, 2));
  });

  it("does not restore an old route over a replacement created during unsubscribe", async () => {
    const manager = new ElectronRelayManager();
    const oldOutput = vi.fn();
    const siblingOutput = vi.fn();
    const replacementOutput = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", oldOutput);
    await manager.subscribe("pod-1", "logs", "wss://relay", "token", siblingOutput);
    let rejectUnsubscribe!: (error: Error) => void;
    invoke.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectUnsubscribe = reject; }),
    );
    const removing = manager.unsubscribe("pod-1", "terminal");
    const removalRejected = expect(removing).rejects.toThrow("unsubscribe failed");
    await vi.waitFor(() => expect(invoke).toHaveBeenCalledTimes(3));

    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", replacementOutput);
    rejectUnsubscribe(new Error("unsubscribe failed"));
    await removalRejected;
    emitFromSubscribe(0, Uint8Array.of(1));
    emitFromSubscribe(1, Uint8Array.of(2));
    emitFromSubscribe(3, Uint8Array.of(3));

    expect(oldOutput).not.toHaveBeenCalled();
    expect(siblingOutput).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(2));
    expect(replacementOutput).toHaveBeenCalledExactlyOnceWith(Uint8Array.of(3));
  });

  it("treats removing an unknown route as an idempotent IPC operation", async () => {
    const manager = new ElectronRelayManager();

    await manager.unsubscribe("missing-pod", "missing-sub");

    expect(invoke).toHaveBeenCalledExactlyOnceWith(
      "relay:unsubscribe",
      "missing-pod",
      "missing-sub",
    );
  });

  it("forwards commands and normalizes pod size replies", async () => {
    const manager = new ElectronRelayManager();
    await manager.send("pod-1", "input");
    await manager.send_resize("pod-1", 100, 40);
    await manager.force_resize("pod-1", 120, 50);
    await manager.send_acp_command("pod-1", '{"type":"cancel"}');
    invoke.mockResolvedValueOnce("connected");
    await expect(manager.get_status("pod-1")).resolves.toBe("connected");
    invoke.mockResolvedValueOnce(true);
    await expect(manager.is_runner_disconnected("pod-1")).resolves.toBe(true);
    invoke.mockResolvedValueOnce([80, 24]);
    await expect(manager.get_pod_size("pod-1")).resolves.toEqual({ cols: 80, rows: 24 });
    invoke.mockResolvedValueOnce([80]);
    await expect(manager.get_pod_size("pod-1")).resolves.toBeNull();

    expect(invoke.mock.calls.slice(0, 4)).toEqual([
      ["relay:send", "pod-1", "input"],
      ["relay:resize", "pod-1", 100, 40],
      ["relay:forceResize", "pod-1", 120, 50],
      ["relay:acpCommand", "pod-1", '{"type":"cancel"}'],
    ]);
  });

  it("drops pod and global output routes before invoking disconnect", async () => {
    const manager = new ElectronRelayManager();
    const first = vi.fn();
    const second = vi.fn();
    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", first);
    await manager.disconnect("pod-1");
    emitFromSubscribe(0, Uint8Array.of(1));

    await manager.subscribe("pod-1", "terminal", "wss://relay", "token", first);
    await manager.subscribe("pod-2", "terminal", "wss://relay", "token", second);
    await manager.disconnect_all();
    emitFromSubscribe(2, Uint8Array.of(2));

    expect(first).not.toHaveBeenCalled();
    expect(second).not.toHaveBeenCalled();
    expect(invoke).toHaveBeenCalledWith("relay:disconnect", "pod-1");
    expect(invoke).toHaveBeenCalledWith("relay:disconnectAll");
  });

  it("constructs safely when the Electron push API is unavailable", () => {
    (globalThis as { window?: unknown }).window = {};

    expect(() => new ElectronRelayManager()).not.toThrow();
  });
});
