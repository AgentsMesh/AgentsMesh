"use client";

import { useState, useEffect } from "react";
import { terminalPool, type RelayStatusInfo } from "@/stores/terminalConnection";

/**
 * Subscribes to terminal connection status changes for a pod.
 * Wraps terminalPool.onStatusChange() to eliminate direct
 * singleton coupling in UI components.
 */
export function useTerminalStatus(podKey: string): RelayStatusInfo {
  const [status, setStatus] = useState<RelayStatusInfo>({
    status: "none",
    runnerDisconnected: false,
  });

  useEffect(() => {
    return terminalPool.onStatusChange(podKey, setStatus);
  }, [podKey]);

  return status;
}
