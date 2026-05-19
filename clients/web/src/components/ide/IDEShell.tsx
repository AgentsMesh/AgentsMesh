"use client";

import React, { useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import { useCtaModal } from "@/hooks/useCtaModal";
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
import { InfraSidebarContent } from "./sidebar/InfraSidebarContent";
import { MeshSidebarContent } from "./sidebar/MeshSidebarContent";
import { ChannelsSidebarContent } from "./sidebar/ChannelsSidebarContent";
import { LoopsSidebarContent } from "./sidebar/LoopsSidebarContent";
import { SettingsSidebarContent } from "./sidebar/SettingsSidebarContent";
import { BlocksSidebar } from "@/components/blocks/BlocksSidebar";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { toast } from "sonner";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { AddRunnerModal } from "./modals/AddRunnerModal";
import { ImportRepositoryModal } from "./modals/ImportRepositoryModal";

interface IDEShellProps {
  children: React.ReactNode;
  sidebarContent?: React.ReactNode;
  className?: string;
}

interface SidebarCallbacks {
  onCreatePod?: () => void;
  onAddRunner?: () => void;
  onImportRepo?: () => void;
}

function getSidebarContent(
  activity: ActivityType,
  callbacks: SidebarCallbacks
): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent onCreatePod={callbacks.onCreatePod} />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "channels":
      return <ChannelsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
    case "loops":
      return <LoopsSidebarContent />;
    case "blocks":
      return <BlocksSidebar />;
    case "infra":
      return (
        <InfraSidebarContent
          onImportRepo={callbacks.onImportRepo}
          onAddRunner={callbacks.onAddRunner}
        />
      );
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
  const bottomPanelOpen = useIDEStore((state) => state.bottomPanelOpen);
  const activeActivity = useIDEStore((state) => state.activeActivity);
  const _hasHydrated = useIDEStore((state) => state._hasHydrated);
  const addPane = useWorkspaceStore((state) => state.addPane);
  const fetchPods = usePodStore((state) => state.fetchPods);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createPodModalOpen, setCreatePodModalOpen] = useState(false);
  const addRunnerModal = useCtaModal();
  const importRepoModal = useCtaModal();

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

  const sidebarCallbacks: SidebarCallbacks = {
    onCreatePod: handleCreatePod,
    onAddRunner: addRunnerModal.open,
    onImportRepo: importRepoModal.open,
  };
  const effectiveSidebarContent = sidebarContent ?? getSidebarContent(activeActivity, sidebarCallbacks);

  if (!_hasHydrated) {
    return (
      <div className="h-screen bg-background">
        <CenteredSpinner />
      </div>
    );
  }

  return (
    <div className={cn("app-shell flex h-screen bg-background overflow-hidden", className)}>
      <ActivityBar className="flex-shrink-0" />

      <SideBar className="flex-shrink-0">{effectiveSidebarContent}</SideBar>

      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <main
          className={cn(
            "flex-1 overflow-auto",
            activeActivity === "workspace" && bottomPanelOpen ? "" : "pb-8"
          )}
        >
          {children}
        </main>

        {activeActivity === "workspace" && <BottomPanel />}
      </div>

      <CommandPalette
        open={commandPaletteOpen}
        onOpenChange={setCommandPaletteOpen}
      />

      <CreatePodModal
        open={createPodModalOpen}
        onClose={() => setCreatePodModalOpen(false)}
        onCreated={handlePodCreated}
      />

      <AddRunnerModal
        open={addRunnerModal.isOpen}
        onClose={addRunnerModal.close}
        onCreated={addRunnerModal.commit}
      />

      <ImportRepositoryModal
        open={importRepoModal.isOpen}
        onClose={importRepoModal.close}
        onImported={importRepoModal.commit}
      />
    </div>
  );
}

export default IDEShell;
