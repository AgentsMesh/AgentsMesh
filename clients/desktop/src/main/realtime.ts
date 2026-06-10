import { BrowserWindow, ipcMain, app, powerMonitor } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";

// Bridges the backend EventBus realtime stream into the Electron renderer.
//
// Topology:
//   backend EventsService.Subscribe (Connect HTTP/2)
//     ↓
//   AppState.eventsSubscribeAll(jsonCallback) — Rust events crate, owned by
//   main process (single connection per AppState instance, lives until
//   AppState is replaced via server switch).
//     ↓
//   webContents.send("realtime:event", eventJson)
//     ↓
//   preload `onRealtimeEvent` listener → renderer ElectronEventsManager →
//   wasm-shaped EventSubscriptionManager → handler dispatch.
//
// Why route through main and not stream Connect directly from renderer:
// the auth token + base URL + reconnect policy already live in the Rust
// AppState. Spawning a parallel renderer-side Connect client would duplicate
// auth state, fight over org-switch races, and require shipping the Rust
// events crate via wasm (the whole reason desktop exists separately).

type RealtimeState = "disconnected" | "connecting" | "connected" | "reconnecting";

const CHANNEL_MESSAGE_EVENTS = new Set([
  "channel:message",
  "channel:message_edited",
  "channel:message_deleted",
]);

const POD_EVENTS = new Set([
  "pod:status_changed",
  "pod:agent_status_changed",
  "pod:terminated",
  "pod:title_changed",
  "pod:alias_changed",
  "pod:perpetual_changed",
]);

const RUNNER_EVENTS = new Set([
  "runner:online",
  "runner:offline",
  "runner:updated",
]);

const AUTOPILOT_EVENTS = new Set([
  "autopilot:status_changed",
  "autopilot:iteration",
  "autopilot:thinking",
  "autopilot:terminated",
]);

// status/agent patch a mesh node in place; structural changes (created/terminated)
// go through the renderer's fetchTopology, not here.
const MESH_NODE_EVENTS = new Set([
  "pod:status_changed",
  "pod:agent_status_changed",
]);

// After a channel message event, read the post-dispatch runtime.state channel
// snapshot (messages + unread + mention, all Rust-computed) and push it to the
// renderer over a dedicated IPC channel. The renderer mirrors it into its
// ElectronChannelService cache — that cache is renderer-local and, unlike web's
// wasm, is NOT the same memory the dispatch wrote, so a push is the only way
// the SSOT result reaches the view.
function pushChannelSnapshot(
  appState: AppState,
  eventJson: string,
  send: (channel: string, payload: unknown) => void,
): void {
  let ev: { type?: string; data?: Record<string, unknown> };
  try {
    ev = JSON.parse(eventJson) as { type?: string; data?: Record<string, unknown> };
  } catch {
    return;
  }
  if (!ev.type || !CHANNEL_MESSAGE_EVENTS.has(ev.type)) return;
  const rawId = ev.data?.channel_id ?? ev.data?.channelId;
  const channelId = Number(rawId);
  if (!Number.isFinite(channelId) || channelId <= 0) return;
  try {
    send("realtime:state-sync", JSON.stringify({
      domain: "channel",
      channelId,
      messages: appState.appChannelMessagesJson(channelId),
      unreadCounts: appState.appChannelUnreadCountsJson(),
      mentionCounts: appState.appChannelMentionCountsJson(),
    }));
  } catch {
    /* best-effort: a stale window or lock contention must not break forwarding */
  }
}

// Surgical pod snapshot: read the single Rust-computed pod from runtime.state
// and push it. The renderer upserts it in-place only if already cached (the
// pod sidebar is filtered — a brand-new pod arrives via the handler's
// fetchPod refetch, not this mirror). Empty pod → nothing to mirror.
function pushPodSnapshot(
  appState: AppState,
  eventJson: string,
  send: (channel: string, payload: unknown) => void,
): void {
  let ev: { type?: string; data?: Record<string, unknown> };
  try {
    ev = JSON.parse(eventJson) as { type?: string; data?: Record<string, unknown> };
  } catch {
    return;
  }
  if (!ev.type || !POD_EVENTS.has(ev.type)) return;
  const rawKey = ev.data?.pod_key ?? ev.data?.podKey;
  if (typeof rawKey !== "string" || rawKey.length === 0) return;
  try {
    const pod = appState.appGetPodJson(rawKey);
    if (pod) send("realtime:state-sync", JSON.stringify({ domain: "pod", podKey: rawKey, pod }));
  } catch {
    /* best-effort */
  }
}

// Runner online/offline/updated → push the Rust-computed runner lists. The
// renderer replaces its three caches (runners + available + current). Runner
// realtime has no refetch fallback on desktop, so this mirror is what keeps
// the runner views live after the JS pure-patch is removed.
function pushRunnerSnapshot(
  appState: AppState,
  eventJson: string,
  send: (channel: string, payload: unknown) => void,
): void {
  let ev: { type?: string };
  try {
    ev = JSON.parse(eventJson) as { type?: string };
  } catch {
    return;
  }
  if (!ev.type || !RUNNER_EVENTS.has(ev.type)) return;
  try {
    send("realtime:state-sync", JSON.stringify({
      domain: "runner",
      runners: appState.appRunnersJson(),
      available: appState.appAvailableRunnersJson(),
      current: appState.appCurrentRunnerJson(),
    }));
  } catch {
    /* best-effort */
  }
}

