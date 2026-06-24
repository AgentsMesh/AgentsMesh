// Multi-window seam. On desktop (Electron) window operations go through IPC to
// the main process — the renderer's native window.open is denied by the main
// setWindowOpenHandler. On web they fall back to a browser popout. Runtime-
// detected (mirrors lib/shims electron-shell), so no build-time platform branch.
interface ElectronApi {
  invoke?: (channel: string, ...args: unknown[]) => Promise<unknown>;
  // Projected from the main-process capability table (preload). The raw kind string
  // is deliberately not exposed, so policy can't drift back to `kind === "..."` tests.
  ephemeralLayout?: boolean;
  ownsNotificationsSeed?: boolean;
  onPrimaryChanged?: (cb: () => void) => () => void;
}

export function electronApi(): ElectronApi | undefined {
  return (globalThis as { window?: { electronAPI?: ElectronApi } }).window?.electronAPI;
}

function openBrowserPopout(path: string, name: string): void {
  window.open(path, name, "width=1000,height=720,menubar=no,toolbar=no,location=no,status=no");
}

function warnOpenFailed(target: string, e: unknown): void {
  console.warn(`[windowing] open ${target} failed:`, e);
}

export function openTerminalWindow(podKey: string): void {
  const api = electronApi();
  if (api?.invoke) {
    api.invoke("window:openTerminal", podKey).catch((e) => warnOpenFailed("terminal", e));
    return;
  }
  openBrowserPopout(`/popout/terminal/${podKey}`, `terminal-${podKey}`);
}

export function openChannelWindow(channelId: number): void {
  if (!Number.isFinite(channelId) || channelId <= 0) return;
  const api = electronApi();
  if (api?.invoke) {
    api.invoke("window:open", "channel", { id: String(channelId) }).catch((e) =>
      warnOpenFailed("channel", e),
    );
    return;
  }
  openBrowserPopout(`/popout/channel/${channelId}`, `channel-${channelId}`);
}

// Notification ownership — only ONE window fires the JS-layer browser
// notifications in DashboardShell, else multi-window duplicates every OS toast.
// Resolved against the main process's LIVE primary designation (not the static
// windowKind) so it follows primary re-election when the primary closes. Web
// (no electronAPI) is the sole tab → always the owner.
export function queryIsPrimaryWindow(): Promise<boolean> {
  const api = electronApi();
  if (!api?.invoke) return Promise.resolve(true);
  return api.invoke("window:isPrimary").then((p) => Boolean(p)).catch((e: unknown) => {
    console.warn("[windowing] queryIsPrimaryWindow failed:", e);
    return false;
  });
}

// Subscribe to main-process primary re-election; returns an unsubscribe fn.
// No-op on web.
export function onPrimaryWindowChanged(cb: () => void): () => void {
  return electronApi()?.onPrimaryChanged?.(cb) ?? (() => {});
}

// True when a screen-space pointer sits outside the current window's content
// rect. Tab tear-off uses it to decide whether a dragged pane should spawn its
// own window. screenX/screenY (window content origin) is paired with
// inner{Width,Height} (content size) so both sides share the content frame —
// outer dims would overshoot the right/bottom edge by the window chrome.
// Compared in CSS px; good enough for "dragged clearly out of the window".
export function isOutsideCurrentWindow(screenX: number, screenY: number): boolean {
  const { screenX: wx, screenY: wy, innerWidth: ww, innerHeight: wh } = window;
  return screenX < wx || screenX > wx + ww || screenY < wy || screenY > wy + wh;
}
