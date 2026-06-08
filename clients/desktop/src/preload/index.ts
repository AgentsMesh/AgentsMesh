import { contextBridge, ipcRenderer, IpcRendererEvent } from "electron";
import { type ServerConfig } from "../shared/server-config-types";
import { type UpdaterSnapshot } from "../shared/updater-reducer";

// Sync IPC by design: renderer code (env.ts, OAuth URL builders, WS connect) is synchronous.
// Reading at preload (before any renderer code runs) blocks no UI thread; mainWindow.reload()
// re-runs preload to propagate serverConfig:set.
// No try/catch: sendSync blocks if handler isn't registered, so structural failure must be visible
// rather than silently falling back to "" (would render OAuth URLs as relative paths).
// Invariant: main MUST register serverConfig:*Sync handlers before createWindow() (main/index.ts).
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
  log: (level: string, target: string, message: string) =>
    ipcRenderer.invoke("core:log", level, target, message),
  openLogsFolder: () => ipcRenderer.invoke("logs:openFolder"),
  onOAuthCallback: (handler: (url: string) => void) => {
    const listener = (_e: IpcRendererEvent, url: string) => handler(url);
    ipcRenderer.on("oauth:callback", listener);
    return () => ipcRenderer.removeListener("oauth:callback", listener);
  },
  onRealtimeEvent: (handler: (eventJson: string) => void) => {
    const listener = (_e: IpcRendererEvent, eventJson: string) => handler(eventJson);
    ipcRenderer.on("realtime:event", listener);
    return () => ipcRenderer.removeListener("realtime:event", listener);
  },
  onUpdaterSnapshot: (handler: (snap: UpdaterSnapshot) => void) => {
    const listener = (_e: IpcRendererEvent, snap: UpdaterSnapshot) => handler(snap);
    ipcRenderer.on("updater:snapshot", listener);
    return () => ipcRenderer.removeListener("updater:snapshot", listener);
  },
  onRealtimeState: (handler: (state: string) => void) => {
    const listener = (_e: IpcRendererEvent, state: string) => handler(state);
    ipcRenderer.on("realtime:state", listener);
    return () => ipcRenderer.removeListener("realtime:state", listener);
  },
  // Rust-computed domain snapshot pushed after each EventBus dispatch. The
  // renderer mirrors it into the Electron service cache (the renderer has no
  // in-process Rust; main owns the SSOT runtime.state). See main/realtime.ts.
  onRealtimeStateSync: (handler: (snapshotJson: string) => void) => {
    const listener = (_e: IpcRendererEvent, snapshotJson: string) => handler(snapshotJson);
    ipcRenderer.on("realtime:state-sync", listener);
    return () => ipcRenderer.removeListener("realtime:state-sync", listener);
  },
  // Relay (terminal data plane) push channels: the main-process Rust pool fans
  // PTY output / status / ACP to the renderer. ElectronRelayManager subscribes.
  onRelayOutput: (handler: (payload: { podKey: string; data: Uint8Array }) => void) => {
    const listener = (_e: IpcRendererEvent, payload: { podKey: string; data: Uint8Array }) => handler(payload);
    ipcRenderer.on("relay:output", listener);
    return () => ipcRenderer.removeListener("relay:output", listener);
  },
  onRelayStatus: (handler: (payload: { podKey: string; json: string }) => void) => {
    const listener = (_e: IpcRendererEvent, payload: { podKey: string; json: string }) => handler(payload);
    ipcRenderer.on("relay:status", listener);
    return () => ipcRenderer.removeListener("relay:status", listener);
  },
  onRelayAcp: (handler: (payload: { podKey: string; json: string }) => void) => {
    const listener = (_e: IpcRendererEvent, payload: { podKey: string; json: string }) => handler(payload);
    ipcRenderer.on("relay:acp", listener);
    return () => ipcRenderer.removeListener("relay:acp", listener);
  },
  onRelayPodDisconnected: (handler: (payload: { podKey: string }) => void) => {
    const listener = (_e: IpcRendererEvent, payload: { podKey: string }) => handler(payload);
    ipcRenderer.on("relay:pod-disconnected", listener);
    return () => ipcRenderer.removeListener("relay:pod-disconnected", listener);
  },
  serverConfig: {
    snapshot: serverConfigSnapshot,
    get: () => ipcRenderer.invoke("serverConfig:get"),
    // Resolves BEFORE main's reload — callers cannot depend on work scheduled after this.
    set: (cfg: ServerConfig) => ipcRenderer.invoke("serverConfig:set", cfg),
  },
};

contextBridge.exposeInMainWorld("electronAPI", api);

declare global {
  interface Window {
    electronAPI: typeof api;
  }
}
