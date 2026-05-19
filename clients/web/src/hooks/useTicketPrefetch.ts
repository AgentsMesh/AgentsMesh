"use client";

import { useCallback, useRef } from "react";
import { getTicketService, getTicketRelationsService } from "@/lib/wasm-core";

const prefetchCache = new Map<string, { data: unknown; timestamp: number }>();
const CACHE_TTL = 5 * 60 * 1000;

const pendingRequests = new Set<string>();

export function useTicketPrefetch() {
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);

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

  const getCached = useCallback(<T>(slug: string): T | null => {
    if (!isCached(slug)) return null;
    return prefetchCache.get(slug)?.data as T;
  }, [isCached]);

  const prefetchOnHover = useCallback((slug: string) => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }

    if (isCached(slug) || pendingRequests.has(slug)) {
      return;
    }

    hoverTimeoutRef.current = setTimeout(async () => {
      if (isCached(slug) || pendingRequests.has(slug)) {
        return;
      }

      pendingRequests.add(slug);

      try {
        const ticketData = JSON.parse(await getTicketService().fetch_ticket(slug));
        prefetchCache.set(slug, {
          data: ticketData,
          timestamp: Date.now(),
        });

        const [subTickets, relations, commits] = await Promise.allSettled([
          getTicketService().get_sub_tickets(slug).then((j: string) => JSON.parse(j)),
          getTicketRelationsService().list_relations(slug).then((j: string) => JSON.parse(j)),
          getTicketRelationsService().list_commits(slug).then((j: string) => JSON.parse(j)),
        ]);

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
        console.debug("Prefetch failed for:", slug, error);
      } finally {
        pendingRequests.delete(slug);
      }
    }, 150);
  }, [isCached]);

  const cancelPrefetch = useCallback(() => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
  }, []);

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
