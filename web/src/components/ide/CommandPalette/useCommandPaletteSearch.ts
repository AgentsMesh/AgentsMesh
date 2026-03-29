"use client";

import { useState, useEffect } from "react";
import { podApi, ticketApi, repositoryApi } from "@/lib/api";
import type { SearchResults, PodSearchResult, TicketSearchResult, RepositorySearchResult } from "./types";

/**
 * Custom hook for managing command palette search
 */
export function useCommandPaletteSearch(search: string): SearchResults {
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
        const [podsRes, ticketsRes, reposRes] = await Promise.all([
          podApi.list().catch(() => ({ pods: [] })),
          ticketApi.list().catch(() => ({ tickets: [] })),
          repositoryApi.list().catch(() => ({ repositories: [] })),
        ]);

        // Filter by search term
        const searchLower = search.toLowerCase();
        setPods(
          (podsRes.pods || [])
            .filter((p: { pod_key: string }) => p.pod_key.toLowerCase().includes(searchLower))
            .slice(0, 5)
        );
        setTickets(
          (ticketsRes.tickets || [])
            .filter(
              (ticket: { slug: string; title: string }) =>
                ticket.slug.toLowerCase().includes(searchLower) ||
                ticket.title.toLowerCase().includes(searchLower)
            )
            .slice(0, 5)
        );
        setRepositories(
          (reposRes.repositories || [])
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
  }, [search]);

  return { pods, tickets, repositories, loading };
}
