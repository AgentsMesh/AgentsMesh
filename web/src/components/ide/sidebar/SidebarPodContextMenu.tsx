"use client";

import { useTranslations } from "next-intl";
import { Pencil, Square, RefreshCw } from "lucide-react";
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
  onTogglePerpetual: (perpetual: boolean) => void;
  children: React.ReactNode;
}

export function SidebarPodContextMenu({
  pod,
  onRename,
  onTerminate,
  onTogglePerpetual,
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

        {isActive && (
          <ContextMenuItem onClick={() => onTogglePerpetual(!pod.perpetual)}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {pod.perpetual
              ? t("contextMenu.disablePerpetual")
              : t("contextMenu.enablePerpetual")}
          </ContextMenuItem>
        )}

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
