"use client";

import { useEffect, useRef, useMemo, useState } from "react";
import { usePodStore } from "@/stores/pod";
import { ApiError } from "@/lib/api";

interface UsePodStatusResult {
  podStatus: string;
  isPodReady: boolean;
  podError: string | null;
}

const MAX_FETCH_RETRIES = 3;

/**
 * Hook for tracking pod readiness status
 * Uses realtime events via store - only fetches once on mount for initial state
 */
export function usePodStatus(podKey: string): UsePodStatusResult {
  const initialFetchDone = useRef(false);
  const retryCount = useRef(0);
  const [fetchError, setFetchError] = useState<string | null>(null);

  // Granular selectors — only re-render when THIS pod changes or fetchPod ref changes
  const storePod = usePodStore((state) => state.pods.find((p) => p.pod_key === podKey));
  const fetchPod = usePodStore((state) => state.fetchPod);

  // Derive status from store - no local state needed
  const { podStatus, isPodReady, podError } = useMemo(() => {
    // Live store data (from WS events) always takes priority over initial fetch error.
    // This prevents a transient network error from permanently shadowing
    // a pod that later becomes available via realtime updates.
    const storeStatus = storePod?.status;
    if (!storeStatus && fetchError) {
      return { podStatus: "error", isPodReady: false, podError: fetchError };
    }

    const status = storeStatus ?? "unknown";
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
      error = storePod?.error_message || "Pod error";
    }

    return { podStatus: status, isPodReady: isReady, podError: error };
  }, [storePod?.status, storePod?.error_message, fetchError]);

  // Initial status fetch — only runs once on mount (or retries on transient failure).
  // Sets initialFetchDone only on success or deterministic 404 to allow
  // automatic retry when storePod appears via WS or component remounts.
  // Capped at MAX_FETCH_RETRIES to prevent runaway loops.
  useEffect(() => {
    if (initialFetchDone.current || storePod) return;
    if (retryCount.current >= MAX_FETCH_RETRIES) return;

    retryCount.current++;
    fetchPod(podKey)
      .then(() => {
        initialFetchDone.current = true;
      })
      .catch((error) => {
        if (error instanceof ApiError && error.status === 404) {
          initialFetchDone.current = true; // deterministic — no retry
          setFetchError("Pod not found");
        } else {
          // Transient error — leave initialFetchDone false so next render retries
          // (up to MAX_FETCH_RETRIES)
          setFetchError("Failed to load pod");
        }
      });
  }, [podKey, fetchPod, storePod]);

  return { podStatus, isPodReady, podError };
}
