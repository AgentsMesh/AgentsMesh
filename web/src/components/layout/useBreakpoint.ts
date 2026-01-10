"use client";

import { useState, useEffect, useCallback } from "react";

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
  // SSR safe initial state - default to desktop for SSR
  const [state, setState] = useState<{ breakpoint: Breakpoint; width: number }>(
    () => ({
      breakpoint: "desktop",
      width: typeof window !== "undefined" ? window.innerWidth : 1200,
    })
  );

  const handleResize = useCallback(() => {
    const width = window.innerWidth;
    const breakpoint = getBreakpoint(width, config);
    setState((prev) => {
      if (prev.breakpoint === breakpoint && prev.width === width) {
        return prev;
      }
      return { breakpoint, width };
    });
  }, [config]);

  useEffect(() => {
    // Initialize on mount
    handleResize();

    // Listen for resize events
    window.addEventListener("resize", handleResize);

    // Optional: Listen for orientation change on mobile
    window.addEventListener("orientationchange", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("orientationchange", handleResize);
    };
  }, [handleResize]);

  return {
    breakpoint: state.breakpoint,
    isMobile: state.breakpoint === "mobile",
    isTablet: state.breakpoint === "tablet",
    isDesktop: state.breakpoint === "desktop",
    width: state.width,
  };
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
