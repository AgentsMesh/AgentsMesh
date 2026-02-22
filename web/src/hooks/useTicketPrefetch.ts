"use client";

import { useCallback, useRef } from "react";
import { ticketApi } from "@/lib/api";

// Cache for prefetched ticket data
const prefetchCache = new Map<string, { data: unknown; timestamp: number }>();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

// Pending prefetch requests to avoid duplicates
const pendingRequests = new Set<string>();

/**
 * Hook for prefetching ticket details on hover.
 * Uses a simple in-memory cache with TTL.
 */
export function useTicketPrefetch() {
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Check if data is cached and still valid
   */
  const isCached = useCallback((slug: string): boolean => {
    const cached = prefetchCache.get(slug);
    if (!cached) return false;

    const isExpired = Date.now() - cached.timestamp > CACHE_TTL;
    if (isExpired) {
      prefetchCache.delete(slug);
      return false;
    }
    return true;
  }, []);

  /**
   * Get cached data if available
   */
  const getCached = useCallback(<T>(slug: string): T | null => {
    if (!isCached(slug)) return null;
    return prefetchCache.get(slug)?.data as T;
  }, [isCached]);

  /**
   * Prefetch ticket details after a short delay (to avoid prefetching on quick hovers)
   */
  const prefetchOnHover = useCallback((slug: string) => {
    // Clear any existing timeout
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }

    // Skip if already cached or pending
    if (isCached(slug) || pendingRequests.has(slug)) {
      return;
    }

    // Delay prefetch by 150ms to avoid unnecessary requests on quick hovers
    hoverTimeoutRef.current = setTimeout(async () => {
      if (isCached(slug) || pendingRequests.has(slug)) {
        return;
      }

      pendingRequests.add(slug);

      try {
        // Prefetch main ticket data
        const ticketData = await ticketApi.get(slug);
        prefetchCache.set(slug, {
          data: ticketData,
          timestamp: Date.now(),
        });

        // Also prefetch related data in parallel
        const [subTickets, relations, commits] = await Promise.allSettled([
          ticketApi.getSubTickets(slug),
          ticketApi.listRelations(slug),
          ticketApi.listCommits(slug),
        ]);

        // Cache related data
        if (subTickets.status === "fulfilled") {
          prefetchCache.set(`${slug}:subTickets`, {
            data: subTickets.value,
            timestamp: Date.now(),
          });
        }
        if (relations.status === "fulfilled") {
          prefetchCache.set(`${slug}:relations`, {
            data: relations.value,
            timestamp: Date.now(),
          });
        }
        if (commits.status === "fulfilled") {
          prefetchCache.set(`${slug}:commits`, {
            data: commits.value,
            timestamp: Date.now(),
          });
        }
      } catch (error) {
        // Silently fail - prefetch is best-effort
        console.debug("Prefetch failed for:", slug, error);
      } finally {
        pendingRequests.delete(slug);
      }
    }, 150);
  }, [isCached]);

  /**
   * Cancel any pending prefetch (call on mouse leave)
   */
  const cancelPrefetch = useCallback(() => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
  }, []);

  /**
   * Clear all cached data
   */
  const clearCache = useCallback(() => {
    prefetchCache.clear();
  }, []);

  return {
    prefetchOnHover,
    cancelPrefetch,
    getCached,
    isCached,
    clearCache,
  };
}

export default useTicketPrefetch;
