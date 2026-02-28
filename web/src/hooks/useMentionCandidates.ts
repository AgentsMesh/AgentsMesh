import { useState, useEffect, useMemo } from "react";
import { useAuthStore } from "@/stores/auth";
import { organizationApi } from "@/lib/api/organization";
import { channelApi } from "@/lib/api/channel";

export interface MentionItem {
  /** Unique identifier: "user:id" or "pod:pod_key" */
  id: string;
  /** Mention type */
  type: "user" | "pod";
  /** Text inserted into message (e.g. username or pod_key short) */
  mentionText: string;
  /** Display name shown in dropdown */
  displayName: string;
  /** Secondary description in dropdown */
  description?: string;
  /** Avatar URL for users */
  avatarUrl?: string;
}

interface UseMentionCandidatesOptions {
  channelId: number | null;
  enabled?: boolean;
}

/**
 * Hook to fetch and merge mention candidates from organization members
 * and running channel pods.
 */
export function useMentionCandidates({
  channelId,
  enabled = true,
}: UseMentionCandidatesOptions) {
  const { currentOrg, user } = useAuthStore();
  const [members, setMembers] = useState<MentionItem[]>([]);
  const [pods, setPods] = useState<MentionItem[]>([]);
  const [loading, setLoading] = useState(false);

  // Fetch organization members
  const orgSlug = currentOrg?.slug;
  useEffect(() => {
    if (!enabled || !orgSlug) return;

    let cancelled = false;

    async function fetchMembers() {
      try {
        const response = await organizationApi.listMembers(orgSlug!);
        if (cancelled) return;

        const memberItems: MentionItem[] = (response.members || [])
          .filter((m) => m.user && m.user.id !== user?.id)
          .map((m) => ({
            id: `user:${m.user!.id}`,
            type: "user" as const,
            mentionText: m.user!.username,
            displayName: m.user!.name || m.user!.username,
            description: m.user!.email,
            avatarUrl: m.user!.avatar_url,
          }));

        setMembers(memberItems);
      } catch (error) {
        console.error("Failed to fetch members for mentions:", error);
      }
    }

    fetchMembers();

    return () => {
      cancelled = true;
    };
  }, [orgSlug, enabled, user?.id]);

  // Fetch channel pods (running only)
  useEffect(() => {
    if (!enabled || !channelId) {
      setPods([]);
      return;
    }

    let cancelled = false;

    async function fetchPods() {
      try {
        setLoading(true);
        const response = await channelApi.getPods(channelId!);
        if (cancelled) return;

        const runningPods = (response.pods || []).filter(
          (p) => p.status === "running" || p.status === "initializing"
        );

        const podItems: MentionItem[] = runningPods.map((p) => ({
          id: `pod:${p.pod_key}`,
          type: "pod" as const,
          mentionText: p.pod_key.slice(0, 8),
          displayName: p.pod_key.slice(0, 8),
          description: `Pod ${p.status}`,
        }));

        setPods(podItems);
      } catch (error) {
        console.error("Failed to fetch pods for mentions:", error);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchPods();

    return () => {
      cancelled = true;
    };
  }, [channelId, enabled]);

  const candidates = useMemo(
    () => [...members, ...pods],
    [members, pods]
  );

  return { candidates, members, pods, loading };
}
