"use client";

import { useEffect, useRef, useMemo } from "react";
import { usePodStore } from "@/stores/pod";

interface UsePodStatusResult {
  podStatus: string;
  isPodReady: boolean;
  podError: string | null;
}

/**
 * Hook for tracking pod readiness status
 * Uses realtime events via store - only fetches once on mount for initial state
 */
export function usePodStatus(podKey: string): UsePodStatusResult {
  const initialFetchDone = useRef(false);
  const { pods, fetchPod } = usePodStore();

  // Get pod from store (updated via realtime events)
  const storePod = pods.find((p) => p.pod_key === podKey);

  // Derive status from store - no local state needed
  const { podStatus, isPodReady, podError } = useMemo(() => {
    const status = storePod?.status ?? "unknown";
    const isReady = status === "running";

    // Terminal states that indicate pod cannot be used
    let error: string | null = null;
    if (status === "failed") {
      error = "Pod failed";
    } else if (status === "terminated") {
      error = "Pod terminated";
    } else if (status === "orphaned") {
      error = "Pod orphaned - Runner connection lost";
    } else if (status === "error") {
      error = "Pod error";
    }

    return { podStatus: status, isPodReady: isReady, podError: error };
  }, [storePod?.status]);

  // Initial status fetch (once only) - updates store via fetchPod
  useEffect(() => {
    if (initialFetchDone.current || storePod) return;
    initialFetchDone.current = true;

    fetchPod(podKey).catch((error) => {
      console.error("Failed to fetch initial pod status:", error);
    });
  }, [podKey, fetchPod, storePod]);

  return { podStatus, isPodReady, podError };
}
