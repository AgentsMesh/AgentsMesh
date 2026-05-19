"use client";

import React, { useState, useCallback } from "react";
import { Drawer } from "vaul";
import * as VisuallyHidden from "@radix-ui/react-visually-hidden";
import { cn } from "@/lib/utils";
import { useCtaModal } from "@/hooks/useCtaModal";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { getPodDisplayName } from "@/lib/pod-display-name";

import { WorkspaceSidebarContent } from "@/components/ide/sidebar/WorkspaceSidebarContent";
import { TicketsSidebarContent } from "@/components/ide/sidebar/TicketsSidebarContent";
import { MeshSidebarContent } from "@/components/ide/sidebar/MeshSidebarContent";
import { RepositoriesSidebarContent } from "@/components/ide/sidebar/RepositoriesSidebarContent";
import { RunnersSidebarContent } from "@/components/ide/sidebar/RunnersSidebarContent";
import { SettingsSidebarContent } from "@/components/ide/sidebar/SettingsSidebarContent";

import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { AddRunnerModal } from "@/components/ide/modals/AddRunnerModal";
import { ImportRepositoryModal } from "@/components/ide/modals/ImportRepositoryModal/index";

interface MobileSidebarProps {
  className?: string;
}

function getActivityTitle(activity: ActivityType): string {
  switch (activity) {
    case "workspace":
      return "Workspace";
    case "tickets":
      return "Tickets";
    case "mesh":
      return "Mesh";
    case "repositories":
      return "Repositories";
    case "runners":
      return "Runners";
    case "settings":
      return "Settings";
    default:
      return "Mesh";
  }
}

interface SidebarCallbacks {
  onCreatePod?: () => void;
  onAddRunner?: () => void;
  onImportRepo?: () => void;
  onTerminatePod?: () => void;
}

function getSidebarContent(
  activity: ActivityType,
  callbacks: SidebarCallbacks
): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent onCreatePod={callbacks.onCreatePod} onTerminatePod={callbacks.onTerminatePod} />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
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

export function MobileSidebar({ className }: MobileSidebarProps) {
  const activeActivity = useIDEStore((s) => s.activeActivity);
  const mobileSidebarOpen = useIDEStore((s) => s.mobileSidebarOpen);
  const setMobileSidebarOpen = useIDEStore((s) => s.setMobileSidebarOpen);
  const currentOrg = useCurrentOrg();
  const addPane = useWorkspaceStore((s) => s.addPane);
  const fetchPods = usePodStore((s) => s.fetchPods);

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

  const handleTerminatePod = useCallback(() => {
    setMobileSidebarOpen(false);
  }, [setMobileSidebarOpen]);

  // Guard: prevent drawer from closing when a nested dialog is open
  const handleDrawerOpenChange = useCallback((open: boolean) => {
    if (!open && document.querySelector('[data-dialog-overlay]')) {
      return;
    }
    setMobileSidebarOpen(open);
  }, [setMobileSidebarOpen]);

  const title = getActivityTitle(activeActivity);
  const sidebarCallbacks: SidebarCallbacks = {
    onCreatePod: handleCreatePod,
    onAddRunner: addRunnerModal.open,
    onImportRepo: importRepoModal.open,
    onTerminatePod: handleTerminatePod,
  };
  const content = getSidebarContent(activeActivity, sidebarCallbacks);

  return (
    <Drawer.Root
      open={mobileSidebarOpen}
      onOpenChange={handleDrawerOpenChange}
      direction="right"
    >
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40 z-50" />
        <Drawer.Content
          className={cn(
            "fixed right-0 top-0 bottom-0 w-[300px] bg-background z-50 flex flex-col",
            className
          )}
          aria-describedby={undefined}
        >
          {/* Hidden title for accessibility */}
          <VisuallyHidden.Root>
            <Drawer.Title>{title} Panel</Drawer.Title>
          </VisuallyHidden.Root>

          {/* Header */}
          <div className="h-14 flex items-center justify-between px-4 border-b border-border">
            <div className="flex items-center gap-2 min-w-0">
              {currentOrg?.name && (
                <div className="w-6 h-6 rounded bg-primary/10 flex items-center justify-center text-xs font-medium text-primary flex-shrink-0">
                  {currentOrg.name.charAt(0).toUpperCase()}
                </div>
              )}
              <span className="font-semibold truncate">{title}</span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="p-2 flex-shrink-0"
              onClick={() => setMobileSidebarOpen(false)}
            >
              <X className="w-5 h-5" />
            </Button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto">
            {content}
          </div>
        </Drawer.Content>
      </Drawer.Portal>

      {/* Modals */}
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
    </Drawer.Root>
  );
}

export default MobileSidebar;
