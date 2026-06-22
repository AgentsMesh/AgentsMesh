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
import { getSidebarContent, type SidebarCallbacks } from "./sidebarContentResolver";
import { useIDEStore } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { toast } from "sonner";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { AddRunnerModal } from "./modals/AddRunnerModal";
import { ImportRepositoryModal } from "./modals/ImportRepositoryModal";
import { CreateChannelDialog } from "@/components/channel";
import { useChannelStore } from "@/stores/channel";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { useTranslations } from "next-intl";

interface IDEShellProps {
  children: React.ReactNode;
  sidebarContent?: React.ReactNode;
  className?: string;
}

export function IDEShell({
  children,
  sidebarContent,
  className,
}: IDEShellProps) {
  const t = useTranslations();
  const bottomPanelOpen = useIDEStore((state) => state.bottomPanelOpen);
  const activeActivity = useIDEStore((state) => state.activeActivity);
  const _hasHydrated = useIDEStore((state) => state._hasHydrated);
  const addPane = useWorkspaceStore((state) => state.addPane);
  const fetchPods = usePodStore((state) => state.fetchPods);
  const setSelectedChannelId = useChannelStore((state) => state.setSelectedChannelId);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createPodModalOpen, setCreatePodModalOpen] = useState(false);
  const [createChannelOpen, setCreateChannelOpen] = useState(false);
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

  const handleChannelCreated = useCallback(
    (channelId: number) => {
      setCreateChannelOpen(false);
      setSelectedChannelId(channelId);
    },
    [setSelectedChannelId],
  );

  const sidebarCallbacks: SidebarCallbacks = {
    onCreatePod: handleCreatePod,
    onAddRunner: addRunnerModal.open,
    onImportRepo: importRepoModal.open,
  };
  const effectiveSidebarContent = sidebarContent ?? getSidebarContent(activeActivity, sidebarCallbacks);
  const sidebarHeaderAction =
    activeActivity === "channels" ? (
      <Button
        variant="ghost"
        size="sm"
        className="h-7 w-7 flex-shrink-0 p-0"
        onClick={() => setCreateChannelOpen(true)}
        aria-label={t("channels.sidebar.createChannel")}
      >
        <Plus className="h-4 w-4" />
      </Button>
    ) : undefined;

  if (!_hasHydrated) {
    return (
      <div className="h-screen bg-background">
        <CenteredSpinner />
      </div>
    );
  }

  return (
    <div className={cn("app-shell flex flex-col h-screen bg-sidebar overflow-hidden", className)}>
      <div
        className="app-drag shrink-0"
        style={{ height: "var(--titlebar-drag-height)" }}
        aria-hidden="true"
      />

      <div className="flex min-h-0 flex-1">
        <ActivityBar className="flex-shrink-0" />

        <div className="flex min-w-0 flex-1 gap-2 p-2">
          <SideBar className="flex-shrink-0" headerAction={sidebarHeaderAction}>{effectiveSidebarContent}</SideBar>

          <div className="flex-1 flex flex-col min-w-0 overflow-hidden rounded-xl border border-border/50 bg-background shadow-sm">
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
        </div>
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

      <CreateChannelDialog
        open={createChannelOpen}
        onOpenChange={setCreateChannelOpen}
        onCreated={handleChannelCreated}
      />
    </div>
  );
}

export default IDEShell;
