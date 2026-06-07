import { invoke } from "./invoke";

// Renderer-side bridge to the main-process EventBus stream.
//
// Surface area matches the wasm-emitted `WasmEventsManager` so the
// renderer's `EventSubscriptionManager` (clients/web/src/lib/realtime/
// EventSubscriptionManager.ts) is wire-compatible with both runtimes:
//   - subscribe_all(cb) → Promise<number>
//   - on_connection_state_change(cb) → Promise<number>
//   - unsubscribe(id) → Promise<void>
//   - connect()/disconnect() → Promise<void>
//
// Main process owns the actual Connect-RPC stream (see
// clients/desktop/src/main/realtime.ts); this class only fan-outs the
// IPC events it pushes via `webContents.send("realtime:event", json)`.

type EventCallback = (eventJson: string) => void;
type StateCallback = (state: string) => void;

interface RealtimeBridgeApi {
  invoke: (channel: string, ...args: unknown[]) => Promise<unknown>;
  onRealtimeEvent?: (handler: EventCallback) => () => void;
  onRealtimeState?: (handler: StateCallback) => () => void;
}

export class ElectronEventsManager {
  private eventCallbacks = new Map<number, EventCallback>();
  private stateCallbacks = new Map<number, StateCallback>();
  private nextId = 1;
  private unsubFns: Array<() => void> = [];
  private currentState: string = "disconnected";

  constructor() {
    const api = (globalThis as { window?: { electronAPI?: RealtimeBridgeApi } }).window
      ?.electronAPI;
    if (!api?.onRealtimeEvent || !api?.onRealtimeState) {
      // Preload didn't expose realtime IPC channels — stale preload bundle or
      // non-Electron host. Subscribers still register but nothing dispatches;
      // warn so a dead realtime stream is diagnosable from the log file.
      // eslint-disable-next-line no-console
      console.warn("realtime: IPC channels unavailable — preload stale? events will not dispatch");
      return;
    }
    this.unsubFns.push(
      api.onRealtimeEvent((eventJson: string) => {
        for (const cb of this.eventCallbacks.values()) cb(eventJson);
      }),
    );
    this.unsubFns.push(
      api.onRealtimeState((state: string) => {
        this.currentState = state;
        for (const cb of this.stateCallbacks.values()) cb(state);
      }),
    );
  }

  async subscribe_all(cb: EventCallback): Promise<number> {
    const id = this.nextId++;
    this.eventCallbacks.set(id, cb);
    return id;
  }

  async on_connection_state_change(cb: StateCallback): Promise<number> {
    const id = this.nextId++;
    this.stateCallbacks.set(id, cb);
    // Match wasm behavior: synchronously call back with the current state so
    // the renderer's RealtimeProvider can render the initial badge correctly.
    queueMicrotask(() => cb(this.currentState));
    return id;
  }

  async unsubscribe(id: number): Promise<void> {
    this.eventCallbacks.delete(id);
    this.stateCallbacks.delete(id);
  }

  async connect(): Promise<void> {
    await invoke("realtime:connect");
  }

  async disconnect(): Promise<void> {
    await invoke("realtime:disconnect");
  }

  async nudge(): Promise<void> {
    await invoke("realtime:nudge");
  }
}
