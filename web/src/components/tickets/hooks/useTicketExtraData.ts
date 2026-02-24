"use client";

import { useState, useCallback, useEffect } from "react";
import { Ticket } from "@/stores/ticket";
import { ticketApi, TicketRelation, TicketCommit, TicketComment } from "@/lib/api";

/**
 * Extra data associated with a ticket (sub-tickets, relations, commits, comments)
 */
export interface TicketExtraData {
  subTickets: Ticket[];
  relations: TicketRelation[];
  commits: TicketCommit[];
  comments: TicketComment[];
}

/**
 * Hook for fetching extra ticket data (sub-tickets, relations, commits, comments)
 *
 * @param slug - Ticket slug
 * @param enabled - Whether to enable fetching (e.g., when ticket is loaded)
 */
export function useTicketExtraData(slug: string, enabled: boolean) {
  const [subTickets, setSubTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [commits, setCommits] = useState<TicketCommit[]>([]);
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchExtraData = useCallback(async () => {
    if (!enabled || !slug) return;

    setLoading(true);
    try {
      const [subTicketsRes, relationsRes, commitsRes, commentsRes] = await Promise.all([
        ticketApi.getSubTickets(slug).catch(() => ({ sub_tickets: [] })),
        ticketApi.listRelations(slug).catch(() => ({ relations: [] })),
        ticketApi.listCommits(slug).catch(() => ({ commits: [] })),
        ticketApi.listComments(slug).catch(() => ({ comments: [], total: 0 })),
      ]);

      setSubTickets(subTicketsRes.sub_tickets || []);
      setRelations(relationsRes.relations || []);
      setCommits(commitsRes.commits || []);
      setComments(commentsRes.comments || []);
    } catch (err) {
      console.error("Failed to fetch extra data:", err);
    } finally {
      setLoading(false);
    }
  }, [slug, enabled]);

  useEffect(() => {
    fetchExtraData();
  }, [fetchExtraData]);

  // Comment CRUD operations
  const addComment = useCallback(
    async (
      content: string,
      parentId?: number,
      mentions?: Array<{ user_id: number; username: string }>
    ) => {
      await ticketApi.createComment(slug, content, parentId, mentions);
      // Refetch to get updated list with replies properly nested
      const commentsRes = await ticketApi.listComments(slug).catch(() => ({ comments: [], total: 0 }));
      setComments(commentsRes.comments || []);
    },
    [slug]
  );

  const updateComment = useCallback(
    async (
      commentId: number,
      content: string,
      mentions?: Array<{ user_id: number; username: string }>
    ) => {
      await ticketApi.updateComment(slug, commentId, content, mentions);
      const commentsRes = await ticketApi.listComments(slug).catch(() => ({ comments: [], total: 0 }));
      setComments(commentsRes.comments || []);
    },
    [slug]
  );

  const deleteComment = useCallback(
    async (commentId: number) => {
      await ticketApi.deleteComment(slug, commentId);
      const commentsRes = await ticketApi.listComments(slug).catch(() => ({ comments: [], total: 0 }));
      setComments(commentsRes.comments || []);
    },
    [slug]
  );

  return {
    subTickets,
    relations,
    commits,
    comments,
    loading,
    refetch: fetchExtraData,
    addComment,
    updateComment,
    deleteComment,
  };
}

export default useTicketExtraData;
