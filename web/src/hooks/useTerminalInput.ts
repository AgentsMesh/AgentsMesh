"use client";

import { useCallback } from "react";
import { useWorkspaceStore, relayPool, terminalRegistry } from "@/stores/workspace";

/**
 * Encapsulates terminal input operations (send data, scroll to bottom)
 * for the currently active pane.
 *
 * Eliminates direct relayPool/terminalRegistry coupling in UI components.
 */
export function useTerminalInput() {
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);

  const activePodKey = panes.find((p) => p.id === activePane)?.podKey ?? null;

  /** Send raw data to the active terminal's pod. */
  const send = useCallback(
    (data: string) => {
      if (activePodKey) relayPool.send(activePodKey, data);
    },
    [activePodKey],
  );

  /** Scroll the active terminal to the bottom. */
  const scrollToBottom = useCallback(() => {
    if (activePodKey) terminalRegistry.scrollToBottom(activePodKey);
  }, [activePodKey]);

  /** Force-sync terminal size to pod using real xterm dimensions. */
  const syncSize = useCallback(() => {
    if (!activePodKey) return;
    const term = terminalRegistry.get(activePodKey);
    if (term && term.cols > 0 && term.rows > 0) {
      relayPool.forceResize(activePodKey, term.cols, term.rows);
    }
  }, [activePodKey]);

  return { activePodKey, send, scrollToBottom, syncSize };
}
