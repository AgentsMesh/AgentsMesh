import { BrowserWindow, ipcMain } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";

// Bridges the Rust `RelayConnectionPool` (terminal data plane SSOT, owned by the
// main process) to the renderer. Renderer → main: `relay:*` invoke handlers map
// onto the `appState.relay*` NAPI methods. Main → renderer: the pool's
// output/status/acp callbacks fan out via `webContents.send`.
//
// The pool fans output to EVERY subscriber. To keep ONE coalesced IPC stream
// per pod (and avoid doubling when terminal + ACP share a pod), the bridge is
// itself the single pool subscriber per pod (`__bridge__`); the renderer's
// ElectronRelayManager re-fans the stream to its per-subId callbacks. Renderer
// subIds drive ref-counting only — when the last one unsubscribes the bridge
// drops `__bridge__`, letting the pool's grace timer disconnect.

const BRIDGE_SUB = "__bridge__";

export interface RelayBridge {
  dispose: () => void;
}

export function setupRelayBridge(
  appState: AppState,
  getMainWindow: () => BrowserWindow | null,
): RelayBridge {
  const send = (channel: string, payload: unknown) => {
    const win = getMainWindow();
    if (!win || win.isDestroyed()) return;
    win.webContents.send(channel, payload);
  };

  // Per-pod output coalescing: accumulate frames, flush once on the next tick.
  const pending = new Map<string, number[]>();
  let flushScheduled = false;
  const scheduleFlush = () => {
    if (flushScheduled) return;
    flushScheduled = true;
    setImmediate(() => {
      flushScheduled = false;
      for (const [podKey, bytes] of pending) {
        send("relay:output", { podKey, data: Uint8Array.from(bytes) });
      }
      pending.clear();
    });
  };
  const onOutput = (podKey: string) => (_err: unknown, bytes: number[]) => {
    const buf = pending.get(podKey);
    if (buf) for (const b of bytes) buf.push(b);
    else pending.set(podKey, [...bytes]);
    scheduleFlush();
  };

  const rendererSubs = new Map<string, Set<string>>();
  // status/acp listeners are pod-scoped and survive reconnects (the pool keeps
  // them keyed by pod, and has no per-listener removal) — wire them once.
  const listenersWired = new Set<string>();

  const handlers: Record<string, (...args: never[]) => unknown> = {
    "relay:subscribe": async (podKey: string, subId: string, url: string, token: string) => {
      const subs = rendererSubs.get(podKey);
      if (subs) {
        subs.add(subId);
        logEvent("debug", "relay", `subscribe ${podKey} (reuse, ${subs.size} subs)`);
        return;
      }
      rendererSubs.set(podKey, new Set([subId]));
      logEvent("info", "relay", `subscribe ${podKey} (new link)`);
      await appState.relaySubscribe(podKey, BRIDGE_SUB, url, token, onOutput(podKey));
      if (!listenersWired.has(podKey)) {
        listenersWired.add(podKey);
        await appState.relayOnStatusChange(podKey, (_e: unknown, json: string) =>
          send("relay:status", { podKey, json }),
        );
        await appState.relayOnAcpMessage(podKey, (_e: unknown, json: string) =>
          send("relay:acp", { podKey, json }),
        );
      }
    },
    "relay:unsubscribe": (podKey: string, subId: string) => {
      const subs = rendererSubs.get(podKey);
      if (!subs) return undefined;
      subs.delete(subId);
      if (subs.size > 0) {
        logEvent("debug", "relay", `unsubscribe ${podKey} (${subs.size} left)`);
        return undefined;
      }
      rendererSubs.delete(podKey);
      logEvent("info", "relay", `unsubscribe ${podKey} (last, dropping link)`);
      return appState.relayUnsubscribe(podKey, BRIDGE_SUB);
    },
    "relay:send": (podKey: string, data: string) => appState.relaySend(podKey, data),
    "relay:resize": (podKey: string, cols: number, rows: number) => appState.relaySendResize(podKey, cols, rows),
    "relay:forceResize": (podKey: string, cols: number, rows: number) => appState.relayForceResize(podKey, cols, rows),
    "relay:acpCommand": (podKey: string, command: string) => appState.relaySendAcpCommand(podKey, command),
    "relay:disconnect": (podKey: string) => appState.relayDisconnect(podKey),
    "relay:disconnectAll": () => appState.relayDisconnectAll(),
    "relay:getStatus": (podKey: string) => appState.relayGetStatus(podKey),
    "relay:isRunnerDisconnected": (podKey: string) => appState.relayIsRunnerDisconnected(podKey),
    "relay:getPodSize": (podKey: string) => appState.relayGetPodSize(podKey),
  };

  for (const [channel, fn] of Object.entries(handlers)) {
    ipcMain.handle(channel, (_e, ...args) => (fn as (...a: unknown[]) => unknown)(...args));
  }

  // Single pod-disconnected sink: the pool fires this once a pod's connection
  // is fully torn down (having already cleared that pod's status/ACP listeners).
  // Reset the per-pod wired guard so the next subscribe re-registers them, and
  // tell the renderer's mirror to drop its guards too.
  void appState.relayOnPodDisconnected((_e: unknown, podKey: string) => {
    listenersWired.delete(podKey);
    logEvent("info", "relay", `pod disconnected ${podKey}`);
    send("relay:pod-disconnected", { podKey });
  });

  return {
    dispose: () => {
      for (const channel of Object.keys(handlers)) ipcMain.removeHandler(channel);
      rendererSubs.clear();
      listenersWired.clear();
      void appState.relayDisconnectAll().catch(() => undefined);
    },
  };
}
