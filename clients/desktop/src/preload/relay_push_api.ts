import type {
  RelayPushApi,
  RelayPushChannel,
  RelayPushPayload,
} from "@agentsmesh/electron-adapter/relay-ipc-contract";
import { ipcRenderer, type IpcRendererEvent } from "electron";

function onRelay<C extends RelayPushChannel>(
  channel: C,
  handler: (payload: RelayPushPayload<C>) => void,
): () => void {
  const listener = (_event: IpcRendererEvent, payload: RelayPushPayload<C>) => handler(payload);
  ipcRenderer.on(channel, listener);
  return () => ipcRenderer.removeListener(channel, listener);
}

export const relayPushApi: RelayPushApi = {
  onRelayOutput: (handler) => onRelay("relay:output", handler),
  onRelayStatus: (handler) => onRelay("relay:status", handler),
  onRelayAcp: (handler) => onRelay("relay:acp", handler),
  onRelayPodDisconnected: (handler) => onRelay("relay:pod-disconnected", handler),
};
