"use client";

import React, { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { getPodDisplayName } from "@/lib/pod-utils";
import { Button } from "@/components/ui/button";
import {
  X,
  Plus,
  Grid2X2,
  Rows,
  Columns,
  Square,
  Circle,
  Maximize2,
  Minimize2,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { terminalPool } from "@/stores/workspace";

interface TerminalTabsProps {
  onAddNew?: () => void;
  className?: string;
  isFullscreen?: boolean;
  onToggleFullscreen?: () => void;
}

export function TerminalTabs({ onAddNew, className, isFullscreen, onToggleFullscreen }: TerminalTabsProps) {
  const t = useTranslations();
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const removePane = useWorkspaceStore((s) => s.removePane);
  const gridLayout = useWorkspaceStore((s) => s.gridLayout);
  const setGridLayout = useWorkspaceStore((s) => s.setGridLayout);

  return (
    <div
      className={cn(
        "h-9 flex items-center bg-terminal-bg-secondary border-b border-terminal-border",
        className
      )}
    >
      {/* Tabs */}
      <div className="flex-1 flex items-center overflow-x-auto scrollbar-none">
        {panes.map((pane) => (
          <div
            key={pane.id}
            className={cn(
              "group flex items-center gap-1.5 px-3 h-9 text-sm cursor-pointer border-r border-terminal-border min-w-0 max-w-48",
              activePane === pane.id
                ? "bg-terminal-bg text-terminal-text-active"
                : "bg-terminal-bg-hover text-terminal-text-muted hover:bg-terminal-bg-active"
            )}
            onClick={() => setActivePane(pane.id)}
          >
            <ConnectionDot podKey={pane.podKey} />
            <span className="truncate"><TabPaneTitle podKey={pane.podKey} /></span>
            <button
              className={cn(
                "ml-1 p-0.5 rounded hover:bg-terminal-bg-active flex-shrink-0",
                "opacity-0 group-hover:opacity-100",
                activePane === pane.id && "opacity-100"
              )}
              onClick={(e) => {
                e.stopPropagation();
                removePane(pane.id);
              }}
            >
              <X className="w-3 h-3" />
            </button>
          </div>
        ))}

        {/* Add new tab button */}
        {onAddNew && (
          <Button
            variant="ghost"
            size="sm"
            className="h-9 px-3 rounded-none text-terminal-text-muted hover:text-terminal-text-active hover:bg-terminal-bg-active"
            onClick={onAddNew}
          >
            <Plus className="w-4 h-4" />
          </Button>
        )}
      </div>

      {/* Layout controls */}
      <div className="flex items-center gap-1 px-2 border-l border-terminal-border">
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active",
            gridLayout.type === "1x1" && "bg-terminal-bg-active text-terminal-text-active"
          )}
          onClick={() => setGridLayout({ type: "1x1", rows: 1, cols: 1 })}
          title={t("terminalTabs.singleView")}
        >
          <Square className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active",
            gridLayout.type === "1x2" && "bg-terminal-bg-active text-terminal-text-active"
          )}
          onClick={() => setGridLayout({ type: "1x2", rows: 1, cols: 2 })}
          title={t("terminalTabs.twoColumns")}
        >
          <Columns className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active",
            gridLayout.type === "2x1" && "bg-terminal-bg-active text-terminal-text-active"
          )}
          onClick={() => setGridLayout({ type: "2x1", rows: 2, cols: 1 })}
          title={t("terminalTabs.twoRows")}
        >
          <Rows className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active",
            gridLayout.type === "2x2" && "bg-terminal-bg-active text-terminal-text-active"
          )}
          onClick={() => setGridLayout({ type: "2x2", rows: 2, cols: 2 })}
          title={t("terminalTabs.grid2x2")}
        >
          <Grid2X2 className="w-3.5 h-3.5" />
        </Button>

        {/* Fullscreen toggle */}
        {onToggleFullscreen && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active ml-1"
            onClick={onToggleFullscreen}
            title={t("terminalTabs.fullscreen")}
          >
            {isFullscreen ? (
              <Minimize2 className="w-3.5 h-3.5" />
            ) : (
              <Maximize2 className="w-3.5 h-3.5" />
            )}
          </Button>
        )}
      </div>
    </div>
  );
}

/** Reactive connection status dot — subscribes to terminalPool status changes. */
function ConnectionDot({ podKey }: { podKey: string }) {
  const [statusClass, setStatusClass] = useState("bg-gray-500");

  useEffect(() => {
    const toClass = (s: string) => {
      switch (s) {
        case "connected": return "bg-green-500";
        case "connecting": return "bg-yellow-500 animate-pulse";
        case "error": return "bg-red-500";
        default: return "bg-gray-500";
      }
    };

    return terminalPool.onStatusChange(podKey, (info) => {
      setStatusClass(toClass(info.status));
    });
  }, [podKey]);

  return <Circle className={cn("w-2 h-2 flex-shrink-0", statusClass)} />;
}

/** Reads pod title from podStore — single source of truth. */
function TabPaneTitle({ podKey }: { podKey: string }) {
  const title = usePodStore((state) => {
    const pod = state.pods.find((p) => p.pod_key === podKey);
    return pod ? getPodDisplayName(pod) : `Pod ${podKey.substring(0, 8)}`;
  });
  return <>{title}</>;
}

export default TerminalTabs;
