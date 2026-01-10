"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { Button } from "@/components/ui/button";
import {
  X,
  Plus,
  Grid2X2,
  LayoutGrid,
  Rows,
  Columns,
  Square,
  Circle,
} from "lucide-react";
import { terminalPool } from "@/stores/workspace";

interface TerminalTabsProps {
  onAddNew?: () => void;
  className?: string;
}

export function TerminalTabs({ onAddNew, className }: TerminalTabsProps) {
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
        "h-9 flex items-center bg-[#252526] border-b border-[#3c3c3c]",
        className
      )}
    >
      {/* Tabs */}
      <div className="flex-1 flex items-center overflow-x-auto scrollbar-none">
        {panes.map((pane) => (
          <div
            key={pane.id}
            className={cn(
              "group flex items-center gap-1.5 px-3 h-9 text-sm cursor-pointer border-r border-[#3c3c3c] min-w-0 max-w-48",
              activePane === pane.id
                ? "bg-[#1e1e1e] text-[#ffffff]"
                : "bg-[#2d2d2d] text-[#969696] hover:bg-[#2a2a2a]"
            )}
            onClick={() => setActivePane(pane.id)}
          >
            <Circle
              className={cn("w-2 h-2 flex-shrink-0", getConnectionStatus(pane.podKey))}
            />
            <span className="truncate">{pane.title}</span>
            <button
              className={cn(
                "ml-1 p-0.5 rounded hover:bg-[#3c3c3c] flex-shrink-0",
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
            className="h-9 px-3 rounded-none text-[#969696] hover:text-[#ffffff] hover:bg-[#2a2a2a]"
            onClick={onAddNew}
          >
            <Plus className="w-4 h-4" />
          </Button>
        )}
      </div>

      {/* Layout controls */}
      <div className="flex items-center gap-1 px-2 border-l border-[#3c3c3c]">
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-[#969696] hover:text-[#ffffff]",
            gridLayout.type === "1x1" && "bg-[#3c3c3c] text-[#ffffff]"
          )}
          onClick={() => setGridLayout({ type: "1x1", rows: 1, cols: 1 })}
          title="Single view"
        >
          <Square className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-[#969696] hover:text-[#ffffff]",
            gridLayout.type === "1x2" && "bg-[#3c3c3c] text-[#ffffff]"
          )}
          onClick={() => setGridLayout({ type: "1x2", rows: 1, cols: 2 })}
          title="Two columns"
        >
          <Columns className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-[#969696] hover:text-[#ffffff]",
            gridLayout.type === "2x1" && "bg-[#3c3c3c] text-[#ffffff]"
          )}
          onClick={() => setGridLayout({ type: "2x1", rows: 2, cols: 1 })}
          title="Two rows"
        >
          <Rows className="w-3.5 h-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-6 w-6 p-0 text-[#969696] hover:text-[#ffffff]",
            gridLayout.type === "2x2" && "bg-[#3c3c3c] text-[#ffffff]"
          )}
          onClick={() => setGridLayout({ type: "2x2", rows: 2, cols: 2 })}
          title="2x2 grid"
        >
          <Grid2X2 className="w-3.5 h-3.5" />
        </Button>
      </div>
    </div>
  );
}

export default TerminalTabs;
