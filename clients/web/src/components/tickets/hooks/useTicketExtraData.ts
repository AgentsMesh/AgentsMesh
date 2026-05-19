"use client";

import { useState, useCallback, useEffect } from "react";
import { Ticket } from "@/stores/ticket";
import { TicketRelation, TicketCommit, TicketComment } from "@/lib/api";
import { getTicketRelationsService, getTicketService } from "@/lib/wasm-core";

export interface TicketExtraData {
  subTickets: Ticket[];
  relations: TicketRelation[];
  commits: TicketCommit[];
  comments: TicketComment[];
}

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
        getTicketService().get_sub_tickets(slug).then((j: string) => JSON.parse(j)).catch(() => ({ sub_tickets: [] })),
        getTicketRelationsService().list_relations(slug).then((j: string) => JSON.parse(j)).catch(() => ({ relations: [] })),
        getTicketRelationsService().list_commits(slug).then((j: string) => JSON.parse(j)).catch(() => ({ commits: [] })),
        getTicketRelationsService().list_comments(slug).then((j: string) => JSON.parse(j)).catch(() => ({ comments: [], total: 0 })),
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

  const addComment = useCallback(
    async (
      content: string,
      parentId?: number,
      mentions?: Array<{ user_id: number; username: string }>
    ) => {
      await getTicketRelationsService().create_comment(slug, JSON.stringify({ content, parent_id: parentId, mentions }));
      const commentsRes = JSON.parse(await getTicketRelationsService().list_comments(slug));
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
      await getTicketRelationsService().update_comment(slug, BigInt(commentId), JSON.stringify({ content, mentions }));
      const commentsRes = JSON.parse(await getTicketRelationsService().list_comments(slug));
      setComments(commentsRes.comments || []);
    },
    [slug]
  );

  const deleteComment = useCallback(
    async (commentId: number) => {
      await getTicketRelationsService().delete_comment(slug, BigInt(commentId));
      const commentsRes = JSON.parse(await getTicketRelationsService().list_comments(slug));
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
