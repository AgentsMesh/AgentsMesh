import { contextBridge, ipcRenderer, IpcRendererEvent } from "electron";
import { type ServerConfig } from "../shared/server-config-types";

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
