import { ipcMain } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import { type WindowRegistry } from "./window_registry";

// Bridges the Rust `RelayConnectionPool` (terminal data plane SSOT, owned by the
// main process) to renderers. Renderer → main: `relay:*` invoke handlers map onto
// `appState.relay*` NAPI methods. Main → renderer: output/status/acp fan out via
// the registry, DIRECTED to only the webContents subscribed to that pod — PTY
// bytes are high-volume, never broadcast them to uninterested windows.
//
// The pool fans output to EVERY subscriber, so to keep ONE coalesced Rust link
// per pod (and avoid doubling when terminal + ACP share a pod) the bridge is the
// single pool subscriber per pod (`__bridge__`). `rendererSubs` is both the
// multicast routing table and the cross-window ref-count: a pod's Rust link lives
// iff ≥1 webContents subscribes it. Keyed by webContentsId so two windows viewing
// the same pod don't collide on the (podKey-derived) subId string. Every teardown
// path (unsubscribe / disconnect / disconnectAll / window close) is per-webContents
// so one window leaving never tears down a link another window still needs.

const BRIDGE_SUB = "__bridge__";

export interface RelayBridge {
  releaseWebContents: (wcId: number) => void;
  dispose: () => void;
}

export function setupRelayBridge(appState: AppState, registry: WindowRegistry): RelayBridge {
  const rendererSubs = new Map<string, Map<number, Set<string>>>();

  const sendToPod = (podKey: string, channel: string, payload: unknown) => {
    const byWc = rendererSubs.get(podKey);
    if (!byWc) return;
    for (const wcId of byWc.keys()) registry.sendTo(wcId, channel, payload);
  };

  const pending = new Map<string, number[]>();
  let flushScheduled = false;
  const scheduleFlush = () => {
    if (flushScheduled) return;
    flushScheduled = true;
    setImmediate(() => {
      flushScheduled = false;
      for (const [podKey, bytes] of pending) {
        sendToPod(podKey, "relay:output", { podKey, data: Uint8Array.from(bytes) });
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

  const subscribe = async (wcId: number, podKey: string, subId: string, url: string, token: string) => {
    const existed = rendererSubs.has(podKey);
    let byWc = rendererSubs.get(podKey);
    if (!byWc) {
      byWc = new Map();
      rendererSubs.set(podKey, byWc);
    }
    let subs = byWc.get(wcId);
    if (!subs) {
      subs = new Set();
      byWc.set(wcId, subs);
    }
    subs.add(subId);
    if (existed) {
      logEvent("debug", "relay", `subscribe ${podKey} (reuse, ${byWc.size} win)`);
      return;
    }
    logEvent("info", "relay", `subscribe ${podKey} (new link)`);
    await appState.relaySubscribe(podKey, BRIDGE_SUB, url, token, onOutput(podKey));
    // Wire status/ACP on every new link. The Rust pool replaces (not appends) the
    // per-pod listener, so re-wiring after a teardown→resubscribe stays a single
    // listener — no stale guard to desync, no double-fire.
    await appState.relayOnStatusChange(podKey, (_e: unknown, json: string) =>
      sendToPod(podKey, "relay:status", { podKey, json }),
    );
    await appState.relayOnAcpMessage(podKey, (_e: unknown, json: string) =>
      sendToPod(podKey, "relay:acp", { podKey, json }),
    );
  };

  // Returns true when this drop emptied the pod across ALL windows (caller tears
  // down the Rust link).
  const dropSub = (wcId: number, podKey: string, subId: string): boolean => {
    const byWc = rendererSubs.get(podKey);
    if (!byWc) return false;
    const subs = byWc.get(wcId);
    if (!subs) return false;
    subs.delete(subId);
    if (subs.size === 0) byWc.delete(wcId);
    if (byWc.size > 0) return false;
    rendererSubs.delete(podKey);
    return true;
  };

  const unsubscribe = (wcId: number, podKey: string, subId: string) => {
    if (!dropSub(wcId, podKey, subId)) {
      logEvent("debug", "relay", `unsubscribe ${podKey}`);
      return undefined;
    }
    logEvent("info", "relay", `unsubscribe ${podKey} (last, dropping link)`);
    return appState.relayUnsubscribe(podKey, BRIDGE_SUB);
  };

  // Drop a single window's hold on one pod. Only when no window subscribes the
  // pod anymore do we actually disconnect the Rust link — a renderer-driven
  // `relay:disconnect` must not tear down a pod another window still views.
  const disconnectPodForWc = (wcId: number, podKey: string) => {
    const byWc = rendererSubs.get(podKey);
    if (!byWc || !byWc.has(wcId)) return undefined;
    byWc.delete(wcId);
    if (byWc.size > 0) return undefined;
    rendererSubs.delete(podKey);
    return appState.relayDisconnect(podKey);
  };

  // A window closing (or its renderer's beforeunload) releases only ITS OWN
  // subscriptions and ref-counts each pod's Rust link down — never a global
  // teardown, which would starve every other live window's terminals.
  const releaseWebContents = (wcId: number) => {
    for (const [podKey, byWc] of [...rendererSubs]) {
      if (!byWc.has(wcId)) continue;
      byWc.delete(wcId);
      if (byWc.size === 0) {
        rendererSubs.delete(podKey);
        void appState.relayUnsubscribe(podKey, BRIDGE_SUB).catch((e: unknown) =>
          logEvent("warn", "relay", `relayUnsubscribe ${podKey} failed: ${e}`),
        );
      }
    }
  };

  ipcMain.handle("relay:subscribe", (e, podKey: string, subId: string, url: string, token: string) =>
    subscribe(e.sender.id, podKey, subId, url, token),
  );
  ipcMain.handle("relay:unsubscribe", (e, podKey: string, subId: string) =>
    unsubscribe(e.sender.id, podKey, subId),
  );
  ipcMain.handle("relay:disconnect", (e, podKey: string) => disconnectPodForWc(e.sender.id, podKey));
  // beforeunload fires this from the unloading window — release just its subs.
  ipcMain.handle("relay:disconnectAll", (e) => releaseWebContents(e.sender.id));

  const handlers: Record<string, (...args: never[]) => unknown> = {
    "relay:send": (podKey: string, data: string) => appState.relaySend(podKey, data),
    "relay:resize": (podKey: string, cols: number, rows: number) => appState.relaySendResize(podKey, cols, rows),
    "relay:forceResize": (podKey: string, cols: number, rows: number) => appState.relayForceResize(podKey, cols, rows),
    "relay:acpCommand": (podKey: string, command: string) => appState.relaySendAcpCommand(podKey, command),
    "relay:getStatus": (podKey: string) => appState.relayGetStatus(podKey),
    "relay:isRunnerDisconnected": (podKey: string) => appState.relayIsRunnerDisconnected(podKey),
    "relay:getPodSize": (podKey: string) => appState.relayGetPodSize(podKey),
  };
  for (const [channel, fn] of Object.entries(handlers)) {
    ipcMain.handle(channel, (_e, ...args) => (fn as (...a: unknown[]) => unknown)(...args));
  }

  void appState.relayOnPodDisconnected((_e: unknown, podKey: string) => {
    logEvent("info", "relay", `pod disconnected ${podKey}`);
    sendToPod(podKey, "relay:pod-disconnected", { podKey });
  });

  const allChannels = [
    "relay:subscribe", "relay:unsubscribe", "relay:disconnect", "relay:disconnectAll",
    ...Object.keys(handlers),
  ];
  return {
    releaseWebContents,
    dispose: () => {
      for (const channel of allChannels) ipcMain.removeHandler(channel);
      rendererSubs.clear();
      void appState.relayDisconnectAll().catch((e: unknown) =>
        logEvent("warn", "relay", `relayDisconnectAll failed: ${e}`),
      );
    },
  };
}
