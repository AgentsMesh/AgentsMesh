"use client";

import { useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Terminal, ExternalLink, Square } from "lucide-react";
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
import { podApi } from "@/lib/api";
import type { MeshNode } from "@/stores/mesh";
import { useMeshStore } from "@/stores/mesh";

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
  const { dialogProps, confirm } = useConfirmDialog();

  const isActive = node.status === "running" || node.status === "initializing";

  const handleOpenTerminal = useCallback(() => {
    router.push(`/${orgSlug}/workspace?pod=${node.pod_key}`);
  }, [router, orgSlug, node.pod_key]);

  const handleViewTicket = useCallback(() => {
    if (node.ticket_identifier) {
      router.push(`/${orgSlug}/tickets/${node.ticket_identifier}`);
    }
  }, [router, orgSlug, node.ticket_identifier]);

  const handleTerminate = useCallback(async () => {
    const confirmed = await confirm({
      title: t("contextMenu.terminateTitle"),
      description: t("contextMenu.terminateDescription"),
      confirmText: t("contextMenu.terminateConfirm"),
      variant: "destructive",
    });
    if (confirmed) {
      await podApi.terminate(node.pod_key);
      fetchTopology();
    }
  }, [confirm, t, node.pod_key, fetchTopology]);

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

          {node.ticket_identifier && (
            <ContextMenuItem onClick={handleViewTicket}>
              <ExternalLink className="mr-2 h-4 w-4" />
              {t("contextMenu.viewTicket", {
                identifier: node.ticket_identifier,
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
      <ConfirmDialog {...dialogProps} />
    </>
  );
}
