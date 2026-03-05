"use client";

import { useEffect, useRef, useState, useCallback, MutableRefObject } from "react";
import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { terminalPool, terminalRegistry } from "@/stores/workspace";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";

interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
  disconnect: () => void;
}

interface UseTerminalResult {
  terminalRef: MutableRefObject<HTMLDivElement | null>;
  xtermRef: MutableRefObject<XTerm | null>;
  fitAddonRef: MutableRefObject<FitAddon | null>;
  connectionStatus: "connecting" | "connected" | "disconnected" | "error";
  isRunnerDisconnected: boolean;
  syncSize: () => void;
}

/** Debounce delay for size sync operations (ms) */
const SIZE_SYNC_DEBOUNCE_MS = 100;

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
 * Safely fit terminal only when container has valid dimensions.
 * Uses FitAddon.proposeDimensions() to check before fitting.
 * Returns true if fit was successful, false if dimensions are invalid.
 *
 * @see https://github.com/xtermjs/xterm.js/issues/3029
 */
function safeFit(fitAddon: FitAddon): { cols: number; rows: number } | null {
  const dims = fitAddon.proposeDimensions();
  if (!dims || !Number.isFinite(dims.cols) || !Number.isFinite(dims.rows) || dims.cols <= 0 || dims.rows <= 0) {
    return null;
  }
  fitAddon.fit();
  return { cols: dims.cols, rows: dims.rows };
}

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
  const sizeSyncTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Track last synced size to avoid redundant resize messages
  const lastSyncedSizeRef = useRef<{ cols: number; rows: number } | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected" | "error">("connecting");
  const [isRunnerDisconnected, setIsRunnerDisconnected] = useState(false);

  /**
   * Debounced size sync to PTY.
   * Prevents excessive resize messages when switching panes rapidly or during animations.
   * Only sends if size actually changed.
   */
  const debouncedSizeSync = useCallback((cols: number, rows: number) => {
    // Skip if size hasn't changed
    const last = lastSyncedSizeRef.current;
    if (last && last.cols === cols && last.rows === rows) {
      return;
    }

    if (sizeSyncTimerRef.current) {
      clearTimeout(sizeSyncTimerRef.current);
    }
    sizeSyncTimerRef.current = setTimeout(() => {
      lastSyncedSizeRef.current = { cols, rows };
      terminalPool.forceResize(podKey, cols, rows);
      sizeSyncTimerRef.current = null;
    }, SIZE_SYNC_DEBOUNCE_MS);
  }, [podKey]);

  /**
   * Force immediate size sync to PTY (no debounce).
   * Used for initial connection and explicit sync requests.
   */
  const forceImmediateSizeSync = useCallback((cols: number, rows: number) => {
    if (cols <= 0 || rows <= 0) return;

    // Skip if size hasn't changed
    const last = lastSyncedSizeRef.current;
    if (last && last.cols === cols && last.rows === rows) {
      return;
    }

    // Clear any pending debounced sync
    if (sizeSyncTimerRef.current) {
      clearTimeout(sizeSyncTimerRef.current);
      sizeSyncTimerRef.current = null;
    }

    lastSyncedSizeRef.current = { cols, rows };
    terminalPool.forceResize(podKey, cols, rows);
  }, [podKey]);

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

    // Deferred initial fit — use requestAnimationFrame to ensure the container
    // layout is complete before measuring.  We store the dims for the
    // post-subscribe resize message below.
    // Store initial dimensions after layout completes. Typed as a mutable
    // container so TypeScript doesn't narrow to `never` inside the async IIFE.
    const initialDims: { value: { cols: number; rows: number } | null } = { value: null };
    const deferredFitRaf = requestAnimationFrame(() => {
      const dims = safeFit(fitAddon);
      if (dims) {
        initialDims.value = dims;
        lastSyncedSizeRef.current = dims;
      }
    });

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

    // Use stable subscriptionId so that StrictMode remount hits the idempotent
    // `hadPrevious` branch in subscribe(), updating the callback without
    // tearing down and recreating the WebSocket connection.
    const subscriptionId = `terminal-${podKey}`;

    // Async connection setup with AbortController for cleanup coordination.
    // When StrictMode unmounts during the async subscribe(), the abort signal
    // tells the resolved promise to skip storing the handle (but NOT to
    // unsubscribe), so the subscriber entry stays in the Map and the
    // remount's subscribe() hits the idempotent `hadPrevious` branch.
    const abortController = new AbortController();
    (async () => {
      try {
        const handle = await terminalPool.subscribe(podKey, subscriptionId, handleMessage);
        if (abortController.signal.aborted) {
          // Effect was cleaned up while subscribe was in flight.
          // Don't unsubscribe — the remount's subscribe will reuse the
          // same subscriptionId and update the callback idempotently.
          return;
        }
        connectionRef.current = handle;
        // Send initial resize after connection is established
        // This ensures the resize is sent after WebSocket is connected
        if (initialDims.value) {
          terminalPool.forceResize(podKey, initialDims.value.cols, initialDims.value.rows);
        }
      } catch (error) {
        if (abortController.signal.aborted) return;
        console.error("Failed to connect terminal:", error);
        setConnectionStatus("error");
      }
    })();

    // Subscribe to connection status changes (event-based, real-time)
    const unsubscribeStatus = terminalPool.onStatusChange(podKey, (info) => {
      if (info.status !== "none") {
        setConnectionStatus(info.status);
      }
      setIsRunnerDisconnected(info.runnerDisconnected);
    });

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
      const initialSyncRafId = requestAnimationFrame(syncTextareaPosition);

      disposablesRef.current.push(
        cursorDisposable,
        writeDisposable,
        { dispose: () => cancelAnimationFrame(initialSyncRafId) },
      );
    }

    // Image paste support: intercept paste events with image data
    const handlePaste = (e: ClipboardEvent) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.type.startsWith('image/')) {
          e.preventDefault();
          e.stopPropagation();
          const blob = item.getAsFile();
          if (!blob) continue;

          if (blob.size > 2 * 1024 * 1024) {
            console.warn('Image too large for paste (max 2MB)');
            return;
          }

          blob.arrayBuffer().then((buffer) => {
            const imageData = new Uint8Array(buffer);
            const success = terminalPool.sendImage(podKey, imageData, item.type);
            if (!success) {
              console.warn('Failed to send image paste');
            }
          });
          return; // Only handle first image
        }
      }
      // No image found — let xterm.js handle normal text paste
    };

    // Listen on the terminal container div (captures before xterm's textarea)
    const containerEl = terminalRef.current;
    containerEl.addEventListener('paste', handlePaste, true);
    disposablesRef.current.push({ dispose: () => containerEl.removeEventListener('paste', handlePaste, true) });

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

    // --- ResizeObserver: bound to terminal lifecycle to guarantee creation ---
    // Previously in a separate effect whose deps could desync from terminal creation.
    const resizeObserver = new ResizeObserver(() => {
      const dims = safeFit(fitAddon);
      if (dims) {
        debouncedSizeSync(dims.cols, dims.rows);
      }
    });
    resizeObserver.observe(terminalRef.current);

    const handleWindowResize = () => {
      const dims = safeFit(fitAddon);
      if (dims) {
        debouncedSizeSync(dims.cols, dims.rows);
      }
    };
    window.addEventListener("resize", handleWindowResize);

    // Cleanup
    return () => {
      abortController.abort();  // Prevent late async subscribe from storing handle
      unsubscribeStatus();
      cancelAnimationFrame(deferredFitRaf);
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleWindowResize);
      // Clear any pending size sync timer
      if (sizeSyncTimerRef.current) {
        clearTimeout(sizeSyncTimerRef.current);
        sizeSyncTimerRef.current = null;
      }
      // Unregister terminal from registry
      terminalRegistry.unregister(podKey);
      // Explicitly dispose event listeners before disposing terminal
      disposablesRef.current.forEach(d => d.dispose());
      disposablesRef.current = [];
      // Unsubscribe from terminal data - connection stays open if other subscribers exist
      // or for 30s delay allowing quick re-subscribe
      connectionRef.current?.unsubscribe();
      // Dispose scheduler before terminal to ensure no pending writes
      schedulerRef.current?.dispose();
      schedulerRef.current = null;
      term.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
      lastSyncedSizeRef.current = null;
    };
    // Note: fontSize is intentionally excluded from dependencies to prevent terminal recreation
    // Font size changes are handled separately in another useEffect below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [podKey, isPodReady]);

  /**
   * Handle page visibility change.
   * When browser tab becomes visible again, ensure terminal is properly sized.
   */
  useEffect(() => {
    let rafId: number | undefined;

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible" && isActive) {
        const fitAddon = fitAddonRef.current;
        if (!fitAddon) return;

        // Delay slightly to allow browser to update layout
        rafId = requestAnimationFrame(() => {
          const dims = safeFit(fitAddon);
          if (dims) {
            debouncedSizeSync(dims.cols, dims.rows);
          }
        });
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      if (rafId !== undefined) cancelAnimationFrame(rafId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isActive, debouncedSizeSync]);

  /**
   * Focus terminal and sync size when pane becomes active.
   * This handles tab switching within the application.
   */
  useEffect(() => {
    let rafId: number | undefined;

    if (isActive && xtermRef.current) {
      xtermRef.current.focus();

      const fitAddon = fitAddonRef.current;
      if (fitAddon) {
        // Fit after next paint to ensure layout is complete
        rafId = requestAnimationFrame(() => {
          const dims = safeFit(fitAddon);
          if (dims) {
            // Force immediate sync when pane becomes active
            // to ensure PTY size matches terminal display
            forceImmediateSizeSync(dims.cols, dims.rows);
          }
        });
      }
    }

    return () => {
      if (rafId !== undefined) cancelAnimationFrame(rafId);
    };
  }, [isActive, forceImmediateSizeSync]);

  // Update font size
  useEffect(() => {
    const term = xtermRef.current;
    const fitAddon = fitAddonRef.current;
    if (term && fitAddon) {
      term.options.fontSize = fontSize;
      const dims = safeFit(fitAddon);
      if (dims) {
        debouncedSizeSync(dims.cols, dims.rows);
      }
    }
  }, [fontSize, debouncedSizeSync]);

  // Sync terminal size to PTY (manual trigger)
  const syncSize = useCallback(() => {
    const term = xtermRef.current;
    if (term && term.cols > 0 && term.rows > 0) {
      forceImmediateSizeSync(term.cols, term.rows);
    }
  }, [forceImmediateSizeSync]);

  return {
    terminalRef,
    xtermRef,
    fitAddonRef,
    connectionStatus,
    isRunnerDisconnected,
    syncSize,
  };
}
