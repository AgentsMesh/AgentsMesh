import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { MutableRefObject } from "react";
import { relayPool, useWorkspaceStore } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/relayConnection";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";
import { isResourceNotFound, isPodNotConnectable } from "@/lib/errors/serviceError";

export interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
}

let terminalSubscriptionAttempt = 0;

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

  // The subscription ID is also the unsubscribe ownership token in the relay
  // pool. A stale aborted attempt may resolve after its replacement; reusing a
  // pod-scoped ID would let the stale handle remove the replacement subscription.
  const subscriptionId = `terminal-${podKey}-${++terminalSubscriptionAttempt}`;
  const abort = new AbortController();

  (async () => {
    try {
      const handle = await relayPool.subscribe(
        podKey,
        subscriptionId,
        handleMessage,
        abort.signal,
      );
      if (abort.signal.aborted) {
        handle.unsubscribe();
        return;
      }
      connectionRef.current = handle;
    } catch (error) {
      if (abort.signal.aborted) return;
      if (isResourceNotFound(error)) {
        useWorkspaceStore.getState().removePaneByPodKey(podKey);
        return;
      }
      // Pod is spinning up or just completed — a benign lifecycle transient,
      // not a connection failure. The relay status listener + the pod-status
      // effect dep re-drive subscribe once it's connectable again, so don't
      // flip to "error" (which conflates "starting" with "broken").
      if (isPodNotConnectable(error)) return;
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
    if (term.hasSelection()) {
      term.clearSelection();
    }
    relayPool.sendResize(podKey, cols, rows);
  });

  disposables.push(dataDisposable, resizeDisposable);
}
