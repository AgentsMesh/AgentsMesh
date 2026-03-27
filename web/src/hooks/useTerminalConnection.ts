import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { MutableRefObject } from "react";
import { relayPool } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/relayConnection";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";

export interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
  disconnect: () => void;
}

/**
 * Subscribes to the terminal pool WebSocket, wiring incoming data
 * through the scheduler. Returns an AbortController for cleanup.
 */
export function setupConnection(
  podKey: string,
  scheduler: TerminalWriteScheduler,
  initialDims: { value: { cols: number; rows: number } | null },
  connectionRef: MutableRefObject<TerminalConnection | null>,
  setConnectionStatus: (status: ConnectionStatus) => void,
  setIsRunnerDisconnected: (v: boolean) => void,
): { abort: AbortController; unsubscribeStatus: () => void } {
  const handleMessage = (data: Uint8Array | string) => {
    if (data instanceof Uint8Array) {
      scheduler.schedule(data);
    } else {
      scheduler.schedule(new TextEncoder().encode(data));
    }
  };

  const subscriptionId = `terminal-${podKey}`;
  const abort = new AbortController();

  (async () => {
    try {
      const handle = await relayPool.subscribe(podKey, subscriptionId, handleMessage);
      if (abort.signal.aborted) return;
      connectionRef.current = handle;
      if (initialDims.value) {
        relayPool.forceResize(podKey, initialDims.value.cols, initialDims.value.rows);
      }
    } catch (error) {
      if (abort.signal.aborted) return;
      console.error("Failed to connect terminal:", error);
      setConnectionStatus("error");
    }
  })();

  const unsubscribeStatus = relayPool.onStatusChange(podKey, (info) => {
    if (info.status !== "none") {
      setConnectionStatus(info.status);
    }
    setIsRunnerDisconnected(info.runnerDisconnected);
  });

  return { abort, unsubscribeStatus };
}

/**
 * Wires onData (user input) and onResize handlers on the xterm instance.
 */
export function setupDataHandlers(
  term: XTerm,
  podKey: string,
  connectionRef: MutableRefObject<TerminalConnection | null>,
  isComposing: { current: boolean },
  disposables: IDisposable[],
): void {
  const dataDisposable = term.onData((data) => {
    if (isComposing.current) return;
    connectionRef.current?.send(data);
  });

  const resizeDisposable = term.onResize(({ rows, cols }) => {
    relayPool.sendResize(podKey, cols, rows);
  });

  disposables.push(dataDisposable, resizeDisposable);
}
