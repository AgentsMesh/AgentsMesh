import { ipcMain, app, powerMonitor } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import { type WindowRegistry } from "./window_registry";
import { pushAllSnapshots } from "./realtime_snapshots";

// Bridges the backend EventBus realtime stream into renderers.
//
// Topology:
//   backend EventsService.Subscribe (Connect HTTP/2)
//     ↓ AppState.eventsSubscribeAll(jsonCallback) — Rust events crate, single
//       connection per AppState instance, owned by the main process.
//     ↓ registry.broadcast("realtime:event", json) — every window's
//       ElectronEventsManager needs the stream; each mirrors snapshots into its
//       own renderer-local cache.
//
// Routed through main (not a per-renderer Connect client) because the auth
// token + base URL + reconnect policy live in the Rust AppState; a parallel
// renderer client would duplicate auth state and fight org-switch races.

type RealtimeState = "disconnected" | "connecting" | "connected" | "reconnecting";

export interface RealtimeBridge {
  dispose: () => Promise<void>;
  currentState: () => RealtimeState;
  releaseWebContents: (wcId: number) => void;
}

export async function setupRealtimeBridge(
  appState: AppState,
  registry: WindowRegistry,
): Promise<RealtimeBridge> {
  let state: RealtimeState = "disconnected";
  const send = (channel: string, payload: unknown) => registry.broadcast(channel, payload);

  // The EventBus connection lives iff ≥1 window's RealtimeProvider is mounted.
  // connect() always runs (it re-targets the current org on switch); disconnect()
  // only tears down the shared stream when the LAST window leaves — otherwise
  // closing a popout would kill realtime for every other window.
  const connectedWcs = new Set<number>();
  const disconnectIfIdle = async (): Promise<void> => {
    // Prune ids whose window died without firing `closed` (e.g. renderer crash),
    // else a leaked id would pin the shared stream open forever.
    for (const id of connectedWcs) if (!registry.has(id)) connectedWcs.delete(id);
    if (connectedWcs.size > 0) return;
    logEvent("info", "realtime", "disconnect requested (last window)");
    await appState.eventsDisconnect();
  };

  const eventSubId = await appState.eventsSubscribeAll((_err: unknown, eventJson: string) => {
    send("realtime:event", eventJson);
    // Skip the (up to 5) Rust proto-encodes when no window is open to receive them.
    if (registry.hasWindows()) pushAllSnapshots(appState, eventJson, send);
  });
  logEvent("info", "realtime", "bridge subscribed");
  const stateSubId = await appState.eventsOnConnectionStateChange((_err: unknown, next: string) => {
    if (typeof next === "string" && next.length > 0) {
      if (next !== state) logEvent("info", "realtime", `state ${state} → ${next}`);
      state = next as RealtimeState;
    }
    send("realtime:state", next);
  });

  ipcMain.handle("realtime:connect", async (e) => {
    connectedWcs.add(e.sender.id);
    logEvent("info", "realtime", "connect requested");
    await appState.eventsConnect();
  });
  ipcMain.handle("realtime:disconnect", async (e) => {
    connectedWcs.delete(e.sender.id);
    await disconnectIfIdle();
  });
  ipcMain.handle("realtime:nudge", () => appState.eventsNudge());
  ipcMain.handle("realtime:getState", () => state);

  // Re-arm on laptop wake / app refocus: skip the Rust reconnect backoff.
  // nudge() is a no-op when already connected, so firing on every focus is safe.
  const reArm = () => {
    logEvent("debug", "realtime", "re-arm (resume/focus) → nudge");
    void appState.eventsNudge();
  };
  powerMonitor.on("resume", reArm);
  app.on("browser-window-focus", reArm);

  return {
    currentState: () => state,
    releaseWebContents: (wcId: number) => {
      connectedWcs.delete(wcId);
      void disconnectIfIdle();
    },
    dispose: async () => {
      logEvent("info", "realtime", "bridge disposed");
      for (const ch of ["realtime:connect", "realtime:disconnect", "realtime:nudge", "realtime:getState"])
        ipcMain.removeHandler(ch);
      powerMonitor.removeListener("resume", reArm);
      app.removeListener("browser-window-focus", reArm);
      connectedWcs.clear();
      try { await appState.eventsDisconnect(); } catch { /* best-effort */ }
      try { await appState.eventsUnsubscribe(eventSubId); } catch { /* best-effort */ }
      try { await appState.eventsUnsubscribe(stateSubId); } catch { /* best-effort */ }
    },
  };
}
