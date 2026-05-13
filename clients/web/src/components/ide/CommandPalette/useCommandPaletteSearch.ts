"use client";

import { useState, useEffect } from "react";
import { getPodService } from "@/lib/wasm-core";
import { listTickets } from "@/lib/api/ticketConnect";
import { listRepositories } from "@/lib/api/repositoryConnect";
import { useCurrentOrg } from "@/stores/auth";
import type { SearchResults, PodSearchResult, TicketSearchResult, RepositorySearchResult } from "./types";

/**
 * Custom hook for managing command palette search
 */
export function useCommandPaletteSearch(search: string): SearchResults {
  const currentOrg = useCurrentOrg();
  const [pods, setPods] = useState<PodSearchResult[]>([]);
  const [tickets, setTickets] = useState<TicketSearchResult[]>([]);
  const [repositories, setRepositories] = useState<RepositorySearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!search || search.length < 2) {
      setPods([]);
      setTickets([]);
      setRepositories([]);
      return;
    }

    const loadSearchResults = async () => {
      setLoading(true);
      try {
        const reposPromise = currentOrg
          ? listRepositories(currentOrg.slug).catch(() => ({ items: [] }))
          : Promise.resolve({ items: [] });
        const ticketsPromise = currentOrg
          ? listTickets(currentOrg.slug, { limit: 500 }).catch(() => ({ items: [], total: 0, limit: 0, offset: 0 }))
          : Promise.resolve({ items: [], total: 0, limit: 0, offset: 0 });
        const [podsRes, ticketsRes, reposRes] = await Promise.all([
          getPodService().fetch_pods(null, null, null, null, null).then((j: string) => JSON.parse(j)).catch(() => ({ pods: [] })),
          ticketsPromise,
          reposPromise,
        ]);

        // Filter by search term
        const searchLower = search.toLowerCase();
        setPods(
          (podsRes.pods || [])
            .filter((p: { pod_key: string }) => p.pod_key.toLowerCase().includes(searchLower))
            .slice(0, 5)
        );
        setTickets(
          (ticketsRes.items || [])
            .filter(
              (ticket: { slug: string; title: string }) =>
                ticket.slug.toLowerCase().includes(searchLower) ||
                ticket.title.toLowerCase().includes(searchLower)
            )
            .slice(0, 5)
        );
        setRepositories(
          (reposRes.items || [])
            .filter((r: { slug: string }) => r.slug.toLowerCase().includes(searchLower))
            .slice(0, 5)
        );
      } catch (error) {
        console.error("Search error:", error);
      } finally {
        setLoading(false);
      }
    };

    const debounce = setTimeout(loadSearchResults, 300);
    return () => clearTimeout(debounce);
  }, [search, currentOrg]);

  return { pods, tickets, repositories, loading };
}
