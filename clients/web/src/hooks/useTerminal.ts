"use client";

import { useEffect, useRef, useState, MutableRefObject } from "react";
import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";
import { terminalRegistry } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/relayConnection";
import {
  setupTerminal,
  setupConnection,
  setupIME,
  setupImagePaste,
  setupDataHandlers,
} from "./useTerminalInit";
import { useTerminalResize } from "./useTerminalResize";

interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
  disconnect: () => void;
}

interface UseTerminalResult {
  terminalRef: MutableRefObject<HTMLDivElement | null>;
  xtermRef: MutableRefObject<XTerm | null>;
  fitAddonRef: MutableRefObject<FitAddon | null>;
  connectionStatus: ConnectionStatus;
  isRunnerDisconnected: boolean;
  syncSize: () => void;
}

export function useTerminal(
  podKey: string,
  fontSize: number,
  isPodReady: boolean,
  isActive: boolean,
): UseTerminalResult {
  const terminalRef = useRef<HTMLDivElement | null>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const connectionRef = useRef<TerminalConnection | null>(null);
  const schedulerRef = useRef<TerminalWriteScheduler | null>(null);
  const disposablesRef = useRef<IDisposable[]>([]);
  const lastSyncedSizeRef = useRef<{ cols: number; rows: number } | null>(null);
  const [isTerminalReady, setIsTerminalReady] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("connecting");
  const [isRunnerDisconnected, setIsRunnerDisconnected] = useState(false);

  useEffect(() => {
    if (!terminalRef.current || xtermRef.current || !isPodReady) return;

    const container = terminalRef.current;

    const { term, fitAddon, scheduler, deferredFitRafId } =
      setupTerminal(container, podKey, fontSize, lastSyncedSizeRef);

    schedulerRef.current = scheduler;

    const { abort, unsubscribeStatus } = setupConnection(
      podKey, scheduler, connectionRef,
      setConnectionStatus, setIsRunnerDisconnected,
    );

    const { isComposing } = setupIME(container, term, disposablesRef.current);

    setupImagePaste(container, connectionRef, disposablesRef.current);

    setupDataHandlers(term, podKey, connectionRef, isComposing, disposablesRef.current);

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;
    setIsTerminalReady(true);

    // connectionRef is read at cleanup — async-set in setupConnection.
    return () => {
      abort.abort();
      unsubscribeStatus();
      cancelAnimationFrame(deferredFitRafId);
      terminalRegistry.unregister(podKey);
      disposablesRef.current.forEach((d) => d.dispose());
      disposablesRef.current = [];
      // eslint-disable-next-line react-hooks/exhaustive-deps -- async-set ref, intentional
      const connection = connectionRef.current;
      connection?.unsubscribe();
      schedulerRef.current?.dispose();
      schedulerRef.current = null;
      term.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
      lastSyncedSizeRef.current = null;
      setIsTerminalReady(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [podKey, isPodReady]);

  const { syncSize } = useTerminalResize(
    podKey, fitAddonRef, xtermRef, terminalRef, isActive, fontSize, isTerminalReady,
  );

  return {
    terminalRef,
    xtermRef,
    fitAddonRef,
    connectionStatus,
    isRunnerDisconnected,
    syncSize,
  };
}
