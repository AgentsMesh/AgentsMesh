"use client";

import { useEffect, MutableRefObject } from "react";
import { Terminal as XTerm } from "@xterm/xterm";

// Workaround: xterm.js#5377 — no native touch scroll, implement manually.
export function useTouchScroll(
  containerRef: MutableRefObject<HTMLDivElement | null>,
  xtermRef: MutableRefObject<XTerm | null>,
  enabled: boolean
): void {
  useEffect(() => {
    if (!containerRef.current || !xtermRef.current || !enabled) return;

    let lastTouchY: number | null = null;

    const handleTouchStart = (e: TouchEvent) => {
      if (e.touches.length === 1) {
        lastTouchY = e.touches[0].clientY;
      }
    };

    const handleTouchEnd = () => {
      lastTouchY = null;
    };

    const handleTouchMove = (e: TouchEvent) => {
      if (e.touches.length !== 1 || lastTouchY === null || !xtermRef.current) return;

      const currentY = e.touches[0].clientY;
      const deltaY = lastTouchY - currentY;
      lastTouchY = currentY;

      const linesToScroll = Math.round(deltaY / 10);
      if (linesToScroll !== 0) {
        xtermRef.current.scrollLines(linesToScroll);
        const viewport = containerRef.current?.querySelector('.xterm-viewport');
        if (viewport && viewport.scrollHeight > viewport.clientHeight) {
          e.preventDefault();
        }
      }
    };

    const container = containerRef.current;
    container.addEventListener('touchstart', handleTouchStart, { passive: true });
    container.addEventListener('touchend', handleTouchEnd, { passive: true });
    container.addEventListener('touchmove', handleTouchMove, { passive: false });

    return () => {
      container.removeEventListener('touchstart', handleTouchStart);
      container.removeEventListener('touchend', handleTouchEnd);
      container.removeEventListener('touchmove', handleTouchMove);
    };
  }, [containerRef, xtermRef, enabled]);
}
