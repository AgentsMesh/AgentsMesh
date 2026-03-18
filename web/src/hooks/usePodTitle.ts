"use client";

import { usePodStore } from "@/stores/pod";
import { getPodDisplayName } from "@/lib/pod-utils";

/**
 * Derives a display title for a pod from the pod store.
 * Falls back to truncated podKey when pod is not found.
 *
 * Eliminates duplicate pod title derivation across
 * TerminalTabs, TerminalSwiper, and AutopilotStartButton.
 */
export function usePodTitle(podKey: string, fallback?: string): string {
  return usePodStore((state) => {
    const pod = state.pods.find((p) => p.pod_key === podKey);
    if (pod) return getPodDisplayName(pod);
    return fallback ?? podKey.substring(0, 8);
  });
}
