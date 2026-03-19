import { useState, useEffect, useMemo } from "react";
import { useAuthStore } from "@/stores/auth";
import { usePodStore } from "@/stores/pod";
import { organizationApi } from "@/lib/api/organization";
import { channelApi } from "@/lib/api/channel";
import { getPodDisplayName, getShortPodKey } from "@/lib/pod-utils";

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

interface ChannelPodRaw {
  pod_key: string;
  status: string;
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
  const allPods = usePodStore((s) => s.pods);
  const [members, setMembers] = useState<MentionItem[]>([]);
  const [rawChannelPods, setRawChannelPods] = useState<ChannelPodRaw[]>([]);
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

  // Fetch channel pods (running only) — store raw data
  useEffect(() => {
    if (!enabled || !channelId) {
      setRawChannelPods([]);
      return;
    }

    let cancelled = false;

    async function fetchPods() {
      try {
        setLoading(true);
        const response = await channelApi.getPods(channelId!);
        if (cancelled) return;

        const running = (response.pods || []).filter(
          (p) => p.status === "running" || p.status === "initializing"
        );
        setRawChannelPods(running.map((p) => ({ pod_key: p.pod_key, status: p.status })));
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

  // Derive pod mention items from raw data + pod store (reactive to alias changes)
  const pods: MentionItem[] = useMemo(
    () =>
      rawChannelPods.map((p) => {
        const storePod = allPods.find((sp) => sp.pod_key === p.pod_key);
        const displayName = storePod
          ? getPodDisplayName(storePod)
          : getShortPodKey(p.pod_key);
        return {
          id: `pod:${p.pod_key}`,
          type: "pod" as const,
          mentionText: getShortPodKey(p.pod_key),
          displayName,
          description: `Pod ${p.status}`,
        };
      }),
    [rawChannelPods, allPods]
  );

  const candidates = useMemo(
    () => [...members, ...pods],
    [members, pods]
  );

  return { candidates, members, pods, loading };
}
