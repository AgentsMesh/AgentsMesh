"use client";

import { useState, useEffect } from "react";
import { getPodService, getTicketService } from "@/lib/wasm-core";
import { useRepositories, useRepositoryStore } from "@/stores/repository";
import type { SearchResults, PodSearchResult, TicketSearchResult, RepositorySearchResult } from "./types";

export function useCommandPaletteSearch(search: string): SearchResults {
  const [pods, setPods] = useState<PodSearchResult[]>([]);
  const [tickets, setTickets] = useState<TicketSearchResult[]>([]);
  const [repositories, setRepositories] = useState<RepositorySearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  const allRepos = useRepositories();
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  useEffect(() => { fetchRepositories(); }, [fetchRepositories]);

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
        const [podsRes, ticketsRes] = await Promise.all([
          getPodService().fetch_pods(null, null, null, null, null).then((j: string) => JSON.parse(j)).catch(() => ({ pods: [] })),
          getTicketService().fetch_tickets(undefined, 500, undefined).then((j: string) => JSON.parse(j)).catch(() => ({ tickets: [] })),
        ]);

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
          allRepos
            .filter((r) => r.slug.toLowerCase().includes(searchLower))
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
  }, [search, allRepos]);

  return { pods, tickets, repositories, loading };
}
