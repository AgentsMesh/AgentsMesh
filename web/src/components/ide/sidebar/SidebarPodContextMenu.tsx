"use client";

import { useTranslations } from "next-intl";
import { Pencil, Square } from "lucide-react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import type { Pod } from "@/stores/pod";

interface SidebarPodContextMenuProps {
  pod: Pod;
  onRename: () => void;
  onTerminate: () => void;
  children: React.ReactNode;
}

export function SidebarPodContextMenu({
  pod,
  onRename,
  onTerminate,
  children,
}: SidebarPodContextMenuProps) {
  const t = useTranslations("workspace");
  const isActive = pod.status === "running" || pod.status === "initializing";

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="w-48">
        <ContextMenuItem onClick={onRename}>
          <Pencil className="mr-2 h-4 w-4" />
          {t("contextMenu.rename")}
        </ContextMenuItem>

        <ContextMenuSeparator />

        <ContextMenuItem
          onClick={onTerminate}
          disabled={!isActive}
          className="text-destructive focus:text-destructive"
        >
          <Square className="mr-2 h-4 w-4" />
          {t("contextMenu.terminate")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
