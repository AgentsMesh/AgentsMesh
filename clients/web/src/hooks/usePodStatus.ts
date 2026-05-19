"use client";

import { useEffect, useRef, useMemo, useState } from "react";
import { usePod, usePodStore } from "@/stores/pod";
import { ApiError } from "@/lib/api/api-types";

interface UsePodStatusResult {
  podStatus: string;
  isPodReady: boolean;
  podError: string | null;
}

const MAX_FETCH_RETRIES = 3;

export function usePodStatus(podKey: string): UsePodStatusResult {
  const initialFetchDone = useRef(false);
  const retryCount = useRef(0);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const storePod = usePod(podKey);
  const fetchPod = usePodStore((state) => state.fetchPod);

  const { podStatus, isPodReady, podError } = useMemo(() => {
    const storeStatus = storePod?.status;
    if (!storeStatus && fetchError) {
      return { podStatus: "error", isPodReady: false, podError: fetchError };
    }

    const status = storeStatus ?? "unknown";
    const isReady = status === "running";

    let error: string | null = null;
    if (status === "failed") {
      error = "Pod failed";
    } else if (status === "terminated") {
      error = "Pod terminated";
    } else if (status === "error") {
      error = storePod?.error_message || "Pod error";
    }
    // "orphaned" is loading/reconnecting (Runner restart auto-recovers), not error.

    return { podStatus: status, isPodReady: isReady, podError: error };
  }, [storePod?.status, storePod?.error_message, fetchError]);

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
          initialFetchDone.current = true;
          setFetchError("Pod not found");
        } else {
          setFetchError("Failed to load pod");
        }
      });
  }, [podKey, fetchPod, storePod]);

  return { podStatus, isPodReady, podError };
}
