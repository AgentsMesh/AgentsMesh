"use client";

import React, { useMemo } from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore, type TerminalPane as TerminalPaneType } from "@/stores/workspace";
import { TerminalPane } from "./TerminalPane";
import { Terminal as TerminalIcon, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TerminalGridProps {
  onPopout?: (paneId: string) => void;
  onAddNew?: () => void;
  className?: string;
}

export function TerminalGrid({ onPopout, onAddNew, className }: TerminalGridProps) {
  const { panes, activePane, gridLayout, removePane } = useWorkspaceStore();

  // Calculate visible panes based on grid layout
  const visiblePanes = useMemo(() => {
    const maxVisible = gridLayout.rows * gridLayout.cols;

    // If we have an active pane, make sure it's included
    if (activePane) {
      const activeIndex = panes.findIndex((p) => p.id === activePane);
      if (activeIndex !== -1) {
        // Get panes around the active one
        const startIndex = Math.max(0, activeIndex - Math.floor(maxVisible / 2));
        return panes.slice(startIndex, startIndex + maxVisible);
      }
    }

    return panes.slice(0, maxVisible);
  }, [panes, activePane, gridLayout]);

  // Calculate grid template
  const gridStyle = useMemo(() => {
    return {
      gridTemplateRows: `repeat(${gridLayout.rows}, 1fr)`,
      gridTemplateColumns: `repeat(${gridLayout.cols}, 1fr)`,
    };
  }, [gridLayout]);

  if (panes.length === 0) {
    return (
      <div className={cn("flex-1 flex items-center justify-center bg-[#1e1e1e]", className)}>
        <div className="text-center">
          <TerminalIcon className="w-16 h-16 mx-auto mb-4 text-[#3c3c3c]" />
          <h3 className="text-lg font-medium text-[#cccccc] mb-2">No terminals open</h3>
          <p className="text-sm text-[#808080] mb-4">
            Open a pod to start a terminal session
          </p>
          {onAddNew && (
            <Button
              onClick={onAddNew}
              className="bg-primary hover:bg-primary/90"
            >
              <Plus className="w-4 h-4 mr-2" />
              Open Terminal
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div
      className={cn("flex-1 grid gap-1 p-1 bg-[#1e1e1e]", className)}
      style={gridStyle}
    >
      {visiblePanes.map((pane, index) => (
        <TerminalPane
          key={pane.id}
          paneId={pane.id}
          podKey={pane.podKey}
          title={pane.title}
          isActive={pane.id === activePane}
          onClose={() => removePane(pane.id)}
          onPopout={onPopout ? () => onPopout(pane.id) : undefined}
          showHeader={true}
        />
      ))}

      {/* Fill empty grid cells with placeholder */}
      {visiblePanes.length < gridLayout.rows * gridLayout.cols &&
        Array.from({ length: gridLayout.rows * gridLayout.cols - visiblePanes.length }).map(
          (_, index) => (
            <div
              key={`empty-${index}`}
              className="flex items-center justify-center bg-[#252526] rounded-lg border border-dashed border-[#3c3c3c]"
            >
              {onAddNew && (
                <Button
                  variant="ghost"
                  className="text-[#808080] hover:text-[#cccccc] hover:bg-[#3c3c3c]"
                  onClick={onAddNew}
                >
                  <Plus className="w-5 h-5 mr-2" />
                  Add Terminal
                </Button>
              )}
            </div>
          )
        )}
    </div>
  );
}

export default TerminalGrid;
