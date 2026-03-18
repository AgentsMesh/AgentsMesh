"use client";

import { useEffect, useRef, useState, MutableRefObject } from "react";
import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";
import { terminalRegistry } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/terminalConnection";
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

/**
 * Facade hook for initializing and managing an xterm.js terminal.
 *
 * Composes:
 * - useTerminalInit setup functions (xterm, connection, IME, paste, data handlers)
 * - useTerminalResize (debounced sync, ResizeObserver, visibility, focus, font size)
 */
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
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("connecting");
  const [isRunnerDisconnected, setIsRunnerDisconnected] = useState(false);

  // Main effect: create terminal, connect, wire handlers — single lifecycle
  useEffect(() => {
    if (!terminalRef.current || xtermRef.current || !isPodReady) return;

    const container = terminalRef.current;

    // 1. Create XTerm + addons
    const { term, fitAddon, scheduler, initialDims, deferredFitRafId } =
      setupTerminal(container, podKey, fontSize, lastSyncedSizeRef);

    schedulerRef.current = scheduler;

    // 2. Connect to WebSocket pool
    const { abort, unsubscribeStatus } = setupConnection(
      podKey, scheduler, initialDims, connectionRef,
      setConnectionStatus, setIsRunnerDisconnected,
    );

    // 3. IME composition tracking
    const { isComposing } = setupIME(container, term, disposablesRef.current);

    // 4. Image paste support
    setupImagePaste(container, connectionRef, disposablesRef.current);

    // 5. Data & resize handlers
    setupDataHandlers(term, podKey, connectionRef, isComposing, disposablesRef.current);

    // Store refs
    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Cleanup — connectionRef is read at cleanup time intentionally because
    // it's set asynchronously in setupConnection after the effect body runs.
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
    };
    // fontSize intentionally excluded — handled by useTerminalResize
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [podKey, isPodReady]);

  // Delegate all resize concerns to the resize hook
  const { syncSize } = useTerminalResize(
    podKey, fitAddonRef, xtermRef, terminalRef, isActive, fontSize,
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
