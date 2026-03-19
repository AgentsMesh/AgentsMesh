"use client";

import { useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Terminal, ExternalLink, Square, Pencil } from "lucide-react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  ConfirmDialog,
  useConfirmDialog,
} from "@/components/ui/confirm-dialog";
import { RenameDialog } from "@/components/shared/RenameDialog";
import { podApi } from "@/lib/api";
import type { MeshNode } from "@/stores/mesh";
import { useMeshStore } from "@/stores/mesh";
import { usePodStore } from "@/stores/pod";
import { useWorkspaceStore } from "@/stores/workspace";

interface PodContextMenuProps {
  node: MeshNode;
  children: React.ReactNode;
}

export default function PodContextMenu({ node, children }: PodContextMenuProps) {
  const t = useTranslations("mesh");
  const params = useParams();
  const router = useRouter();
  const orgSlug = params.org as string;
  const { fetchTopology } = useMeshStore();
  const updatePodAlias = usePodStore((s) => s.updatePodAlias);
  const removePaneByPodKey = useWorkspaceStore((s) => s.removePaneByPodKey);
  const { dialogProps, confirm } = useConfirmDialog();
  const [renameOpen, setRenameOpen] = useState(false);

  const isActive = node.status === "running" || node.status === "initializing";

  const handleOpenTerminal = useCallback(() => {
    router.push(`/${orgSlug}/workspace?pod=${node.pod_key}`);
  }, [router, orgSlug, node.pod_key]);

  const handleViewTicket = useCallback(() => {
    if (node.ticket_slug) {
      router.push(`/${orgSlug}/tickets/${node.ticket_slug}`);
    }
  }, [router, orgSlug, node.ticket_slug]);

  const handleTerminate = useCallback(async () => {
    const confirmed = await confirm({
      title: t("contextMenu.terminateTitle"),
      description: t("contextMenu.terminateDescription"),
      confirmText: t("contextMenu.terminateConfirm"),
      variant: "destructive",
    });
    if (confirmed) {
      await podApi.terminate(node.pod_key);
      removePaneByPodKey(node.pod_key);
      fetchTopology();
    }
  }, [confirm, t, node.pod_key, removePaneByPodKey, fetchTopology]);

  const handleRenameConfirm = useCallback(
    async (newName: string) => {
      try {
        await updatePodAlias(node.pod_key, newName || null);
        fetchTopology();
      } catch (error) {
        console.error("Failed to rename pod:", error);
      }
    },
    [node.pod_key, updatePodAlias, fetchTopology]
  );

  return (
    <>
      <ContextMenu>
        <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
        <ContextMenuContent className="w-56">
          <ContextMenuItem
            onClick={handleOpenTerminal}
            disabled={!isActive}
          >
            <Terminal className="mr-2 h-4 w-4" />
            {t("contextMenu.openTerminal")}
          </ContextMenuItem>

          <ContextMenuItem onClick={() => setRenameOpen(true)}>
            <Pencil className="mr-2 h-4 w-4" />
            {t("contextMenu.rename")}
          </ContextMenuItem>

          {node.ticket_slug && (
            <ContextMenuItem onClick={handleViewTicket}>
              <ExternalLink className="mr-2 h-4 w-4" />
              {t("contextMenu.viewTicket", {
                slug: node.ticket_slug,
              })}
            </ContextMenuItem>
          )}

          <ContextMenuSeparator />

          <ContextMenuItem
            onClick={handleTerminate}
            disabled={!isActive}
            className="text-destructive focus:text-destructive"
          >
            <Square className="mr-2 h-4 w-4" />
            {t("contextMenu.terminatePod")}
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      <RenameDialog
        open={renameOpen}
        onOpenChange={setRenameOpen}
        currentName={node.alias || ""}
        onConfirm={handleRenameConfirm}
      />
      <ConfirmDialog {...dialogProps} />
    </>
  );
}
