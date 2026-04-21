"use client";

import { usePods } from "@/stores/pod";
import { getPodDisplayName, getShortPodKey } from "@/lib/pod-display-name";

/**
 * Derives a display title for a pod from the pod store.
 * Falls back to truncated podKey when pod is not found.
 */
export function usePodTitle(podKey: string, fallback?: string): string {
  const pods = usePods();
  const pod = pods.find((p) => p.pod_key === podKey);
  if (pod) return getPodDisplayName(pod);
  return fallback ?? getShortPodKey(podKey);
}
