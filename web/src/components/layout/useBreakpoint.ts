"use client";

import { useSyncExternalStore, useMemo } from "react";

/**
 * Responsive breakpoints
 * - mobile: < 768px
 * - tablet: 768px - 1024px
 * - desktop: > 1024px
 */
export type Breakpoint = "mobile" | "tablet" | "desktop";

export interface BreakpointConfig {
  mobile: number;
  tablet: number;
  desktop: number;
}

const DEFAULT_BREAKPOINTS: BreakpointConfig = {
  mobile: 0,
  tablet: 768,
  desktop: 1024,
};

function getBreakpoint(width: number, config: BreakpointConfig): Breakpoint {
  if (width >= config.desktop) {
    return "desktop";
  }
  if (width >= config.tablet) {
    return "tablet";
  }
  return "mobile";
}

// Get current window width
function getWidthSnapshot(): number {
  return typeof window !== "undefined" ? window.innerWidth : 1200;
}

// Server snapshot
function getServerWidthSnapshot(): number {
  return 1200; // Default to desktop width for SSR
}

// Subscribe to resize events
function subscribeToResize(callback: () => void): () => void {
  if (typeof window === "undefined") return () => {};

  window.addEventListener("resize", callback);
  window.addEventListener("orientationchange", callback);

  return () => {
    window.removeEventListener("resize", callback);
    window.removeEventListener("orientationchange", callback);
  };
}

/**
 * Hook to detect current responsive breakpoint
 * Returns the current breakpoint based on window width
 */
export function useBreakpoint(
  config: BreakpointConfig = DEFAULT_BREAKPOINTS
): {
  breakpoint: Breakpoint;
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  width: number;
} {
  const width = useSyncExternalStore(
    subscribeToResize,
    getWidthSnapshot,
    getServerWidthSnapshot
  );

  const result = useMemo(() => {
    const breakpoint = getBreakpoint(width, config);
    return {
      breakpoint,
      isMobile: breakpoint === "mobile",
      isTablet: breakpoint === "tablet",
      isDesktop: breakpoint === "desktop",
      width,
    };
  }, [width, config]);

  return result;
}

/**
 * Hook to check if current breakpoint matches or is larger than the specified breakpoint
 */
export function useMinBreakpoint(
  minBreakpoint: Breakpoint,
  config: BreakpointConfig = DEFAULT_BREAKPOINTS
): boolean {
  const { breakpoint } = useBreakpoint(config);

  const order: Breakpoint[] = ["mobile", "tablet", "desktop"];
  const currentIndex = order.indexOf(breakpoint);
  const minIndex = order.indexOf(minBreakpoint);

  return currentIndex >= minIndex;
}

/**
 * Hook to check if current breakpoint matches or is smaller than the specified breakpoint
 */
export function useMaxBreakpoint(
  maxBreakpoint: Breakpoint,
  config: BreakpointConfig = DEFAULT_BREAKPOINTS
): boolean {
  const { breakpoint } = useBreakpoint(config);

  const order: Breakpoint[] = ["mobile", "tablet", "desktop"];
  const currentIndex = order.indexOf(breakpoint);
  const maxIndex = order.indexOf(maxBreakpoint);

  return currentIndex <= maxIndex;
}

export { DEFAULT_BREAKPOINTS };
