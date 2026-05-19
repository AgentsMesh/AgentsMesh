import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { MutableRefObject } from "react";
import { relayPool, useWorkspaceStore } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/relayConnection";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";
import { isResourceNotFound } from "@/lib/errors/serviceError";

export interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
}

// Self-healing: 404 → drop pane to stop stale-snapshot reconnect loops.
export function setupConnection(
  podKey: string,
  scheduler: TerminalWriteScheduler,
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
    } catch (error) {
      if (abort.signal.aborted) return;
      if (isResourceNotFound(error)) {
        useWorkspaceStore.getState().removePaneByPodKey(podKey);
        return;
      }
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
