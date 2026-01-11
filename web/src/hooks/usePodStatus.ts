"use client";

import { useEffect, useState, useRef } from "react";
import { usePodStore } from "@/stores/pod";
import { podApi } from "@/lib/api/client";

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
  const [podStatus, setPodStatus] = useState<string>("unknown");
  const [isPodReady, setIsPodReady] = useState(false);
  const [podError, setPodError] = useState<string | null>(null);
  const initialFetchDone = useRef(false);

  // Get pod from store (updated via realtime events)
  const { pods } = usePodStore();
  const storePod = pods.find((p) => p.pod_key === podKey);

  // Initial status fetch (once only)
  useEffect(() => {
    if (initialFetchDone.current) return;
    initialFetchDone.current = true;

    let mounted = true;

    const fetchInitialStatus = async () => {
      try {
        const { pod } = await podApi.get(podKey);
        if (!mounted) return;

        setPodStatus(pod.status);

        if (pod.status === "running") {
          setIsPodReady(true);
          setPodError(null);
        } else if (pod.status === "failed" || pod.status === "terminated") {
          setIsPodReady(false);
          setPodError(`Pod ${pod.status}`);
        }
        // For "initializing" or "paused", realtime events will update
      } catch (error) {
        console.error("Failed to fetch initial pod status:", error);
      }
    };

    fetchInitialStatus();

    return () => {
      mounted = false;
    };
  }, [podKey]);

  // React to store updates from realtime events
  useEffect(() => {
    if (!storePod) return;

    setPodStatus(storePod.status);

    if (storePod.status === "running") {
      setIsPodReady(true);
      setPodError(null);
    } else if (storePod.status === "failed" || storePod.status === "terminated") {
      setIsPodReady(false);
      setPodError(`Pod ${storePod.status}`);
    }
  }, [storePod?.status]);

  return { podStatus, isPodReady, podError };
}
