import { invoke } from "./invoke";

type OutputCb = (data: Uint8Array) => void;
type StatusCb = (info: { status: string; runnerDisconnected: boolean }) => void;
type AcpCb = (msgType: number, payload: unknown) => void;

interface RelayPushApi {
  onRelayOutput?: (h: (p: { podKey: string; data: Uint8Array }) => void) => () => void;
  onRelayStatus?: (h: (p: { podKey: string; json: string }) => void) => () => void;
  onRelayAcp?: (h: (p: { podKey: string; json: string }) => void) => () => void;
  onRelayPodDisconnected?: (h: (p: { podKey: string }) => void) => () => void;
}

// Renderer-side mirror of WASM's WasmRelayManager. The main-process Rust pool
// owns the single WS per pod; main fans PTY output / status / ACP to the
// renderer over `relay:*` push channels (clients/desktop/src/main/relay.ts).
//
// The web WasmRelayManager gets per-subscriber output fan-out for free from the
// pool's subscriber map. The IPC bridge collapses to ONE coalesced stream per
// pod (to avoid doubling when terminal + ACP share a pod), so this manager
// re-fans that single per-pod stream to each subId callback. Method names and
// callback argument shapes match WasmRelayManager exactly — the shared
// relayConnection adapter is written once against both.
export class ElectronRelayManager {
  private outputCbs = new Map<string, Map<string, OutputCb>>();
  private statusCbs = new Map<string, Set<StatusCb>>();
  private acpCbs = new Map<string, Set<AcpCb>>();
  private disconnectCbs = new Set<(podKey: string) => void>();

  constructor() {
    const api = (globalThis as { window?: { electronAPI?: RelayPushApi } }).window?.electronAPI;
    if (!api?.onRelayOutput || !api.onRelayStatus || !api.onRelayAcp) return;
    api.onRelayOutput(({ podKey, data }) => {
      const m = this.outputCbs.get(podKey);
      if (m) for (const cb of m.values()) cb(data);
    });
    api.onRelayStatus(({ podKey, json }) => {
      let info: { status: string; runnerDisconnected: boolean };
      try {
        info = JSON.parse(json) as { status: string; runnerDisconnected: boolean };
      } catch {
        // eslint-disable-next-line no-console
        console.warn(`relay: malformed status frame for pod ${podKey}`);
        return;
      }
      const s = this.statusCbs.get(podKey);
      if (s) for (const cb of s) cb(info);
    });
    api.onRelayAcp(({ podKey, json }) => {
      let parsed: { msgType: number; payload: unknown };
      try {
        parsed = JSON.parse(json) as { msgType: number; payload: unknown };
      } catch {
        // eslint-disable-next-line no-console
        console.warn(`relay: malformed acp frame for pod ${podKey}`);
        return;
      }
      const s = this.acpCbs.get(podKey);
      if (s) for (const cb of s) cb(parsed.msgType, parsed.payload);
    });
    api.onRelayPodDisconnected?.(({ podKey }) => {
      // Pool tore the pod down + cleared its Rust listeners. Drop our mirror's
      // per-pod sets so the next subscribe re-registers cleanly (no dupes), then
      // notify the relayConnection adapter to reset its upstream guard.
      this.statusCbs.delete(podKey);
      this.acpCbs.delete(podKey);
      for (const cb of this.disconnectCbs) cb(podKey);
    });
  }

  async subscribe(podKey: string, subId: string, url: string, token: string, cb: OutputCb) {
    let m = this.outputCbs.get(podKey);
    if (!m) { m = new Map(); this.outputCbs.set(podKey, m); }
    m.set(subId, cb);
    await invoke("relay:subscribe", podKey, subId, url, token);
  }

  async unsubscribe(podKey: string, subId: string) {
    const m = this.outputCbs.get(podKey);
    if (m) { m.delete(subId); if (m.size === 0) this.outputCbs.delete(podKey); }
    await invoke("relay:unsubscribe", podKey, subId);
  }

  async send(podKey: string, data: string) { await invoke("relay:send", podKey, data); }
  async send_resize(podKey: string, cols: number, rows: number) { await invoke("relay:resize", podKey, cols, rows); }
  async force_resize(podKey: string, cols: number, rows: number) { await invoke("relay:forceResize", podKey, cols, rows); }
  async send_acp_command(podKey: string, command: string) { await invoke("relay:acpCommand", podKey, command); }
  async disconnect(podKey: string) { await invoke("relay:disconnect", podKey); }
  async disconnect_all() { await invoke("relay:disconnectAll"); }
  async get_status(podKey: string): Promise<string> { return invoke<string>("relay:getStatus", podKey); }
  async is_runner_disconnected(podKey: string): Promise<boolean> { return invoke<boolean>("relay:isRunnerDisconnected", podKey); }

  async get_pod_size(podKey: string): Promise<{ cols: number; rows: number } | null> {
    const arr = await invoke<number[]>("relay:getPodSize", podKey);
    return Array.isArray(arr) && arr.length === 2 ? { cols: arr[0], rows: arr[1] } : null;
  }

  async on_status_change(podKey: string, cb: StatusCb) {
    let s = this.statusCbs.get(podKey);
    if (!s) { s = new Set(); this.statusCbs.set(podKey, s); }
    s.add(cb);
  }

  async on_acp_message(podKey: string, cb: AcpCb) {
    let s = this.acpCbs.get(podKey);
    if (!s) { s = new Set(); this.acpCbs.set(podKey, s); }
    s.add(cb);
  }

  async on_pod_disconnected(cb: (podKey: string) => void) {
    this.disconnectCbs.add(cb);
  }
}
