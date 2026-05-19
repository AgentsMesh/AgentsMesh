import type { WasmWebSocket as WasmWebSocketType } from "@/lib/wasm-core";

export type RelayMessageHandler = (data: ArrayBuffer) => void;
export type RelayCloseHandler = (code: number, reason: string) => void;
export type RelayOpenHandler = () => void;
export type RelayErrorHandler = (error: unknown) => void;

export interface IRelayTransport {
  readonly isOpen: boolean;
  readonly isClosed: boolean;
  send(data: Uint8Array): void;
  close(): void;
}

export interface RelayTransportCallbacks {
  onOpen: RelayOpenHandler;
  onMessage: RelayMessageHandler;
  onClose: RelayCloseHandler;
  onError: RelayErrorHandler;
}

export interface IRelayBackend {
  connect(url: string, callbacks: RelayTransportCallbacks): IRelayTransport;
}

class TsRelayTransport implements IRelayTransport {
  private ws: WebSocket;

  constructor(url: string, callbacks: RelayTransportCallbacks) {
    this.ws = new WebSocket(url);
    this.ws.binaryType = "arraybuffer";
    this.ws.onopen = () => callbacks.onOpen();
    this.ws.onmessage = (e) => callbacks.onMessage(e.data as ArrayBuffer);
    this.ws.onclose = (e) => callbacks.onClose(e.code, e.reason);
    this.ws.onerror = (e) => callbacks.onError(e);
  }

  get isOpen(): boolean { return this.ws.readyState === WebSocket.OPEN; }
  get isClosed(): boolean {
    return this.ws.readyState === WebSocket.CLOSED || this.ws.readyState === WebSocket.CLOSING;
  }

  send(data: Uint8Array): void { this.ws.send(data); }

  close(): void {
    this.ws.onopen = null;
    this.ws.onmessage = null;
    this.ws.onerror = null;
    this.ws.onclose = null;
    if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
      this.ws.close();
    }
  }
}

class TsRelayBackend implements IRelayBackend {
  connect(url: string, callbacks: RelayTransportCallbacks): IRelayTransport {
    return new TsRelayTransport(url, callbacks);
  }
}

class WasmRelayTransport implements IRelayTransport {
  private ws: WasmWebSocketType;

  constructor(ws: WasmWebSocketType) {
    this.ws = ws;
  }

  get isOpen(): boolean { return this.ws.is_open(); }
  get isClosed(): boolean { return this.ws.is_closed(); }

  send(data: Uint8Array): void { this.ws.send_binary(data); }

  close(): void { this.ws.close(); }
}

class WasmRelayBackend implements IRelayBackend {
  private WasmWebSocket: typeof WasmWebSocketType;

  constructor(WasmWs: typeof WasmWebSocketType) {
    this.WasmWebSocket = WasmWs;
  }

  connect(url: string, callbacks: RelayTransportCallbacks): IRelayTransport {
    const ws = this.WasmWebSocket.connect(
      url,
      () => callbacks.onOpen(),
      (data: ArrayBuffer | string) => {
        if (data instanceof ArrayBuffer) {
          callbacks.onMessage(data);
        }
      },
      (info: { code: number; reason: string }) => callbacks.onClose(info.code, info.reason ?? ""),
      () => callbacks.onError(new Error("WebSocket error")),
    );
    return new WasmRelayTransport(ws);
  }
}

let activeBackend: IRelayBackend = new TsRelayBackend();

export function getRelayBackend(): IRelayBackend { return activeBackend; }
export function setRelayBackend(backend: IRelayBackend): void { activeBackend = backend; }

export function activateWasmRelayBackend(WasmWs: typeof WasmWebSocketType): void {
  activeBackend = new WasmRelayBackend(WasmWs);
}
