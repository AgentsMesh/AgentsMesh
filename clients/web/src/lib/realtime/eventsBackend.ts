/**
 * Events WebSocket transport abstraction.
 *
 * Separates transport (connect/send/close) from subscription logic.
 * Backends:
 * - TsEventsTransport:   browser `new WebSocket` (fallback / tests)
 * - WasmEventsTransport: WASM WebSocket via agentsmesh-wasm WasmWebSocket
 */

import type { WasmWebSocket as WasmWebSocketType } from "@/lib/wasm-core";

export interface IEventsTransport {
  readonly isOpen: boolean;
  send(data: string): void;
  close(code?: number, reason?: string): void;
}

export interface EventsTransportCallbacks {
  onOpen: () => void;
  onMessage: (data: string) => void;
  onClose: (code: number) => void;
  onError: () => void;
}

export interface IEventsBackend {
  connect(url: string, callbacks: EventsTransportCallbacks): IEventsTransport;
}

class TsEventsTransport implements IEventsTransport {
  private ws: WebSocket;

  constructor(url: string, callbacks: EventsTransportCallbacks) {
    this.ws = new WebSocket(url);
    this.ws.onopen = () => callbacks.onOpen();
    this.ws.onmessage = (e) => callbacks.onMessage(e.data as string);
    this.ws.onclose = (e) => callbacks.onClose(e.code);
    this.ws.onerror = () => callbacks.onError();
  }

  get isOpen(): boolean { return this.ws.readyState === WebSocket.OPEN; }

  send(data: string): void { this.ws.send(data); }

  close(code?: number, reason?: string): void {
    // Detach business callbacks first — whoever called close() is tearing
    // down the session and doesn't want late onmessage/onerror/onclose to
    // bubble into the (already-replaced) EventSubscriptionManager.
    this.ws.onmessage = null;
    this.ws.onerror = null;
    this.ws.onclose = null;

    if (this.ws.readyState === WebSocket.CONNECTING) {
      // Calling ws.close() during the handshake makes Chrome log
      // "WebSocket is closed before the connection is established" — a
      // spec-mandated warning whenever a CONNECTING socket is aborted.
      // Defer the close until onopen so readyState is OPEN at close time.
      // This fires under React Strict Mode's double-mount: the first
      // mount's ws is still negotiating when the second mount runs.
      const ws = this.ws;
      ws.onopen = () => {
        ws.onopen = null;
        ws.close(code, reason);
      };
      return;
    }

    this.ws.onopen = null;
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.close(code, reason);
    }
  }
}

class TsEventsBackend implements IEventsBackend {
  connect(url: string, callbacks: EventsTransportCallbacks): IEventsTransport {
    return new TsEventsTransport(url, callbacks);
  }
}

class WasmEventsTransport implements IEventsTransport {
  private ws: WasmWebSocketType;

  constructor(ws: WasmWebSocketType) {
    this.ws = ws;
  }

  get isOpen(): boolean { return this.ws.is_open(); }

  send(data: string): void { this.ws.send_text(data); }

  close(): void { this.ws.close(); }
}

class WasmEventsBackend implements IEventsBackend {
  private WasmWebSocket: typeof WasmWebSocketType;

  constructor(WasmWs: typeof WasmWebSocketType) {
    this.WasmWebSocket = WasmWs;
  }

  connect(url: string, callbacks: EventsTransportCallbacks): IEventsTransport {
    const ws = this.WasmWebSocket.connect(
      url,
      () => callbacks.onOpen(),
      (data: ArrayBuffer | string) => {
        if (typeof data === "string") {
          callbacks.onMessage(data);
        }
      },
      (info: { code: number }) => callbacks.onClose(info.code),
      () => callbacks.onError(),
    );
    return new WasmEventsTransport(ws);
  }
}

let activeBackend: IEventsBackend = new TsEventsBackend();

export function getEventsBackend(): IEventsBackend { return activeBackend; }
export function setEventsBackend(backend: IEventsBackend): void { activeBackend = backend; }

export function activateWasmEventsBackend(WasmWs: typeof WasmWebSocketType): void {
  activeBackend = new WasmEventsBackend(WasmWs);
}
