"use client";

import { useEffect, useRef, useState, MutableRefObject } from "react";
import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { terminalPool } from "@/stores/workspace";

interface TerminalConnection {
  send: (data: string) => void;
  disconnect: () => void;
}

interface UseTerminalResult {
  terminalRef: MutableRefObject<HTMLDivElement | null>;
  xtermRef: MutableRefObject<XTerm | null>;
  fitAddonRef: MutableRefObject<FitAddon | null>;
  connectionStatus: "connecting" | "connected" | "disconnected" | "error";
  syncSize: () => void;
}

const TERMINAL_THEME = {
  background: "#1e1e1e",
  foreground: "#d4d4d4",
  cursor: "#d4d4d4",
  cursorAccent: "#1e1e1e",
  selectionBackground: "#264f78",
  black: "#000000",
  red: "#cd3131",
  green: "#0dbc79",
  yellow: "#e5e510",
  blue: "#2472c8",
  magenta: "#bc3fbc",
  cyan: "#11a8cd",
  white: "#e5e5e5",
  brightBlack: "#666666",
  brightRed: "#f14c4c",
  brightGreen: "#23d18b",
  brightYellow: "#f5f543",
  brightBlue: "#3b8eea",
  brightMagenta: "#d670d6",
  brightCyan: "#29b8db",
  brightWhite: "#e5e5e5",
};

/**
 * Hook for initializing and managing xterm.js terminal
 */
export function useTerminal(
  podKey: string,
  fontSize: number,
  isPodReady: boolean,
  isActive: boolean
): UseTerminalResult {
  const terminalRef = useRef<HTMLDivElement | null>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const connectionRef = useRef<TerminalConnection | null>(null);
  const disposablesRef = useRef<IDisposable[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected" | "error">("connecting");

  // Initialize terminal (only when Pod is ready)
  useEffect(() => {
    if (!terminalRef.current || xtermRef.current || !isPodReady) return;

    const term = new XTerm({
      cursorBlink: true,
      cursorStyle: "block",
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: fontSize,
      lineHeight: 1.2,
      theme: TERMINAL_THEME,
      allowProposedApi: true,
    });

    // Add addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    const searchAddon = new SearchAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.loadAddon(searchAddon);

    // Open terminal
    // Note: Terminal state will be restored from backend via WebSocket on connect
    term.open(terminalRef.current);

    // Fit after a short delay to ensure container is sized
    setTimeout(() => {
      fitAddon.fit();
    }, 50);

    // Connect to WebSocket pool
    const handleMessage = (data: Uint8Array | string) => {
      if (data instanceof Uint8Array) {
        term.write(data);
      } else {
        term.write(data);
      }
    };

    connectionRef.current = terminalPool.connect(podKey, handleMessage);

    // Update connection status
    const checkStatus = () => {
      const status = terminalPool.getStatus(podKey);
      if (status !== "none") {
        setConnectionStatus(status);
      }
    };
    checkStatus();
    const statusInterval = setInterval(checkStatus, 1000);

    // Handle input - save disposable for cleanup
    const dataDisposable = term.onData((data) => {
      connectionRef.current?.send(data);
    });

    // Handle resize - save disposable for cleanup
    const resizeDisposable = term.onResize(({ rows, cols }) => {
      terminalPool.sendResize(podKey, rows, cols);
    });

    // Store disposables for explicit cleanup
    disposablesRef.current = [dataDisposable, resizeDisposable];

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Cleanup
    return () => {
      clearInterval(statusInterval);
      // Explicitly dispose event listeners before disposing terminal
      disposablesRef.current.forEach(d => d.dispose());
      disposablesRef.current = [];
      connectionRef.current?.disconnect();
      term.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
    // Note: fontSize is intentionally excluded from dependencies to prevent terminal recreation
    // Font size changes are handled separately in another useEffect below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [podKey, isPodReady]);

  // Handle container resize
  useEffect(() => {
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
      }
    };

    // Observe container size changes
    const resizeObserver = new ResizeObserver(handleResize);
    if (terminalRef.current?.parentElement) {
      resizeObserver.observe(terminalRef.current.parentElement);
    }

    window.addEventListener("resize", handleResize);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  // Focus terminal when pane becomes active
  useEffect(() => {
    if (isActive && xtermRef.current) {
      xtermRef.current.focus();
      fitAddonRef.current?.fit();
    }
  }, [isActive]);

  // Update font size
  useEffect(() => {
    if (xtermRef.current) {
      xtermRef.current.options.fontSize = fontSize;
      fitAddonRef.current?.fit();
    }
  }, [fontSize]);

  // Sync terminal size to PTY
  const syncSize = () => {
    if (xtermRef.current) {
      terminalPool.forceResize(podKey, xtermRef.current.rows, xtermRef.current.cols);
    }
  };

  return {
    terminalRef,
    xtermRef,
    fitAddonRef,
    connectionStatus,
    syncSize,
  };
}
