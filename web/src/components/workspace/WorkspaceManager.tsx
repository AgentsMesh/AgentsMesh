"use client";

import React, { useState, useEffect, useMemo } from "react";
import { cn } from "@/lib/utils";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { useWorkspaceStore } from "@/stores/workspace";
import { TerminalTabs } from "./TerminalTabs";
import { TerminalGrid } from "./TerminalGrid";
import { TerminalSwiper } from "./TerminalSwiper";
import { TerminalToolbar } from "./TerminalToolbar";
import { PodSelectorModal } from "./PodSelectorModal";

interface WorkspaceManagerProps {
  className?: string;
}

export function WorkspaceManager({ className }: WorkspaceManagerProps) {
  const { isMobile } = useBreakpoint();
  const panes = useWorkspaceStore((s) => s.panes);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const _hasHydrated = useWorkspaceStore((s) => s._hasHydrated);
  const [showPodSelector, setShowPodSelector] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Memoize to avoid creating a new array reference on every render
  const openPodKeys = useMemo(() => panes.map((p) => p.podKey), [panes]);

  // Handle adding new terminal
  const handleAddNew = () => {
    setShowPodSelector(true);
  };

  // Handle selecting a pod
  const handleSelectPod = (podKey: string) => {
    addPane(podKey);
    setShowPodSelector(false);
  };

  // Handle popout (desktop only)
  const handlePopout = (paneId: string) => {
    const pane = panes.find((p) => p.id === paneId);
    if (!pane) return;

    // Open in new window
    const popoutUrl = `/popout/terminal/${pane.podKey}`;
    const popoutWindow = window.open(
      popoutUrl,
      `terminal-${pane.podKey}`,
      "width=800,height=600,menubar=no,toolbar=no,location=no,status=no"
    );

    if (popoutWindow) {
      // Optionally remove from main workspace
      // removePane(paneId);
    }
  };

  // Toggle fullscreen
  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  };

  // Listen for fullscreen changes
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => {
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
    };
  }, []);

  if (!_hasHydrated) {
    return (
      <div className="h-full bg-terminal-bg">
        <CenteredSpinner />
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col h-full bg-terminal-bg", className)}>
      {/* Desktop layout */}
      {!isMobile && (
        <>
          {/* Tab bar */}
          <TerminalTabs
            onAddNew={handleAddNew}
            isFullscreen={isFullscreen}
            onToggleFullscreen={toggleFullscreen}
          />

          {/* Grid */}
          <TerminalGrid
            onPopout={handlePopout}
            onAddNew={handleAddNew}
            className="flex-1"
          />
        </>
      )}

      {/* Mobile layout */}
      {isMobile && (
        <>
          <TerminalSwiper onAddNew={handleAddNew} className="flex-1" />
          <TerminalToolbar />
        </>
      )}

      {/* Pod selector modal — subscribes to podStore only when open */}
      {showPodSelector && (
        <PodSelectorModal
          openPodKeys={openPodKeys}
          onSelect={handleSelectPod}
          onClose={() => setShowPodSelector(false)}
        />
      )}
    </div>
  );
}

export default WorkspaceManager;
