// Per-window layout storage: non-primary windows use sessionStorage (ephemeralLayout
// trait) to avoid colliding on the shared file:// localStorage origin; web + primary
// keep localStorage.
export function resolvePersistStorage(): Storage {
  const win = (globalThis as { window?: Window & { electronAPI?: { ephemeralLayout?: boolean } } }).window;
  if (!win) throw new Error("Storage not available (SSR)"); // createJSONStorage catches → persist skips
  return win.electronAPI?.ephemeralLayout ? win.sessionStorage : win.localStorage;
}
