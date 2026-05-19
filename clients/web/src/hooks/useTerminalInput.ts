"use client";

import { useCallback } from "react";
import { useWorkspaceStore, relayPool, terminalRegistry } from "@/stores/workspace";

export function useTerminalInput() {
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);

  const activePodKey = panes.find((p) => p.id === activePane)?.podKey ?? null;

  const send = useCallback(
    (data: string) => {
      if (activePodKey) relayPool.send(activePodKey, data);
    },
    [activePodKey],
  );

  const scrollToBottom = useCallback(() => {
    if (activePodKey) terminalRegistry.scrollToBottom(activePodKey);
  }, [activePodKey]);

  const syncSize = useCallback(() => {
    if (!activePodKey) return;
    const term = terminalRegistry.get(activePodKey);
    if (term && term.cols > 0 && term.rows > 0) {
      relayPool.forceResize(activePodKey, term.cols, term.rows);
    }
  }, [activePodKey]);

  return { activePodKey, send, scrollToBottom, syncSize };
}
