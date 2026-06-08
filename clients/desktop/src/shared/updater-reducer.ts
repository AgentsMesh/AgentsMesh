export type UpdaterState = "idle" | "checking" | "downloading" | "ready" | "error";

// Wire-in events from electron-updater — main folds these into the Snapshot that
// preload + renderer mirror; the events themselves never cross the IPC boundary.
export type UpdaterEvent =
  | { type: "checking" }
  | { type: "available"; version: string }
  | { type: "not-available" }
  | { type: "progress"; percent: number }
  | { type: "downloaded"; version: string }
  | { type: "error"; message: string };

export type UpdaterSnapshot = {
  state: UpdaterState;
  percent: number;
  availableVersion: string | null;
  error: string | null;
};

export const INITIAL_SNAPSHOT: UpdaterSnapshot = {
  state: "idle",
  percent: 0,
  availableVersion: null,
  error: null,
};

// ready is terminal: electron-updater re-emits `available` for the already-staged
// build on every re-check, so only a fresh `downloaded` leaves ready (else the
// restart banner flickers). main folds events through this and pushes the
// snapshot; the renderer mirrors it — one reducer, no two-layer drift. Returning
// `prev` unchanged lets main skip the push (see send()).
export function reduceUpdater(prev: UpdaterSnapshot, ev: UpdaterEvent): UpdaterSnapshot {
  if (prev.state === "ready" && ev.type !== "downloaded") return prev;
  switch (ev.type) {
    case "checking":
      return { ...prev, state: "checking", percent: 0, error: null };
    case "available":
      return { ...prev, state: "downloading", availableVersion: ev.version, percent: 0 };
    case "progress":
      return { ...prev, state: "downloading", percent: ev.percent };
    case "downloaded":
      return { ...prev, state: "ready", availableVersion: ev.version };
    case "not-available":
      return prev.state === "idle" ? prev : { ...prev, state: "idle" };
    case "error":
      return { ...prev, state: "error", error: ev.message };
    default: {
      const _exhaustive: never = ev;
      return _exhaustive;
    }
  }
}
