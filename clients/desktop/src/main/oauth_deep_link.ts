import { app, BrowserWindow } from "electron";
import path from "path";

const PROTOCOL = "agentsmesh";
const PROTOCOL_PREFIX = `${PROTOCOL}://`;
const CALLBACK_CHANNEL = "oauth:callback";

// Cold-launch URL is captured before whenReady fires; we keep it
// in this module-level slot until the BrowserWindow is wired up,
// then flush. mac and Windows have different cold-launch entry
// points (open-url vs argv) but both feed into this slot.
let pendingUrl: string | null = null;

/**
 * Registers `agentsmesh://` as a custom protocol handler with the OS.
 *
 * In packaged builds, electron-builder also writes the registration
 * into Info.plist (mac) and the NSIS installer (win). Calling this
 * at runtime is still required for dev mode and is a no-op when
 * the OS already has the registration.
 *
 * Dev mode (`electron-vite dev`): we have to pass `process.execPath`
 * + the dev entry argv so the OS knows what binary to launch. Without
 * it, clicking the link would launch a fresh Electron with no project
 * loaded.
 */
export function registerProtocol(): void {
  if (process.defaultApp && process.argv.length >= 2) {
    app.setAsDefaultProtocolClient(PROTOCOL, process.execPath, [
      path.resolve(process.argv[1]!),
    ]);
  } else {
    app.setAsDefaultProtocolClient(PROTOCOL);
  }
}

/**
 * Caller MUST acquire the single-instance lock first (see ./single_instance.ts) —
 * without it the OS spawns a fresh process instead of waking us, and
 * `second-instance` never fires.
 */
export function attachSecondInstanceUrlHandler(getWindow: () => BrowserWindow | null): void {
  app.on("second-instance", (_event, argv) => {
    const url = argv.find((a) => a.startsWith(PROTOCOL_PREFIX));
    if (url) deliver(getWindow(), url);
    const win = getWindow();
    if (win) {
      if (win.isMinimized()) win.restore();
      win.focus();
    }
  });
}

/** macOS-only: app already running → "open-url" event. */
export function installOpenUrlHandler(getWindow: () => BrowserWindow | null): void {
  app.on("open-url", (event, url) => {
    event.preventDefault();
    deliver(getWindow(), url);
  });
}

/**
 * Capture cold-launch URL from argv (Windows / Linux) so we can
 * deliver it once the window is created. Call before whenReady;
 * on macOS cold-launch the URL arrives via "open-url" instead and
 * this is a no-op.
 */
export function captureColdLaunchUrl(): void {
  const url = process.argv.find((a) => a.startsWith(PROTOCOL_PREFIX));
  if (url) pendingUrl = url;
}

/**
 * After the BrowserWindow exists and webContents is ready, flush
 * any pending cold-launch URL into the renderer.
 */
export function flushPendingUrl(getWindow: () => BrowserWindow | null): void {
  if (!pendingUrl) return;
  deliver(getWindow(), pendingUrl);
  pendingUrl = null;
}

function deliver(win: BrowserWindow | null, url: string): void {
  if (!win) {
    pendingUrl = url;
    return;
  }
  // If the page hasn't finished loading yet (cold-launch race), wait
  // once for did-finish-load; otherwise send immediately.
  if (win.webContents.isLoading()) {
    win.webContents.once("did-finish-load", () => {
      win.webContents.send(CALLBACK_CHANNEL, url);
    });
  } else {
    win.webContents.send(CALLBACK_CHANNEL, url);
  }
}
