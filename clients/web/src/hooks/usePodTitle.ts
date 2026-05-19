"use client";

import { usePods } from "@/stores/pod";
import { getPodDisplayName, getShortPodKey } from "@/lib/pod-display-name";

export function usePodTitle(podKey: string, fallback?: string): string {
  const pods = usePods();
  const pod = pods.find((p) => p.pod_key === podKey);
  if (pod) return getPodDisplayName(pod);
  return fallback ?? getShortPodKey(podKey);
}
