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
};

contextBridge.exposeInMainWorld("electronAPI", api);

declare global {
  interface Window {
    electronAPI: typeof api;
  }
}
