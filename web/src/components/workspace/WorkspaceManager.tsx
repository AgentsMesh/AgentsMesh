"use client";

import React, { useState, useEffect, useMemo } from "react";
import { cn } from "@/lib/utils";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { TerminalTabs } from "./TerminalTabs";
import { TerminalGrid } from "./TerminalGrid";
import { TerminalSwiper } from "./TerminalSwiper";
import { TerminalToolbar } from "./TerminalToolbar";
import { Button } from "@/components/ui/button";
import { getPodDisplayName } from "@/lib/pod-utils";

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

interface PodSelectorModalProps {
  openPodKeys: string[];
  onSelect: (podKey: string) => void;
  onClose: () => void;
}

function PodSelectorModal({ openPodKeys, onSelect, onClose }: PodSelectorModalProps) {
  // Subscribe to raw pods array — filter in useMemo to avoid creating
  // a new array reference inside the selector (causes infinite re-render loop).
  const allPods = usePodStore((s) => s.pods);
  const pods = useMemo(
    () => allPods.filter(
      (pod) => pod.status === "running" && !openPodKeys.includes(pod.pod_key)
    ),
    [allPods, openPodKeys]
  );

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md max-h-[80vh] overflow-hidden">
        <div className="p-4 border-b border-border">
          <h2 className="text-lg font-semibold">Select a Pod</h2>
          <p className="text-sm text-muted-foreground">
            Choose a running pod to open a terminal
          </p>
        </div>

        <div className="overflow-y-auto max-h-96">
          {pods.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">
              <p>No running pods available</p>
              <p className="text-sm mt-1">Create a pod to start a terminal</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {pods.map((pod) => (
                <button
                  key={pod.pod_key}
                  className="w-full p-4 text-left hover:bg-muted transition-colors"
                  onClick={() => onSelect(pod.pod_key)}
                >
                  <div className="flex items-center justify-between">
                    <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">
                      {getPodDisplayName(pod)}
                    </code>
                    <span className="text-xs text-green-500 dark:text-green-400">{pod.status}</span>
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    <span>Agent: {pod.agent_status}</span>
                    {pod.runner && (
                      <span className="ml-2">• Runner: {pod.runner.node_id}</span>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="p-4 border-t border-border">
          <Button variant="outline" className="w-full" onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}

export default WorkspaceManager;
