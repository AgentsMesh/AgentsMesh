import { invoke } from "./invoke";
import {
  ElectronRelayOutputRoutes,
  type RelayOutputCallback,
} from "./electron-relay-output-routes";
import type {
  RelayInvokeArgs,
  RelayInvokeChannel,
  RelayInvokeResult,
  RelayPodDisconnectedPayload,
  RelayPushApi,
} from "./relay-ipc-contract";

type StatusCb = (info: { status: string; runnerDisconnected: boolean }) => void;
type AcpCb = (msgType: number, payload: unknown) => void;

function invokeRelay<C extends RelayInvokeChannel>(
  channel: C,
  ...args: RelayInvokeArgs<C>
): Promise<RelayInvokeResult<C>> {
  return invoke<RelayInvokeResult<C>>(channel, ...args);
}

// Renderer-side mirror of WASM's WasmRelayManager. The main-process Rust pool
// owns the single WS per pod; main fans PTY output / status / ACP to the
// renderer over `relay:*` push channels (clients/desktop/src/main/relay.ts).
//
// The main bridge gives every renderer subscription a matching Rust pool
// subscriber. Output carries its attempt id over IPC, so a replacement baseline
// is delivered only to its callback; status and ACP remain pod-scoped fan-out.
// Method names and callback argument shapes match WasmRelayManager exactly.
export class ElectronRelayManager {
  private readonly outputRoutes = new ElectronRelayOutputRoutes();
  private readonly pendingDisconnects = new Map<string, RelayPodDisconnectedPayload>();
  private statusCbs = new Map<string, Set<StatusCb>>();
  private acpCbs = new Map<string, Set<AcpCb>>();
  private disconnectCbs = new Set<(podKey: string) => void>();

  constructor() {
    const api = (globalThis as { window?: { electronAPI?: Partial<RelayPushApi> } })
      .window?.electronAPI;
    if (!api?.onRelayOutput || !api.onRelayStatus || !api.onRelayAcp) return;
    api.onRelayOutput((payload) => this.outputRoutes.receive(payload));
    api.onRelayStatus(({ podKey, json }) => {
      let info: { status: string; runnerDisconnected: boolean };
      try {
        info = JSON.parse(json) as { status: string; runnerDisconnected: boolean };
      } catch {
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
        console.warn(`relay: malformed acp frame for pod ${podKey}`);
        return;
      }
      const s = this.acpCbs.get(podKey);
      if (s) for (const cb of s) cb(parsed.msgType, parsed.payload);
    });
    api.onRelayPodDisconnected?.((event) => this.handlePodDisconnected(event));
  }

  async subscribe(
    podKey: string,
    subId: string,
    url: string,
    token: string,
    cb: RelayOutputCallback,
  ): Promise<void> {
    const attempt = this.outputRoutes.begin(podKey, subId, cb);
    try {
      await this.outputRoutes.complete(
        attempt,
        invokeRelay("relay:subscribe", podKey, subId, attempt.attemptId, url, token),
        () => this.reconcilePendingDisconnect(podKey),
      );
    } finally {
      this.reconcilePendingDisconnect(podKey);
    }
  }

  async unsubscribe(podKey: string, subId: string): Promise<void> {
    const restore = this.outputRoutes.remove(podKey, subId);
    this.reconcilePendingDisconnect(podKey);
    try {
      await invokeRelay("relay:unsubscribe", podKey, subId);
    } catch (error) {
      restore?.();
      throw error;
    }
  }

  async send(podKey: string, data: string) { await invokeRelay("relay:send", podKey, data); }
  async send_resize(podKey: string, cols: number, rows: number) { await invokeRelay("relay:resize", podKey, cols, rows); }
  async force_resize(podKey: string, cols: number, rows: number) { await invokeRelay("relay:forceResize", podKey, cols, rows); }
  async send_acp_command(podKey: string, command: string) { await invokeRelay("relay:acpCommand", podKey, command); }
  async disconnect(podKey: string) {
    this.outputRoutes.dropPod(podKey);
    await invokeRelay("relay:disconnect", podKey);
  }
  async disconnect_all() {
    this.outputRoutes.dropAll();
    await invokeRelay("relay:disconnectAll");
  }
  async get_status(podKey: string): Promise<string> { return invokeRelay("relay:getStatus", podKey); }
  async is_runner_disconnected(podKey: string): Promise<boolean> { return invokeRelay("relay:isRunnerDisconnected", podKey); }

  async get_pod_size(podKey: string): Promise<{ cols: number; rows: number } | null> {
    const arr = await invokeRelay("relay:getPodSize", podKey);
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

  private handlePodDisconnected(event: RelayPodDisconnectedPayload): void {
    const { podKey } = event;
    if (this.outputRoutes.hasActivePod(podKey)) {
      this.pendingDisconnects.delete(podKey);
      return;
    }
    if (this.outputRoutes.hasPod(podKey)) {
      const pending = this.pendingDisconnects.get(podKey);
      if (!pending || event.generation > pending.generation) {
        this.pendingDisconnects.set(podKey, event);
      }
      return;
    }
    this.applyPodDisconnected(podKey);
  }

  private reconcilePendingDisconnect(podKey: string): void {
    if (!this.pendingDisconnects.has(podKey)) return;
    if (this.outputRoutes.hasActivePod(podKey)) {
      this.pendingDisconnects.delete(podKey);
    } else if (!this.outputRoutes.hasPod(podKey)) {
      this.applyPodDisconnected(podKey);
    }
  }

  private applyPodDisconnected(podKey: string): void {
    this.pendingDisconnects.delete(podKey);
    this.statusCbs.delete(podKey);
    this.acpCbs.delete(podKey);
    this.outputRoutes.dropPod(podKey);
    for (const cb of this.disconnectCbs) cb(podKey);
  }
}
