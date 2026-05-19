"use client";

import { useState, useEffect } from "react";
import { relayPool, type RelayStatusInfo } from "@/stores/relayConnection";

export function useTerminalStatus(podKey: string): RelayStatusInfo {
  const [status, setStatus] = useState<RelayStatusInfo>({
    status: "none",
    runnerDisconnected: false,
  });

  useEffect(() => {
    return relayPool.onStatusChange(podKey, setStatus);
  }, [podKey]);

  return status;
}
