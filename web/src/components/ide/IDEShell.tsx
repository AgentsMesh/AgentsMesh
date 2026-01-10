"use client";

import React, { useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import { ActivityBar } from "./ActivityBar";
import { SideBar } from "./SideBar";
import { BottomPanel } from "./BottomPanel";
import { CommandPalette } from "./CommandPalette";
import { CreatePodModal } from "./CreatePodModal";
import { WorkspaceSidebarContent } from "./sidebar/WorkspaceSidebarContent";
import { TicketsSidebarContent } from "./sidebar/TicketsSidebarContent";
import { RepositoriesSidebarContent } from "./sidebar/RepositoriesSidebarContent";
import { RunnersSidebarContent } from "./sidebar/RunnersSidebarContent";
import { MeshSidebarContent } from "./sidebar/MeshSidebarContent";
import { SettingsSidebarContent } from "./sidebar/SettingsSidebarContent";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { toast } from "sonner";

interface IDEShellProps {
  children: React.ReactNode;
  sidebarContent?: React.ReactNode;
  className?: string;
}

/**
 * IDEShell - Desktop IDE-style layout
 *
 * Layout structure:
 * ┌──────────┬──────────────┬─────────────────────────────────┐
 * │ Activity │  Side Bar    │       Main Content Area         │
 * │   Bar    │  (resizable) │                                 │
 * │  (48px)  │              │                                 │
 * │          │              ├─────────────────────────────────┤
 * │          │              │       Bottom Panel              │
 * └──────────┴──────────────┴─────────────────────────────────┘
 */
// Get sidebar content based on current activity
function getSidebarContent(
  activity: ActivityType,
  onCreatePod?: () => void
): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent onCreatePod={onCreatePod} />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
    case "repositories":
      return <RepositoriesSidebarContent />;
    case "runners":
      return <RunnersSidebarContent />;
    case "settings":
      return <SettingsSidebarContent />;
    default:
      return null;
  }
}

export function IDEShell({
  children,
  sidebarContent,
  className,
}: IDEShellProps) {
  const { bottomPanelOpen, activeActivity, _hasHydrated } = useIDEStore();
  const { addPane } = useWorkspaceStore();
  const { fetchPods } = usePodStore();
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createPodModalOpen, setCreatePodModalOpen] = useState(false);

  // Handle pod creation
  const handleCreatePod = useCallback(() => {
    setCreatePodModalOpen(true);
  }, []);

  const handlePodCreated = useCallback((pod?: { pod_key: string }) => {
    setCreatePodModalOpen(false);
    if (pod?.pod_key) {
      toast.info("Pod created! Waiting for it to start...", {
        description: `Pod: ${pod.pod_key.substring(0, 8)}`,
      });
      addPane(pod.pod_key, `Pod ${pod.pod_key.substring(0, 8)}`);
      fetchPods();
    }
  }, [addPane, fetchPods]);

  // Use provided sidebar content or auto-generate based on activity
  const effectiveSidebarContent = sidebarContent ?? getSidebarContent(activeActivity, handleCreatePod);

  // Show loading state while hydrating to prevent flash
  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className={cn("flex h-screen bg-background overflow-hidden", className)}>
      {/* Activity Bar - fixed width */}
      <ActivityBar className="flex-shrink-0" />

      {/* Side Bar - resizable */}
      <SideBar className="flex-shrink-0">{effectiveSidebarContent}</SideBar>

      {/* Main area - flexible */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Main content */}
        <main
          className={cn(
            "flex-1 overflow-auto",
            bottomPanelOpen ? "" : "pb-8" // Space for collapsed bottom panel
          )}
        >
          {children}
        </main>

        {/* Bottom Panel */}
        <BottomPanel />
      </div>

      {/* Command Palette */}
      <CommandPalette
        open={commandPaletteOpen}
        onOpenChange={setCommandPaletteOpen}
      />

      {/* Create Pod Modal */}
      <CreatePodModal
        open={createPodModalOpen}
        onClose={() => setCreatePodModalOpen(false)}
        onCreated={handlePodCreated}
      />
    </div>
  );
}

export default IDEShell;
