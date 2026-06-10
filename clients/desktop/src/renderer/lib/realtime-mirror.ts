import { getChannelState, getPodState, getRunnerState, getAutopilotState, getMeshState } from "@agentsmesh/service-runtime";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";
import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useAutopilotStore } from "@/stores/autopilot";
import { useMeshStore } from "@/stores/mesh";

interface ChannelSnapshot {
  domain: string;
  channelId: number;
  // Rust-computed `{ messages, has_more }` for the affected channel, or "".
  messages: string;
  // Full `{ channelId: count }` maps from runtime.state (baseline + realtime).
  unreadCounts: string;
  mentionCounts: string;
}

interface PodSnapshot {
  domain: string;
  podKey: string;
  // Rust-computed single pod JSON (already enriched), or "".
  pod: string;
}

interface RunnerSnapshot {
  domain: string;
  // Rust-computed runner lists from runtime.state.
  runners: string;
  available: string;
  current: string;
}

interface AutopilotSnapshot {
  domain: string;
  key: string;
  controllers: string;
  iterations: string;
  thinking: string;
  thinkingHistory: string;
}

interface MeshNodeSnapshot {
  domain: string;
  podKey: string;
  // Rust-computed single mesh node JSON (status/agent patched), or "".
  node: string;
}

interface ChannelMirror {
  set_messages?: (id: bigint, json: string, hasMore: boolean) => void;
  set_unread_counts?: (json: string) => void;
  set_mention_counts?: (json: string) => void;
}

type StateSyncBridge = { onRealtimeStateSync?: (h: (json: string) => void) => () => void };

function applyChannelSnapshot(snap: ChannelSnapshot): void {
  const svc = getChannelState() as unknown as ChannelMirror;
  if (snap.messages) {
    let hasMore = false;
    try {
      hasMore = (JSON.parse(snap.messages) as { has_more?: boolean }).has_more ?? false;
    } catch {
      /* keep false — malformed snapshot shouldn't wedge the mirror */
    }
    svc.set_messages?.(BigInt(snap.channelId), snap.messages, hasMore);
  }
  if (snap.unreadCounts) svc.set_unread_counts?.(snap.unreadCounts);
  if (snap.mentionCounts) svc.set_mention_counts?.(snap.mentionCounts);
  useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));
  useChannelMessageStore.setState((s) => ({
    _messagesTick: s._messagesTick + 1,
    _unreadTick: s._unreadTick + 1,
  }));
}

function applyPodSnapshot(snap: PodSnapshot): void {
  if (!snap.pod) return;
  const svc = getPodState() as unknown as { apply_pod_snapshot?: (json: string) => void };
  svc.apply_pod_snapshot?.(snap.pod);
  usePodStore.setState((s) => ({ _tick: s._tick + 1 }));
}

function applyRunnerSnapshot(snap: RunnerSnapshot): void {
  const svc = getRunnerState() as unknown as {
    apply_runners_snapshot?: (runners: string, available: string, current: string) => void;
  };
  svc.apply_runners_snapshot?.(snap.runners, snap.available, snap.current);
  useRunnerStore.setState((s) => ({ _tick: s._tick + 1 }));
}

function applyAutopilotSnapshot(snap: AutopilotSnapshot): void {
  const svc = getAutopilotState() as unknown as {
    apply_autopilot_snapshot?: (c: string, k: string, i: string, t: string, h: string) => void;
  };
  svc.apply_autopilot_snapshot?.(
    snap.controllers, snap.key, snap.iterations, snap.thinking, snap.thinkingHistory,
  );
  useAutopilotStore.setState((s) => ({ _tick: s._tick + 1 }));
}

// Patch the mesh node in the renderer's topology cache (no-op if not cached).
function applyMeshNodeSnapshot(snap: MeshNodeSnapshot): void {
  if (!snap.node) return;
  const svc = getMeshState() as unknown as { update_node?: (json: string) => void };
  svc.update_node?.(snap.node);
  useMeshStore.setState((s) => ({ _tick: s._tick + 1 }));
}

// Desktop renderer has no in-process Rust — main owns the SSOT runtime.state
// and pushes a Rust-computed snapshot after each EventBus dispatch. This mirror
// projects that snapshot into the renderer-local ElectronChannelService cache,
// then bumps the store ticks so selectors re-read. Web needs none of this: its
// wasm state IS the dispatched state, so a tick bump alone suffices there.
export function installRealtimeMirror(): void {
  const api = (window as unknown as { electronAPI?: StateSyncBridge }).electronAPI;
  if (!api?.onRealtimeStateSync) return;
  api.onRealtimeStateSync((json) => {
    let snap: { domain?: string };
    try {
      snap = JSON.parse(json) as { domain?: string };
    } catch {
      // eslint-disable-next-line no-console
      console.warn("realtime-mirror: malformed snapshot — dropped");
      return;
    }
    try {
      if (snap.domain === "channel") applyChannelSnapshot(snap as ChannelSnapshot);
      else if (snap.domain === "pod") applyPodSnapshot(snap as PodSnapshot);
      else if (snap.domain === "runner") applyRunnerSnapshot(snap as RunnerSnapshot);
      else if (snap.domain === "autopilot") applyAutopilotSnapshot(snap as AutopilotSnapshot);
      else if (snap.domain === "mesh") applyMeshNodeSnapshot(snap as MeshNodeSnapshot);
    } catch {
      // Apply failure = renderer view silently stops tracking the SSOT. Log the
      // domain (never the payload) so the stuck view is diagnosable.
      // eslint-disable-next-line no-console
      console.warn(`realtime-mirror: apply failed for domain=${snap.domain ?? "?"}`);
    }
  });
}
