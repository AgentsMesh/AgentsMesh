"use client";

import { useEffect, useRef, useState, MutableRefObject } from "react";
import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { terminalPool, terminalRegistry } from "@/stores/workspace";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";

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
  const schedulerRef = useRef<TerminalWriteScheduler | null>(null);
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

    // Create write scheduler to aggregate high-frequency writes
    // This reduces xterm.write() calls from 4000-6700/s to ~60/s
    const scheduler = new TerminalWriteScheduler();
    scheduler.attach(term);
    schedulerRef.current = scheduler;

    // Connect to WebSocket pool (async for Relay mode)
    // Use scheduler to batch writes into animation frames
    const handleMessage = (data: Uint8Array | string) => {
      if (data instanceof Uint8Array) {
        scheduler.schedule(data);
      } else {
        scheduler.schedule(new TextEncoder().encode(data));
      }
    };

    // Async connection setup
    let isMounted = true;
    (async () => {
      try {
        const handle = await terminalPool.connect(podKey, handleMessage);
        if (isMounted) {
          connectionRef.current = handle;
        } else {
          // Component unmounted before connection completed
          handle.disconnect();
        }
      } catch (error) {
        console.error("Failed to connect terminal:", error);
        if (isMounted) {
          setConnectionStatus("error");
        }
      }
    })();

    // Update connection status
    const checkStatus = () => {
      const status = terminalPool.getStatus(podKey);
      if (status !== "none") {
        setConnectionStatus(status);
      }
    };
    checkStatus();
    const statusInterval = setInterval(checkStatus, 1000);

    // IME composition state tracking
    // During composition (e.g., Chinese input), we should not send partial data
    // to prevent duplicate input issues on mobile (especially Android + GBoard)
    let isComposing = false;

    const textarea = terminalRef.current.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement;
    if (textarea) {
      const handleCompositionStart = () => {
        isComposing = true;
      };

      const handleCompositionEnd = () => {
        isComposing = false;
      };

      textarea.addEventListener('compositionstart', handleCompositionStart);
      textarea.addEventListener('compositionend', handleCompositionEnd);

      // Store cleanup functions
      const compositionCleanup = () => {
        textarea.removeEventListener('compositionstart', handleCompositionStart);
        textarea.removeEventListener('compositionend', handleCompositionEnd);
      };
      // Add to disposables for cleanup
      disposablesRef.current.push({ dispose: compositionCleanup });

      // Mobile cursor position sync
      // On mobile, the hidden textarea needs to follow cursor position
      // to help virtual keyboard and IME work correctly
      // See: https://github.com/xtermjs/xterm.js/issues/2598
      const syncTextareaPosition = () => {
        const cursorX = term.buffer.active.cursorX;
        const cursorY = term.buffer.active.cursorY - term.buffer.active.viewportY;

        // Calculate pixel position based on font metrics
        // Use actual cell dimensions from xterm's internal rendering
        const cellWidth = term.options.fontSize! * 0.6; // Approximate monospace ratio
        const cellHeight = term.options.fontSize! * (term.options.lineHeight || 1.2);

        // Position textarea near cursor (helps mobile IME positioning)
        textarea.style.left = `${Math.max(0, cursorX * cellWidth)}px`;
        textarea.style.top = `${Math.max(0, cursorY * cellHeight)}px`;
      };

      // Sync on cursor move and after writes
      const cursorDisposable = term.onCursorMove(syncTextareaPosition);
      const writeDisposable = term.onWriteParsed(syncTextareaPosition);

      // Initial sync after terminal is rendered
      requestAnimationFrame(syncTextareaPosition);

      disposablesRef.current.push(cursorDisposable, writeDisposable);
    }

    // Handle input - save disposable for cleanup
    // Note: xterm.js onData fires after compositionend, so checking isComposing
    // helps filter out any edge cases where data might be sent during composition
    const dataDisposable = term.onData((data) => {
      // Skip sending if still composing (edge case protection)
      if (isComposing) return;
      connectionRef.current?.send(data);
    });

    // Handle resize - save disposable for cleanup
    const resizeDisposable = term.onResize(({ rows, cols }) => {
      terminalPool.sendResize(podKey, cols, rows);
    });

    // Add remaining disposables (don't overwrite, push to existing array)
    disposablesRef.current.push(dataDisposable, resizeDisposable);

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Register terminal instance for cross-component access (e.g., TerminalToolbar)
    terminalRegistry.register(podKey, term);

    // Cleanup
    return () => {
      isMounted = false;  // Prevent late connection from being stored
      clearInterval(statusInterval);
      // Unregister terminal from registry
      terminalRegistry.unregister(podKey);
      // Explicitly dispose event listeners before disposing terminal
      disposablesRef.current.forEach(d => d.dispose());
      disposablesRef.current = [];
      connectionRef.current?.disconnect();
      // Dispose scheduler before terminal to ensure no pending writes
      schedulerRef.current?.dispose();
      schedulerRef.current = null;
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
      terminalPool.forceResize(podKey, xtermRef.current.cols, xtermRef.current.rows);
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
