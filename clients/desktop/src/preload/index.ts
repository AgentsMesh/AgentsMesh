import { contextBridge, ipcRenderer, IpcRendererEvent } from "electron";
import { type ServerConfig } from "../shared/server-config-types";

// Sync IPC at preload time — main owns the SSOT, but renderer code
// (env.ts, OAuth URL builders, WS connect) is synchronous. Reading once
// at preload keeps renderer sync. The sync handler runs BEFORE any
// renderer code, so it doesn't block a UI thread; on `mainWindow.reload()`
// preload re-executes and re-reads, propagating `serverConfig:set`
// updates without a separate mutable channel.
//
// No defensive try/catch here: `sendSync` blocks indefinitely if the
// handler isn't registered (Electron API), so a `try/catch` over it can
// only catch a structural failure that doesn't actually happen in
// practice. Letting any genuine failure propagate makes it visible — a
// silent fallback to "" would render OAuth URLs as relative paths and
// fail in obscure ways downstream. Invariant: main MUST register the
// `serverConfig:*Sync` handlers before createWindow() (enforced by
// ordering in main/index.ts).
const apiUrl = ipcRenderer.sendSync("serverConfig:getActiveUrlSync") as string;
const serverConfigSnapshot = ipcRenderer.sendSync("serverConfig:getSync") as ServerConfig;

const api = {
  apiUrl,
  invoke: (channel: string, ...args: unknown[]) => ipcRenderer.invoke(channel, ...args),
  on: (channel: string, listener: (event: IpcRendererEvent, ...args: unknown[]) => void) => {
    ipcRenderer.on(channel, listener);
    return () => ipcRenderer.removeListener(channel, listener);
  },
  shellOpen: (url: string) => ipcRenderer.invoke("shellOpen", url),
  // Deep-link OAuth callback: main forwards `agentsmesh://oauth/callback?token=...`
  // here when the system browser hands the URL back via open-url (mac) or
  // second-instance argv (win/linux). Returns an unsubscribe so React Strict
  // Mode double-mount doesn't accumulate listeners.
  onOAuthCallback: (handler: (url: string) => void) => {
    const listener = (_e: IpcRendererEvent, url: string) => handler(url);
    ipcRenderer.on("oauth:callback", listener);
    return () => ipcRenderer.removeListener("oauth:callback", listener);
  },
  serverConfig: {
    snapshot: serverConfigSnapshot,
    get: () => ipcRenderer.invoke("serverConfig:get"),
    // After set, main reloads the renderer to re-snapshot. This Promise
    // resolves *before* the reload fires (handler returns first), but
    // any work done in the renderer between resolve and reload is about
    // to be torn down — callers shouldn't depend on it.
    set: (cfg: ServerConfig) => ipcRenderer.invoke("serverConfig:set", cfg),
  },
};

contextBridge.exposeInMainWorld("electronAPI", api);

declare global {
  interface Window {
    electronAPI: typeof api;
  }
}
