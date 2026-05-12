"use client";

import { useState, useCallback, useEffect } from "react";
import { Ticket } from "@/stores/ticket";
import { TicketRelation, TicketCommit, TicketComment } from "@/lib/api";
import { getTicketService } from "@/lib/wasm-core";
import {
  listRelations,
  listCommits,
  listComments,
  createComment as createCommentConnect,
  updateComment as updateCommentConnect,
  deleteComment as deleteCommentConnect,
} from "@/lib/api/ticketRelations";
import { useCurrentOrg } from "@/stores/auth";

export interface TicketExtraData {
  subTickets: Ticket[];
  relations: TicketRelation[];
  commits: TicketCommit[];
  comments: TicketComment[];
}

// useTicketExtraData fetches sub-tickets, relations, commits, comments for
// a ticket. Relations / commits / comments go through the Connect-RPC
// adapter (proto.ticket_relations.v1) — sub-tickets remain on the legacy
// ticket service until that migration lands.
export function useTicketExtraData(slug: string, enabled: boolean) {
  const orgSlug = useCurrentOrg()?.slug || "";
  const [subTickets, setSubTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [commits, setCommits] = useState<TicketCommit[]>([]);
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchExtraData = useCallback(async () => {
    if (!enabled || !slug || !orgSlug) return;

    setLoading(true);
    try {
      const [subTicketsRes, relationsRes, commitsRes, commentsRes] = await Promise.all([
        getTicketService().get_sub_tickets(slug).then((j: string) => JSON.parse(j)).catch(() => ({ sub_tickets: [] })),
        listRelations(orgSlug, slug).catch(() => ({ relations: [] })),
        listCommits(orgSlug, slug).catch(() => ({ commits: [] })),
        listComments(orgSlug, slug).catch(() => ({ comments: [], total: 0, limit: 0, offset: 0 })),
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
  }, [slug, enabled, orgSlug]);

  useEffect(() => {
    fetchExtraData();
  }, [fetchExtraData]);

  const addComment = useCallback(
    async (
      content: string,
      parentId?: number,
      mentions?: Array<{ user_id: number; username: string }>
    ) => {
      await createCommentConnect(orgSlug, slug, { content, parent_id: parentId, mentions });
      const refreshed = await listComments(orgSlug, slug);
      setComments(refreshed.comments);
    },
    [slug, orgSlug]
  );

  const updateComment = useCallback(
    async (
      commentId: number,
      content: string,
      mentions?: Array<{ user_id: number; username: string }>
    ) => {
      await updateCommentConnect(orgSlug, slug, commentId, { content, mentions });
      const refreshed = await listComments(orgSlug, slug);
      setComments(refreshed.comments);
    },
    [slug, orgSlug]
  );

  const deleteComment = useCallback(
    async (commentId: number) => {
      await deleteCommentConnect(orgSlug, slug, commentId);
      const refreshed = await listComments(orgSlug, slug);
      setComments(refreshed.comments);
    },
    [slug, orgSlug]
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
