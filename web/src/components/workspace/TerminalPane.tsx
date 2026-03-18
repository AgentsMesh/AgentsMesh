"use client";

import React, { useCallback, useState, useRef } from "react";
import "@xterm/xterm/css/xterm.css";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { useAutopilotStore } from "@/stores/autopilot";
import { usePodStatus, useTerminal, useTouchScroll } from "@/hooks";
import { TerminalPaneHeader } from "./TerminalPaneHeader";
import { TerminalLoadingState, TerminalErrorState } from "./TerminalStateViews";
import { RelayStatusOverlay } from "./RelayStatusOverlay";
import { AutopilotOverlay } from "./AutopilotOverlay";
import { AutopilotStartButton } from "./AutopilotStartButton";

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
  const triggerAutopilotRef = useRef<(() => void) | null>(null);
  const terminalFontSize = useWorkspaceStore((s) => s.terminalFontSize);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const splitPane = useWorkspaceStore((s) => s.splitPane);
  const initProgress = usePodStore((state) => state.initProgress[podKey]);
  const terminatePod = usePodStore((state) => state.terminatePod);
  const hasAutopilot = useAutopilotStore((state) => !!state.getAutopilotControllerByPodKey(podKey));

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
    requestAnimationFrame(() => syncSize());
  }, [onMaximize, syncSize]);

  const handleTerminate = useCallback(async () => {
    setIsTerminating(true);
    try {
      await terminatePod(podKey);
      onClose?.();
    } catch (error) {
      console.error("Failed to terminate pod:", error);
    } finally {
      setIsTerminating(false);
    }
  }, [podKey, terminatePod, onClose]);

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
          onSplitRight={() => splitPane(paneId, "horizontal")}
          onSplitDown={() => splitPane(paneId, "vertical")}
          onMaximize={handleMaximize}
          onClose={onClose}
        />
      )}

      {/* Terminal or Loading/Error State */}
      {!showTerminal ? (
        podError ? (
          <TerminalErrorState error={podError} onClose={onClose} />
        ) : (
          <TerminalLoadingState
            podStatus={podStatus}
            initProgress={initProgress}
            isTerminating={isTerminating}
            onTerminate={handleTerminate}
            onClose={onClose}
          />
        )
      ) : (
        <div className="flex flex-col flex-1 min-h-0">
          <AutopilotOverlay podKey={podKey} />
          <div className="relative flex-1 min-h-0">
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
    </div>
  );
}

export default TerminalPane;
