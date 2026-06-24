import { type AppState } from "@agentsmesh/node-bridge";

// Surgical runtime.state snapshot pushes. After each EventBus dispatch the Rust
// hook has already mutated runtime.state; we read the Rust-computed result and
// push it so renderers — which have no in-process Rust — can mirror the SSOT
// into their renderer-local service caches. Broadcast to every window: each
// mirrors into its own cache.

type Send = (channel: string, payload: unknown) => void;

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

const RUNNER_EVENTS = new Set(["runner:online", "runner:offline", "runner:updated"]);

const AUTOPILOT_EVENTS = new Set([
  "autopilot:status_changed",
  "autopilot:iteration",
  "autopilot:thinking",
  "autopilot:terminated",
]);

// status/agent patch a mesh node in place; structural changes (created/terminated)
// go through the renderer's fetchTopology, not here.
const MESH_NODE_EVENTS = new Set(["pod:status_changed", "pod:agent_status_changed"]);

function parseEvent(eventJson: string): { type?: string; data?: Record<string, unknown> } | null {
  try {
    return JSON.parse(eventJson) as { type?: string; data?: Record<string, unknown> };
  } catch {
    return null;
  }
}

export function pushAllSnapshots(appState: AppState, eventJson: string, send: Send): void {
  const ev = parseEvent(eventJson);
  if (!ev?.type) return;
  pushChannelSnapshot(appState, ev, send);
  pushPodSnapshot(appState, ev, send);
  pushRunnerSnapshot(appState, ev, send);
  pushAutopilotSnapshot(appState, ev, send);
  pushMeshNodeSnapshot(appState, ev, send);
}

type Ev = { type?: string; data?: Record<string, unknown> };

function pushChannelSnapshot(appState: AppState, ev: Ev, send: Send): void {
  if (!ev.type || !CHANNEL_MESSAGE_EVENTS.has(ev.type)) return;
  const rawId = ev.data?.channel_id ?? ev.data?.channelId;
  if (rawId == null) return;
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

// Surgical pod snapshot: read the single Rust-computed pod and push it. The
// renderer upserts in place only if already cached (a brand-new pod arrives via
// the handler's fetchPod refetch, not this mirror). Empty pod → nothing to do.
function pushPodSnapshot(appState: AppState, ev: Ev, send: Send): void {
  if (!ev.type || !POD_EVENTS.has(ev.type)) return;
  const rawKey = ev.data?.pod_key ?? ev.data?.podKey;
  if (typeof rawKey !== "string" || rawKey.length === 0) return;
  try {
    const podBytes = appState.appGetPodProto(rawKey);
    if (podBytes.length)
      send("realtime:state-sync", JSON.stringify({ domain: "pod", podKey: rawKey, pod: Array.from(podBytes) }));
  } catch {
    /* best-effort */
  }
}

function pushRunnerSnapshot(appState: AppState, ev: Ev, send: Send): void {
  if (!ev.type || !RUNNER_EVENTS.has(ev.type)) return;
  try {
    send("realtime:state-sync", JSON.stringify({
      domain: "runner",
      runners: Array.from(appState.appRunnersProto()),
      available: Array.from(appState.appAvailableRunnersProto()),
      current: Array.from(appState.appCurrentRunnerProto()),
    }));
  } catch {
    /* best-effort */
  }
}

function pushAutopilotSnapshot(appState: AppState, ev: Ev, send: Send): void {
  if (!ev.type || !AUTOPILOT_EVENTS.has(ev.type)) return;
  const rawKey = ev.data?.autopilot_controller_key ?? ev.data?.autopilotControllerKey;
  const key = typeof rawKey === "string" ? rawKey : "";
  try {
    send("realtime:state-sync", JSON.stringify({
      domain: "autopilot",
      key,
      controllers: Array.from(appState.appAutopilotControllersProto()),
      iterations: key ? Array.from(appState.appAutopilotIterationsProto(key)) : [],
      thinking: key ? appState.appAutopilotThinkingJson(key) : "",
      thinkingHistory: key ? appState.appAutopilotThinkingHistoryJson(key) : "",
    }));
  } catch {
    /* best-effort */
  }
}

function pushMeshNodeSnapshot(appState: AppState, ev: Ev, send: Send): void {
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
