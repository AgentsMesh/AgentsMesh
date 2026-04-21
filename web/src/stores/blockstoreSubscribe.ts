import { reconnectRegistry } from "@/lib/realtime";
import type { RealtimeEvent } from "@/lib/realtime";
import type { BlockOp } from "@/lib/api/blockstoreTypes";
import { getBlockstoreService } from "@/lib/wasm-core";
import { useBlockstoreStore, readLastOpIds } from "./blockstore";

// handleBlockstoreEvent is invoked by RealtimeProvider for every
// `blockstore:*` event received over the org-wide WebSocket.
//
// Scope guard: the backend fans out ops across the whole organization, so
// users connected to the same org see each other's workspace ops on the wire.
// We drop ops for workspaces the user has not loaded — otherwise block data
// from a workspace the user has no UI access to would still land in the cache.
export function handleBlockstoreEvent(event: RealtimeEvent) {
  if (event.type !== "blockstore:op") return;
  const op = event.data as BlockOp;
  if (!(op.workspace_id in readLastOpIds())) return;
  try {
    getBlockstoreService().apply_remote_op(JSON.stringify(op));
  } catch {
    return;
  }
  useBlockstoreStore.setState((s) => ({ _tick: s._tick + 1 }));
}

// Register on module load so every consumer of the store gets catch-up for free
// after a WebSocket reconnect. Catch-up is per-workspace; we iterate over
// whatever lastOpId entries currently exist in Rust state.
reconnectRegistry.register({
  name: "blockstore:catchup",
  fn: () => {
    const lastOpId = readLastOpIds();
    const actions = useBlockstoreStore.getState().actions;
    Object.keys(lastOpId).forEach((wsID) => {
      void actions.catchup(wsID);
    });
  },
  priority: "deferred",
});
