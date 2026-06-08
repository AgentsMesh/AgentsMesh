import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { INITIAL_SNAPSHOT, type UpdaterSnapshot, type UpdaterState } from "../../shared/updater-reducer";

type UpdaterApi = {
  onUpdaterSnapshot: (handler: (snap: UpdaterSnapshot) => void) => () => void;
  invoke: (channel: string, ...args: unknown[]) => Promise<unknown>;
};

// null in the web build / unit tests (no preload bridge) → a no-op idle updater.
function updaterApi(): UpdaterApi | null {
  if (typeof window === "undefined") return null;
  const api = (window as unknown as { electronAPI?: Partial<UpdaterApi> }).electronAPI;
  if (!api?.onUpdaterSnapshot || !api?.invoke) return null;
  return api as UpdaterApi;
}

export interface Updater {
  state: UpdaterState;
  percent: number;
  availableVersion: string | null;
  currentVersion: string | null;
  error: string | null;
  check: () => void;
  quitAndInstall: () => void;
}

const UpdaterContext = createContext<Updater | null>(null);

// One subscription shared by banner + settings; a per-component hook would fork
// state, and a section mounted after `downloaded` would disagree with the banner.
// main owns the reducer (shared/updater-reducer) and pushes snapshots; this just
// mirrors them.
export function UpdaterProvider({ children }: { children: ReactNode }) {
  const [snap, setSnap] = useState<UpdaterSnapshot>(INITIAL_SNAPSHOT);
  const [currentVersion, setCurrentVersion] = useState<string | null>(null);
  const liveSeen = useRef(false);

  useEffect(() => {
    const api = updaterApi();
    if (!api) return;

    const apply = (s: UpdaterSnapshot, fromSnapshot: boolean) => {
      // A getState snapshot resolving after the first live push must not clobber it.
      if (fromSnapshot && liveSeen.current) return;
      if (!fromSnapshot) liveSeen.current = true;
      setSnap(s);
    };

    void api.invoke("updater:getState").then((s) => apply(s as UpdaterSnapshot, true)).catch(() => {});
    void api
      .invoke("updater:getVersion")
      .then((v) => setCurrentVersion(typeof v === "string" ? v : null))
      .catch(() => {});
    return api.onUpdaterSnapshot((s) => apply(s, false));
  }, []);

  const value = useMemo<Updater>(
    () => ({
      state: snap.state,
      percent: snap.percent,
      availableVersion: snap.availableVersion,
      currentVersion,
      error: snap.error,
      check: () => void updaterApi()?.invoke("updater:check").catch(() => {}),
      quitAndInstall: () => void updaterApi()?.invoke("updater:quitAndInstall").catch(() => {}),
    }),
    [snap, currentVersion],
  );

  return <UpdaterContext.Provider value={value}>{children}</UpdaterContext.Provider>;
}

export function useUpdater(): Updater {
  const ctx = useContext(UpdaterContext);
  if (!ctx) throw new Error("useUpdater must be used within UpdaterProvider");
  return ctx;
}
