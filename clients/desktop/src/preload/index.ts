import { contextBridge, ipcRenderer, IpcRendererEvent } from "electron";

const apiUrl = process.env.AGENTSMESH_API_URL ?? "http://localhost:25350";

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
};

contextBridge.exposeInMainWorld("electronAPI", api);

declare global {
  interface Window {
    electronAPI: typeof api;
  }
}
