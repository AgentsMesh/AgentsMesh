"use client";

import { useEffect, useState, useCallback } from "react";
import { ticketApi } from "@/lib/api";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";

export function useTicketPaneData(slug: string) {
  const updateTicket = useTicketStore((s) => s.updateTicket);
  const updateTicketStatus = useTicketStore((s) => s.updateTicketStatus);
  const tickets = useTicketStore((s) => s.tickets);

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!slug) return;
    const cachedTicket = tickets.find(t => t.slug === slug);
    if (cachedTicket) setTicket(cachedTicket);
    else setTicket(null);

    if (tickets.length > 0 && !cachedTicket) return;

    const loadTicket = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await ticketApi.get(slug);
        setTicket(data);
      } catch (err: unknown) {
        console.error("Failed to load ticket:", err);
        setError(err instanceof Error ? err.message : "Failed to load ticket");
      } finally { setLoading(false); }
    };
    loadTicket();
  }, [slug, tickets]);

  const handleStatusChange = useCallback(async (newStatus: TicketStatus) => {
    if (!ticket) return;
    const oldStatus = ticket.status;
    setTicket({ ...ticket, status: newStatus });
    try { await updateTicketStatus(slug, newStatus); }
    catch (err: unknown) { console.error("Failed to update status:", err); setTicket({ ...ticket, status: oldStatus }); throw err; }
  }, [ticket, slug, updateTicketStatus]);

  const handleTitleChange = useCallback(async (newTitle: string) => {
    if (!ticket || !newTitle.trim()) return;
    const oldTitle = ticket.title;
    setTicket({ ...ticket, title: newTitle });
    try { await updateTicket(slug, { title: newTitle }); }
    catch (err: unknown) { console.error("Failed to update title:", err); setTicket({ ...ticket, title: oldTitle }); throw err; }
  }, [ticket, slug, updateTicket]);

  const handleRepositoryChange = useCallback(async (newRepositoryId: number | null) => {
    if (!ticket) return;
    const oldRepositoryId = ticket.repository_id;
    const oldRepository = ticket.repository;
    setTicket({ ...ticket, repository_id: newRepositoryId ?? undefined, repository: undefined });
    try {
      const updated = await updateTicket(slug, { repositoryId: newRepositoryId });
      setTicket(updated);
    } catch (err: unknown) {
      console.error("Failed to update repository:", err);
      setTicket({ ...ticket, repository_id: oldRepositoryId, repository: oldRepository });
      throw err;
    }
  }, [ticket, slug, updateTicket]);

  return { ticket, loading, error, handleStatusChange, handleTitleChange, handleRepositoryChange };
}
