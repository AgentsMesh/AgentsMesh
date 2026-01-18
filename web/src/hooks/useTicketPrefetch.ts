"use client";

import { useCallback, useRef } from "react";
import { ticketApi } from "@/lib/api/client";

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
  const isCached = useCallback((identifier: string): boolean => {
    const cached = prefetchCache.get(identifier);
    if (!cached) return false;

    const isExpired = Date.now() - cached.timestamp > CACHE_TTL;
    if (isExpired) {
      prefetchCache.delete(identifier);
      return false;
    }
    return true;
  }, []);

  /**
   * Get cached data if available
   */
  const getCached = useCallback(<T>(identifier: string): T | null => {
    if (!isCached(identifier)) return null;
    return prefetchCache.get(identifier)?.data as T;
  }, [isCached]);

  /**
   * Prefetch ticket details after a short delay (to avoid prefetching on quick hovers)
   */
  const prefetchOnHover = useCallback((identifier: string) => {
    // Clear any existing timeout
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }

    // Skip if already cached or pending
    if (isCached(identifier) || pendingRequests.has(identifier)) {
      return;
    }

    // Delay prefetch by 150ms to avoid unnecessary requests on quick hovers
    hoverTimeoutRef.current = setTimeout(async () => {
      if (isCached(identifier) || pendingRequests.has(identifier)) {
        return;
      }

      pendingRequests.add(identifier);

      try {
        // Prefetch main ticket data
        const ticketData = await ticketApi.get(identifier);
        prefetchCache.set(identifier, {
          data: ticketData,
          timestamp: Date.now(),
        });

        // Also prefetch related data in parallel
        const [subTickets, relations, commits] = await Promise.allSettled([
          ticketApi.getSubTickets(identifier),
          ticketApi.listRelations(identifier),
          ticketApi.listCommits(identifier),
        ]);

        // Cache related data
        if (subTickets.status === "fulfilled") {
          prefetchCache.set(`${identifier}:subTickets`, {
            data: subTickets.value,
            timestamp: Date.now(),
          });
        }
        if (relations.status === "fulfilled") {
          prefetchCache.set(`${identifier}:relations`, {
            data: relations.value,
            timestamp: Date.now(),
          });
        }
        if (commits.status === "fulfilled") {
          prefetchCache.set(`${identifier}:commits`, {
            data: commits.value,
            timestamp: Date.now(),
          });
        }
      } catch (error) {
        // Silently fail - prefetch is best-effort
        console.debug("Prefetch failed for:", identifier, error);
      } finally {
        pendingRequests.delete(identifier);
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
