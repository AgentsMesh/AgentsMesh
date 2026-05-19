"use client";

import { useEffect, useCallback, useRef, MutableRefObject } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { relayPool } from "@/stores/workspace";
import { safeFit } from "./useTerminalInit";

const SIZE_SYNC_DEBOUNCE_MS = 100;

export function useTerminalResize(
  podKey: string,
  fitAddonRef: MutableRefObject<FitAddon | null>,
  xtermRef: MutableRefObject<XTerm | null>,
  containerRef: MutableRefObject<HTMLDivElement | null>,
  isActive: boolean,
  fontSize: number,
  isTerminalReady: boolean = false,
): { syncSize: () => void } {
  const sizeSyncTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastSyncedSizeRef = useRef<{ cols: number; rows: number } | null>(null);

  const debouncedSizeSync = useCallback((cols: number, rows: number) => {
    const last = lastSyncedSizeRef.current;
    if (last && last.cols === cols && last.rows === rows) return;

    if (sizeSyncTimerRef.current) {
      clearTimeout(sizeSyncTimerRef.current);
    }
    sizeSyncTimerRef.current = setTimeout(() => {
      lastSyncedSizeRef.current = { cols, rows };
      relayPool.forceResize(podKey, cols, rows);
      sizeSyncTimerRef.current = null;
    }, SIZE_SYNC_DEBOUNCE_MS);
  }, [podKey]);

  const forceImmediateSizeSync = useCallback((cols: number, rows: number) => {
    if (cols <= 0 || rows <= 0) return;
    const last = lastSyncedSizeRef.current;
    if (last && last.cols === cols && last.rows === rows) return;

    if (sizeSyncTimerRef.current) {
      clearTimeout(sizeSyncTimerRef.current);
      sizeSyncTimerRef.current = null;
    }
    lastSyncedSizeRef.current = { cols, rows };
    relayPool.forceResize(podKey, cols, rows);
  }, [podKey]);

  useEffect(() => {
    const fitAddon = fitAddonRef.current;
    const container = containerRef.current;
    if (!fitAddon || !container) return;

    const resizeObserver = new ResizeObserver(() => {
      const dims = safeFit(fitAddon);
      if (dims) debouncedSizeSync(dims.cols, dims.rows);
    });
    resizeObserver.observe(container);

    const handleWindowResize = () => {
      const dims = safeFit(fitAddon);
      if (dims) debouncedSizeSync(dims.cols, dims.rows);
    };
    window.addEventListener("resize", handleWindowResize);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleWindowResize);
      if (sizeSyncTimerRef.current) {
        clearTimeout(sizeSyncTimerRef.current);
        sizeSyncTimerRef.current = null;
      }
    };
  }, [fitAddonRef, containerRef, debouncedSizeSync, isTerminalReady]);

  useEffect(() => {
    let rafId: number | undefined;

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible" && isActive) {
        const fitAddon = fitAddonRef.current;
        if (!fitAddon) return;
        rafId = requestAnimationFrame(() => {
          const dims = safeFit(fitAddon);
          if (dims) debouncedSizeSync(dims.cols, dims.rows);
        });
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      if (rafId !== undefined) cancelAnimationFrame(rafId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isActive, fitAddonRef, debouncedSizeSync]);

  useEffect(() => {
    let rafId: number | undefined;

    if (isActive && xtermRef.current) {
      xtermRef.current.focus();
      const fitAddon = fitAddonRef.current;
      if (fitAddon) {
        rafId = requestAnimationFrame(() => {
          const dims = safeFit(fitAddon);
          if (dims) forceImmediateSizeSync(dims.cols, dims.rows);
        });
      }
    }

    return () => {
      if (rafId !== undefined) cancelAnimationFrame(rafId);
    };
  }, [isActive, xtermRef, fitAddonRef, forceImmediateSizeSync]);

  useEffect(() => {
    const term = xtermRef.current;
    const fitAddon = fitAddonRef.current;
    if (term && fitAddon) {
      term.options.fontSize = fontSize;
      const dims = safeFit(fitAddon);
      if (dims) debouncedSizeSync(dims.cols, dims.rows);
    }
  }, [fontSize, xtermRef, fitAddonRef, debouncedSizeSync]);

  const syncSize = useCallback(() => {
    const term = xtermRef.current;
    if (term && term.cols > 0 && term.rows > 0) {
      forceImmediateSizeSync(term.cols, term.rows);
    }
  }, [xtermRef, forceImmediateSizeSync]);

  return { syncSize };
}
