"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useWorkspaceStore } from "@/stores/workspace";
import { WorkspaceManager } from "@/components/workspace";
import { Button } from "@/components/ui/button";
import { Terminal, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n/client";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { PodData } from "@/lib/api";

export default function WorkspacePage() {
  const t = useTranslations();
  const { panes, addPane, _hasHydrated } = useWorkspaceStore();
  const [showCreateModal, setShowCreateModal] = useState(false);

  const handleOpenPod = (podKey: string, title?: string) => {
    addPane(podKey, title || `Pod ${podKey.substring(0, 8)}`);
  };

  // Show loading while hydrating
  if (!_hasHydrated) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  // Empty state when no terminals are open
  if (panes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8">
        <Terminal className="w-16 h-16 mb-4 text-muted-foreground/30" />
        <h2 className="text-xl font-semibold mb-2">{t("workspace.noTerminalsOpen")}</h2>
        <p className="text-muted-foreground text-center mb-6 max-w-md">
          {t("workspace.noTerminalsDescription")}
        </p>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="w-4 h-4 mr-2" />
          {t("workspace.createNewPod")}
        </Button>

        {/* Create Modal */}
        <CreatePodModal
          open={showCreateModal}
          onClose={() => setShowCreateModal(false)}
          onCreated={(pod?: PodData) => {
            setShowCreateModal(false);
            if (pod?.pod_key) {
              toast.info(t("workspace.podCreated"), {
                description: `Pod: ${pod.pod_key.substring(0, 8)}`,
              });
              handleOpenPod(pod.pod_key);
            }
          }}
        />
      </div>
    );
  }

  // Terminal workspace
  return (
    <div className="flex flex-col h-full">
      <WorkspaceManager className="flex-1" />

      {/* Create Modal */}
      <CreatePodModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onCreated={(pod?: PodData) => {
          setShowCreateModal(false);
          if (pod?.pod_key) {
            toast.info(t("workspace.podCreated"), {
              description: `Pod: ${pod.pod_key.substring(0, 8)}`,
            });
            handleOpenPod(pod.pod_key);
          }
        }}
      />
    </div>
  );
}
