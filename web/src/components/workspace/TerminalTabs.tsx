"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodTitle } from "@/hooks/usePodTitle";
import { useTerminalStatus } from "@/hooks/useTerminalStatus";
import { Button } from "@/components/ui/button";
import {
  X,
  Plus,
  Circle,
  Maximize2,
  Minimize2,
} from "lucide-react";

interface TerminalTabsProps {
  onAddNew?: () => void;
  className?: string;
  isFullscreen?: boolean;
  onToggleFullscreen?: () => void;
}

export function TerminalTabs({ onAddNew, className, isFullscreen, onToggleFullscreen }: TerminalTabsProps) {
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const removePane = useWorkspaceStore((s) => s.removePane);

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

      {/* Fullscreen toggle */}
      {onToggleFullscreen && (
        <div className="flex items-center gap-1 px-2 border-l border-terminal-border">
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 text-terminal-text-muted hover:text-terminal-text-active"
            onClick={onToggleFullscreen}
            title="Fullscreen"
          >
            {isFullscreen ? (
              <Minimize2 className="w-3.5 h-3.5" />
            ) : (
              <Maximize2 className="w-3.5 h-3.5" />
            )}
          </Button>
        </div>
      )}
    </div>
  );
}

/** Reactive connection status dot — subscribes via useTerminalStatus hook. */
function ConnectionDot({ podKey }: { podKey: string }) {
  const { status } = useTerminalStatus(podKey);

  const statusClass = (() => {
    switch (status) {
      case "connected": return "bg-green-500";
      case "connecting": return "bg-yellow-500 animate-pulse";
      case "error": return "bg-red-500";
      default: return "bg-gray-500";
    }
  })();

  return <Circle className={cn("w-2 h-2 flex-shrink-0", statusClass)} />;
}

/** Reads pod title via usePodTitle hook — single source of truth. */
function TabPaneTitle({ podKey }: { podKey: string }) {
  const title = usePodTitle(podKey, `Pod ${podKey.substring(0, 8)}`);
  return <>{title}</>;
}

export default TerminalTabs;
