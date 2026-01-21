"use client";

import { useEffect, MutableRefObject } from "react";
import { Terminal as XTerm } from "@xterm/xterm";

/**
 * Hook for enabling touch scrolling in xterm.js
 * xterm.js doesn't natively support touch scrolling, so we implement it manually
 * Reference: https://github.com/xtermjs/xterm.js/issues/5377
 */
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

      // Calculate lines to scroll (divide by ~10 for smooth scrolling)
      const linesToScroll = Math.round(deltaY / 10);
      if (linesToScroll !== 0) {
        xtermRef.current.scrollLines(linesToScroll);
        // Only prevent default when terminal has scrollable content
        // This allows input events to propagate when there's no scroll buffer
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
