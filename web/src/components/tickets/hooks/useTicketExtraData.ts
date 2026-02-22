"use client";

import { useState, useCallback, useEffect } from "react";
import { Ticket } from "@/stores/ticket";
import { ticketApi, TicketRelation, TicketCommit } from "@/lib/api";

/**
 * Extra data associated with a ticket (sub-tickets, relations, commits)
 */
export interface TicketExtraData {
  subTickets: Ticket[];
  relations: TicketRelation[];
  commits: TicketCommit[];
}

/**
 * Hook for fetching extra ticket data (sub-tickets, relations, commits)
 *
 * @param slug - Ticket slug
 * @param enabled - Whether to enable fetching (e.g., when ticket is loaded)
 */
export function useTicketExtraData(slug: string, enabled: boolean) {
  const [subTickets, setSubTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [commits, setCommits] = useState<TicketCommit[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchExtraData = useCallback(async () => {
    if (!enabled || !slug) return;

    setLoading(true);
    try {
      const [subTicketsRes, relationsRes, commitsRes] = await Promise.all([
        ticketApi.getSubTickets(slug).catch(() => ({ sub_tickets: [] })),
        ticketApi.listRelations(slug).catch(() => ({ relations: [] })),
        ticketApi.listCommits(slug).catch(() => ({ commits: [] })),
      ]);

      setSubTickets(subTicketsRes.sub_tickets || []);
      setRelations(relationsRes.relations || []);
      setCommits(commitsRes.commits || []);
    } catch (err) {
      console.error("Failed to fetch extra data:", err);
    } finally {
      setLoading(false);
    }
  }, [slug, enabled]);

  useEffect(() => {
    fetchExtraData();
  }, [fetchExtraData]);

  return {
    subTickets,
    relations,
    commits,
    loading,
    refetch: fetchExtraData,
  };
}

export default useTicketExtraData;