// Autopilot status/iteration/thinking/terminated → push the Rust-computed
// controller list + the affected key's iterations + thinking + history. The
// renderer mirrors into its per-key caches. autopilot:created stays on the
// handler's refetch (full payload from server).
function pushAutopilotSnapshot(
  appState: AppState,
  eventJson: string,
  send: (channel: string, payload: unknown) => void,
): void {
  let ev: { type?: string; data?: Record<string, unknown> };
  try {
    ev = JSON.parse(eventJson) as { type?: string; data?: Record<string, unknown> };
  } catch {
    return;
  }
  if (!ev.type || !AUTOPILOT_EVENTS.has(ev.type)) return;
  const rawKey = ev.data?.autopilot_controller_key ?? ev.data?.autopilotControllerKey;
  const key = typeof rawKey === "string" ? rawKey : "";
  try {
    send("realtime:state-sync", JSON.stringify({
      domain: "autopilot",
      key,
      controllers: appState.appAutopilotControllersJson(),
      iterations: key ? appState.appAutopilotIterationsJson(key) : "",
      thinking: key ? appState.appAutopilotThinkingJson(key) : "",
      thinkingHistory: key ? appState.appAutopilotThinkingHistoryJson(key) : "",
    }));
  } catch {
    /* best-effort */
  }
}

// Read the Rust-patched mesh node and push it for the renderer to patch its
// topology cache in place. Empty node → not a topology node, skip.
function pushMeshNodeSnapshot(
  appState: AppState,
  eventJson: string,
  send: (channel: string, payload: unknown) => void,
): void {
  let ev: { type?: string; data?: Record<string, unknown> };
  try {
    ev = JSON.parse(eventJson) as { type?: string; data?: Record<string, unknown> };
  } catch {
    return;
  }
  if (!ev.type || !MESH_NODE_EVENTS.has(ev.type)) return;
  const rawKey = ev.data?.pod_key ?? ev.data?.podKey;
  if (typeof rawKey !== "string" || rawKey.length === 0) return;
  try {
    const node = appState.appGetMeshNodeJson(rawKey);
    if (node) send("realtime:state-sync", JSON.stringify({ domain: "mesh", podKey: rawKey, node }));
  } catch {
    /* best-effort */
  }
}

export interface RealtimeBridge {
  // Tear down the subscription, forwarder, and IPC handlers. Required
  // before swapping AppState (server-switch flow) so the old stream's
  // callback closure doesn't keep firing into a stale window.
  dispose: () => Promise<void>;
  currentState: () => RealtimeState;
}

export async function setupRealtimeBridge(
  appState: AppState,
  getMainWindow: () => BrowserWindow | null,
): Promise<RealtimeBridge> {
  let state: RealtimeState = "disconnected";

  const send = (channel: string, payload: unknown) => {
    const win = getMainWindow();
    if (!win || win.isDestroyed()) return;
    win.webContents.send(channel, payload);
  };

  // napi-rs ThreadsafeFunction<T> defaults to CalleeHandled=true, so
  // the JS callback signature is `(err, value)` — not `(value)`. The
  // first argument is null on success; the actual payload is the second.
  const eventSubId = await appState.eventsSubscribeAll((_err: unknown, eventJson: string) => {
    send("realtime:event", eventJson);
    // The dispatch hook has already mutated runtime.state by the time this
    // external subscriber fires (hook runs before subscribers). Read the
    // Rust-computed channel snapshot and push it so the renderer — which has
    // no in-process Rust — can mirror unread/mention/preview the SSOT derived.
    pushChannelSnapshot(appState, eventJson, send);
    pushPodSnapshot(appState, eventJson, send);
    pushRunnerSnapshot(appState, eventJson, send);
    pushAutopilotSnapshot(appState, eventJson, send);
    pushMeshNodeSnapshot(appState, eventJson, send);
  });
  logEvent("info", "realtime", "bridge subscribed");
  const stateSubId = await appState.eventsOnConnectionStateChange((_err: unknown, next: string) => {
    if (typeof next === "string" && next.length > 0) {
      if (next !== state) logEvent("info", "realtime", `state ${state} → ${next}`);
      state = next as RealtimeState;
    }
    send("realtime:state", next);
  });

  const connectHandler = async (): Promise<void> => {
    logEvent("info", "realtime", "connect requested");
    await appState.eventsConnect();
  };
  const disconnectHandler = async (): Promise<void> => {
    logEvent("info", "realtime", "disconnect requested");
    await appState.eventsDisconnect();
  };
  const getStateHandler = (): RealtimeState => state;

  ipcMain.handle("realtime:connect", () => connectHandler());
  ipcMain.handle("realtime:disconnect", () => disconnectHandler());
  ipcMain.handle("realtime:nudge", () => appState.eventsNudge());
  ipcMain.handle("realtime:getState", () => getStateHandler());

  // Re-arm on laptop wake / app refocus: skip the Rust reconnect backoff and
  // retry now. nudge() is a no-op when already connected, so firing it on every
  // focus is harmless. powerMonitor covers system resume the renderer can't see.
  const reArm = () => {
    logEvent("debug", "realtime", "re-arm (resume/focus) → nudge");
    void appState.eventsNudge();
  };
  powerMonitor.on("resume", reArm);
  app.on("browser-window-focus", reArm);

  return {
    currentState: () => state,
    dispose: async () => {
      logEvent("info", "realtime", "bridge disposed");
      ipcMain.removeHandler("realtime:connect");
      ipcMain.removeHandler("realtime:disconnect");
      ipcMain.removeHandler("realtime:nudge");
      ipcMain.removeHandler("realtime:getState");
      powerMonitor.removeListener("resume", reArm);
      app.removeListener("browser-window-focus", reArm);
      try { await appState.eventsDisconnect(); } catch { /* best-effort */ }
      try { await appState.eventsUnsubscribe(eventSubId); } catch { /* best-effort */ }
      try { await appState.eventsUnsubscribe(stateSubId); } catch { /* best-effort */ }
    },
  };
}
