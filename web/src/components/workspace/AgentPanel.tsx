"use client";

import React, { useCallback, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore, type SplitDirection } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { useAcpSessionStore } from "@/stores/acpSession";
import { usePodStatus } from "@/hooks";
import { useAcpRelay } from "@/hooks/useAcpRelay";
import { AgentPanelHeader } from "./AgentPanelHeader";
import {
  PaneLoadingState,
  PaneErrorState,
  PaneReconnectingState,
} from "./PaneStateViews";
import { AcpPlanTracker } from "./acp/AcpPlanTracker";
import { AcpActivityStream } from "./acp/AcpActivityStream";
import { AcpPermissionDialog } from "./acp/AcpPermissionDialog";
import { AcpPromptInput } from "./acp/AcpPromptInput";
import { PodSelectorModal } from "./PodSelectorModal";

// Re-export for backward compatibility (used by tests or other consumers)
export { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";

interface AgentPanelProps {
  paneId: string;
  podKey: string;
  isActive: boolean;
  onClose?: () => void;
  onMaximize?: () => void;
  onPopout?: () => void;
  showHeader?: boolean;
  className?: string;
}

export function AgentPanel({
  paneId,
  podKey,
  isActive,
  onClose,
  onMaximize,
  onPopout,
  showHeader = true,
  className,
}: AgentPanelProps) {
  const [isMaximized, setIsMaximized] = useState(false);
  const [isTerminating, setIsTerminating] = useState(false);
  const [pendingSplitDirection, setPendingSplitDirection] =
    useState<SplitDirection | null>(null);

  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const splitPane = useWorkspaceStore((s) => s.splitPane);
  const removePaneByPodKey = useWorkspaceStore((s) => s.removePaneByPodKey);
  const panes = useWorkspaceStore((s) => s.panes);
  const initProgress = usePodStore((state) => state.initProgress[podKey]);
  const terminatePod = usePodStore((state) => state.terminatePod);
  const session = useAcpSessionStore((s) => s.sessions[podKey]);

  const openPodKeys = useMemo(() => panes.map((p) => p.podKey), [panes]);
  const { podStatus, isPodReady, podError } = usePodStatus(podKey);

  // Subscribe to Relay for ACP messages when pod is ready
  const shouldSubscribe = isPodReady || podStatus === "running";
  useAcpRelay(podKey, paneId, shouldSubscribe);

  const handleFocus = useCallback(() => {
    setActivePane(paneId);
  }, [paneId, setActivePane]);

  const handleMaximize = useCallback(() => {
    setIsMaximized((prev) => !prev);
    onMaximize?.();
  }, [onMaximize]);

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

  return (
    <div
      className={cn(
        "flex flex-col h-full bg-background rounded-lg overflow-hidden border",
        isActive ? "border-primary" : "border-border",
        isMaximized && "fixed inset-4 z-50",
        className
      )}
      onClick={handleFocus}
    >
      {showHeader && (
        <AgentPanelHeader
          podKey={podKey}
          isMaximized={isMaximized}
          onPopout={onPopout}
          onSplitRight={() => setPendingSplitDirection("horizontal")}
          onSplitDown={() => setPendingSplitDirection("vertical")}
          onMaximize={handleMaximize}
          onClose={onClose}
        />
      )}

      {!shouldSubscribe ? (
        podError ? (
          <PaneErrorState error={podError} onClose={onClose} />
        ) : podStatus === "orphaned" ? (
          <PaneReconnectingState onClose={onClose} />
        ) : (
          <PaneLoadingState
            podStatus={podStatus}
            initProgress={initProgress}
            isTerminating={isTerminating}
            onTerminate={handleTerminate}
            onClose={onClose}
          />
        )
      ) : (
        <div className="flex flex-col flex-1 min-h-0">
          <AcpPlanTracker podKey={podKey} />
          <div className="flex-1 overflow-y-auto p-4">
            <AcpActivityStream podKey={podKey} />
          </div>
          {session?.pendingPermissions &&
            session.pendingPermissions.length > 0 && (
              <AcpPermissionDialog
                podKey={podKey}
                permissions={session.pendingPermissions}
              />
            )}
          <AcpPromptInput podKey={podKey} />
        </div>
      )}

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

export default AgentPanel;
