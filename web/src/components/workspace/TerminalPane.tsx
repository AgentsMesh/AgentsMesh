"use client";

import React, { useCallback, useEffect, useState, useRef, useMemo } from "react";
import "@xterm/xterm/css/xterm.css";
import { RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore, type SplitDirection } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { useAutopilotStore } from "@/stores/autopilot";
import { usePodStatus, useTerminal, useTouchScroll } from "@/hooks";
import { TerminalPaneHeader } from "./TerminalPaneHeader";
import { TerminalLoadingState, TerminalErrorState, TerminalReconnectingState } from "./TerminalStateViews";
import { RelayStatusOverlay } from "./RelayStatusOverlay";
import { AutopilotOverlay } from "./AutopilotOverlay";
import { AutopilotStartButton } from "./AutopilotStartButton";
import { PodSelectorModal } from "./PodSelectorModal";

interface TerminalPaneProps {
  paneId: string;
  podKey: string;
  isActive: boolean;
  onClose?: () => void;
  onMaximize?: () => void;
  onPopout?: () => void;
  showHeader?: boolean;
  className?: string;
}

export function TerminalPane({
  paneId,
  podKey,
  isActive,
  onClose,
  onMaximize,
  onPopout,
  showHeader = true,
  className,
}: TerminalPaneProps) {
  const [isMaximized, setIsMaximized] = useState(false);
  const [isTerminating, setIsTerminating] = useState(false);
  const [pendingSplitDirection, setPendingSplitDirection] = useState<SplitDirection | null>(null);
  const triggerAutopilotRef = useRef<(() => void) | null>(null);
  const maximizeRafRef = useRef<number | undefined>(undefined);
  const terminalFontSize = useWorkspaceStore((s) => s.terminalFontSize);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const splitPane = useWorkspaceStore((s) => s.splitPane);
  const removePaneByPodKey = useWorkspaceStore((s) => s.removePaneByPodKey);
  const panes = useWorkspaceStore((s) => s.panes);
  const initProgress = usePodStore((state) => state.initProgress[podKey]);
  const terminatePod = usePodStore((state) => state.terminatePod);
  const hasAutopilot = useAutopilotStore((state) => !!state.getAutopilotControllerByPodKey(podKey));

  const openPodKeys = useMemo(() => panes.map((p) => p.podKey), [panes]);

  // Pod status tracking
  const { podStatus, isPodReady, podError } = usePodStatus(podKey);

  // "Sticky ready" flag: once the terminal has been shown, don't unmount it
  // due to transient status changes (e.g., stale WebSocket events causing
  // status to temporarily revert to "initializing").
  const wasEverReady = useRef(false);
  if (isPodReady) {
    wasEverReady.current = true;
  }
  const showTerminal = wasEverReady.current;

  // Terminal initialization and management
  const {
    terminalRef,
    xtermRef,
    connectionStatus,
    isRunnerDisconnected,
    syncSize,
  } = useTerminal(podKey, terminalFontSize, showTerminal, isActive);

  // Mobile touch scrolling support
  useTouchScroll(terminalRef, xtermRef, showTerminal);

  const handleFocus = useCallback(() => {
    setActivePane(paneId);
  }, [paneId, setActivePane]);

  const handleMaximize = useCallback(() => {
    setIsMaximized((prev) => !prev);
    onMaximize?.();
    // ResizeObserver in useTerminal will auto-fit after layout change.
    // Use syncSize as a fallback to ensure PTY size is updated.
    if (maximizeRafRef.current !== undefined) cancelAnimationFrame(maximizeRafRef.current);
    maximizeRafRef.current = requestAnimationFrame(() => {
      maximizeRafRef.current = undefined;
      syncSize();
    });
  }, [onMaximize, syncSize]);

  const handleTerminate = useCallback(async () => {
    setIsTerminating(true);
    try {
      await terminatePod(podKey);
      removePaneByPodKey(podKey);
    } catch (error) {
      console.error("Failed to terminate pod:", error);
    } finally {
      setIsTerminating(false);
    }
  }, [podKey, terminatePod, removePaneByPodKey]);

  // Cancel pending maximize RAF on unmount
  useEffect(() => {
    return () => {
      if (maximizeRafRef.current !== undefined) cancelAnimationFrame(maximizeRafRef.current);
    };
  }, []);

  return (
    <div
      className={cn(
        "flex flex-col h-full bg-terminal-bg rounded-lg overflow-hidden border",
        isActive ? "border-primary" : "border-border",
        isMaximized && "fixed inset-4 z-50",
        className
      )}
      onClick={handleFocus}
    >
      {/* Header */}
      {showHeader && (
        <TerminalPaneHeader
          podKey={podKey}
          connectionStatus={connectionStatus}
          isMaximized={isMaximized}
          isPodReady={isPodReady}
          hasAutopilot={hasAutopilot}
          onSyncSize={syncSize}
          onStartAutopilot={() => triggerAutopilotRef.current?.()}
          onPopout={onPopout}
          onSplitRight={() => setPendingSplitDirection("horizontal")}
          onSplitDown={() => setPendingSplitDirection("vertical")}
          onMaximize={handleMaximize}
          onClose={onClose}
        />
      )}

      {/* Terminal or Loading/Error/Reconnecting State */}
      {!showTerminal ? (
        podError ? (
          <TerminalErrorState error={podError} onClose={onClose} />
        ) : podStatus === "orphaned" ? (
          <TerminalReconnectingState onClose={onClose} />
        ) : (
          <TerminalLoadingState
            podStatus={podStatus}
            initProgress={initProgress}
            onClose={onClose}
          />
        )
      ) : (
        <div className="flex flex-col flex-1 min-h-0">
          <AutopilotOverlay podKey={podKey} />
          <div className="relative flex-1 min-h-0">
            {/* Reconnecting overlay - shown when pod is orphaned but terminal was previously active */}
            {podStatus === "orphaned" && (
              <div className="absolute inset-0 z-10 flex items-center justify-center bg-terminal-bg/80 backdrop-blur-sm">
                <div className="text-center p-4">
                  <RefreshCw className="w-8 h-8 text-amber-500 dark:text-amber-400 mx-auto mb-2 animate-spin" />
                  <p className="text-terminal-text font-medium text-sm">
                    Runner is restarting...
                  </p>
                  <p className="text-xs text-terminal-text-muted">
                    Session will resume automatically
                  </p>
                </div>
              </div>
            )}
            {/* Relay connection status overlay - always visible, floating at top */}
            <RelayStatusOverlay
              connectionStatus={connectionStatus}
              isRunnerDisconnected={isRunnerDisconnected}
            />
            <div
              ref={terminalRef}
              className="h-full overflow-auto"
              style={{
                touchAction: "pan-y pinch-zoom", // Enable touch scrolling and zoom
              }}
            />
          </div>
        </div>
      )}

      {/* Autopilot modal (managed by AutopilotStartButton) */}
      <AutopilotStartButton podKey={podKey} triggerRef={triggerAutopilotRef} />

      {/* Pod selector for split */}
      {pendingSplitDirection && (
        <PodSelectorModal
          openPodKeys={openPodKeys}
          onSelect={(selectedPodKey) => {
            splitPane(paneId, pendingSplitDirection, selectedPodKey);
            setPendingSplitDirection(null);
          }}
          onClose={() => setPendingSplitDirection(null)}
        />
      )}
    </div>
  );
}

export default TerminalPane;
