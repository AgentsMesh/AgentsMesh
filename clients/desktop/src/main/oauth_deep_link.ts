import { app, BrowserWindow } from "electron";
import path from "path";

const PROTOCOL = "agentsmesh";
const PROTOCOL_PREFIX = `${PROTOCOL}://`;
const CALLBACK_CHANNEL = "oauth:callback";

// Cold-launch URL captured before whenReady; macOS uses open-url, Windows uses argv.
let pendingUrl: string | null = null;

// Dev mode (electron-vite dev) requires passing process.execPath + dev entry argv,
// else clicking the link launches fresh Electron with no project loaded.
export function registerProtocol(): void {
  if (process.defaultApp && process.argv.length >= 2) {
    app.setAsDefaultProtocolClient(PROTOCOL, process.execPath, [
      path.resolve(process.argv[1]!),
    ]);
  } else {
    app.setAsDefaultProtocolClient(PROTOCOL);
  }
}

// Caller MUST acquire single-instance lock first (./single_instance.ts) — without it,
// OS spawns a fresh process and second-instance never fires.
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

export function installOpenUrlHandler(getWindow: () => BrowserWindow | null): void {
  app.on("open-url", (event, url) => {
    event.preventDefault();
    deliver(getWindow(), url);
  });
}

// MUST be called before whenReady. macOS cold-launch uses open-url; this is a no-op there.
export function captureColdLaunchUrl(): void {
  const url = process.argv.find((a) => a.startsWith(PROTOCOL_PREFIX));
  if (url) pendingUrl = url;
}

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
  // Cold-launch race: webContents may still be loading; wait for did-finish-load once.
  if (win.webContents.isLoading()) {
    win.webContents.once("did-finish-load", () => {
      win.webContents.send(CALLBACK_CHANNEL, url);
    });
  } else {
    win.webContents.send(CALLBACK_CHANNEL, url);
  }
}
