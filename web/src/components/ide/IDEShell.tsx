"use client";

import React, { useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import { CenteredSpinner } from "@/components/ui/spinner";
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
import { LoopsSidebarContent } from "./sidebar/LoopsSidebarContent";
import { SettingsSidebarContent } from "./sidebar/SettingsSidebarContent";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { toast } from "sonner";
import { getPodDisplayName } from "@/lib/pod-utils";
import { AddRunnerModal } from "./modals/AddRunnerModal";
import { ImportRepositoryModal } from "./modals/ImportRepositoryModal";

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
interface SidebarCallbacks {
  onCreatePod?: () => void;
  onAddRunner?: () => void;
  onImportRepo?: () => void;
}

// Get sidebar content based on current activity
function getSidebarContent(
  activity: ActivityType,
  callbacks: SidebarCallbacks
): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent onCreatePod={callbacks.onCreatePod} />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
    case "loops":
      return <LoopsSidebarContent />;
    case "repositories":
      return <RepositoriesSidebarContent onImportRepo={callbacks.onImportRepo} />;
    case "runners":
      return <RunnersSidebarContent onAddRunner={callbacks.onAddRunner} />;
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
  // Use selectors to only subscribe to specific state slices
  // This prevents unnecessary re-renders when unrelated state changes
  const bottomPanelOpen = useIDEStore((state) => state.bottomPanelOpen);
  const activeActivity = useIDEStore((state) => state.activeActivity);
  const _hasHydrated = useIDEStore((state) => state._hasHydrated);
  const addPane = useWorkspaceStore((state) => state.addPane);
  // Use selector to only subscribe to fetchPods action, not the entire pods array
  // This prevents re-renders when pod titles or statuses change
  const fetchPods = usePodStore((state) => state.fetchPods);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createPodModalOpen, setCreatePodModalOpen] = useState(false);
  const [addRunnerModalOpen, setAddRunnerModalOpen] = useState(false);
  const [importRepoModalOpen, setImportRepoModalOpen] = useState(false);

  // Handle pod creation
  const handleCreatePod = useCallback(() => {
    setCreatePodModalOpen(true);
  }, []);

  const handlePodCreated = useCallback((pod?: { pod_key: string; title?: string }) => {
    setCreatePodModalOpen(false);
    if (pod?.pod_key) {
      const displayName = getPodDisplayName(pod);
      toast.info("Pod created! Waiting for it to start...", {
        description: `Pod: ${displayName}`,
      });
      addPane(pod.pod_key);
      fetchPods();
    }
  }, [addPane, fetchPods]);

  // Handle add runner
  const handleAddRunner = useCallback(() => {
    setAddRunnerModalOpen(true);
  }, []);

  // Handle import repository
  const handleImportRepo = useCallback(() => {
    setImportRepoModalOpen(true);
  }, []);

  // Use provided sidebar content or auto-generate based on activity
  const sidebarCallbacks: SidebarCallbacks = {
    onCreatePod: handleCreatePod,
    onAddRunner: handleAddRunner,
    onImportRepo: handleImportRepo,
  };
  const effectiveSidebarContent = sidebarContent ?? getSidebarContent(activeActivity, sidebarCallbacks);

  // Show loading state while hydrating to prevent flash
  if (!_hasHydrated) {
    return (
      <div className="h-screen bg-background">
        <CenteredSpinner />
      </div>
    );
  }

  return (
    <div className={cn("app-shell flex h-screen bg-background overflow-hidden", className)}>
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
            activeActivity === "workspace" && bottomPanelOpen ? "" : "pb-8" // Space for collapsed bottom panel (only on workspace)
          )}
        >
          {children}
        </main>

        {/* Bottom Panel - only visible on workspace */}
        {activeActivity === "workspace" && <BottomPanel />}
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

      {/* Add Runner Modal */}
      <AddRunnerModal
        open={addRunnerModalOpen}
        onClose={() => setAddRunnerModalOpen(false)}
        onCreated={() => setAddRunnerModalOpen(false)}
      />

      {/* Import Repository Modal */}
      <ImportRepositoryModal
        open={importRepoModalOpen}
        onClose={() => setImportRepoModalOpen(false)}
        onImported={() => setImportRepoModalOpen(false)}
      />
    </div>
  );
}

export default IDEShell;
