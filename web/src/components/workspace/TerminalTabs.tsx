"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
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
  const {
    panes,
    activePane,
    setActivePane,
    removePane,
    gridLayout,
    setGridLayout,
  } = useWorkspaceStore();

  const getConnectionStatus = (podKey: string) => {
    const status = terminalPool.getStatus(podKey);
    switch (status) {
      case "connected":
        return "bg-green-500";
      case "connecting":
        return "bg-yellow-500 animate-pulse";
      case "disconnected":
        return "bg-gray-500";
      case "error":
        return "bg-red-500";
      default:
        return "bg-gray-500";
    }
  };

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
            <Circle
              className={cn("w-2 h-2 flex-shrink-0", getConnectionStatus(pane.podKey))}
            />
            <span className="truncate">{pane.title}</span>
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

export default TerminalTabs;
