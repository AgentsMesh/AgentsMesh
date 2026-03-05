"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { useWorkspaceStore } from "@/stores/workspace";
import { WorkspaceManager } from "@/components/workspace";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { Terminal, Plus } from "lucide-react";
import { useTranslations } from "next-intl";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import type { PodData } from "@/lib/api";

export default function WorkspacePage() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const router = useRouter();
  const panes = useWorkspaceStore((s) => s.panes);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const _hasHydrated = useWorkspaceStore((s) => s._hasHydrated);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const processedPodRef = useRef<string | null>(null);

  const handleOpenPod = useCallback((podKey: string) => {
    addPane(podKey);
  }, [addPane]);

  // Handle ?pod=xxx query param to auto-open a pod
  useEffect(() => {
    if (!_hasHydrated) return;

    const podKey = searchParams.get("pod");
    if (podKey && podKey !== processedPodRef.current) {
      processedPodRef.current = podKey;
      // Check if pod is already open
      const isAlreadyOpen = panes.some((p) => p.podKey === podKey);
      if (!isAlreadyOpen) {
        handleOpenPod(podKey);
        toast.info(t("workspace.podOpened"), {
          description: `Pod: ${podKey.substring(0, 8)}`,
        });
      }
      // Clear the query param to avoid re-opening on refresh
      router.replace(window.location.pathname);
    }
  }, [_hasHydrated, searchParams, panes, router, t, handleOpenPod]);

  // Show loading while hydrating
  if (!_hasHydrated) {
    return <CenteredSpinner />;
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
