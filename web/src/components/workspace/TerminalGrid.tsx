"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { SplitTreeRenderer } from "./SplitTreeRenderer";
import { Terminal as TerminalIcon, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TerminalGridProps {
  onPopout?: (paneId: string) => void;
  onAddNew?: () => void;
  className?: string;
}

export function TerminalGrid({ onPopout, onAddNew, className }: TerminalGridProps) {
  const panes = useWorkspaceStore((s) => s.panes);
  const splitTree = useWorkspaceStore((s) => s.splitTree);

  // Empty state
  if (panes.length === 0 || !splitTree) {
    return (
      <div className={cn("flex-1 flex items-center justify-center bg-terminal-bg", className)}>
        <div className="text-center">
          <TerminalIcon className="w-16 h-16 mx-auto mb-4 text-terminal-border" />
          <h3 className="text-lg font-medium text-terminal-text mb-2">No terminals open</h3>
          <p className="text-sm text-terminal-text-muted mb-4">
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
    <div className={cn("flex-1 p-1 bg-terminal-bg", className)}>
      <SplitTreeRenderer
        node={splitTree}
        onPopout={onPopout}
      />
    </div>
  );
}

export default TerminalGrid;
