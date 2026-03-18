"use client";

import React, { useCallback } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import type { SplitTreeNode } from "@/stores/workspace";
import { TerminalPane } from "./TerminalPane";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface SplitTreeRendererProps {
  node: SplitTreeNode;
  onPopout?: (paneId: string) => void;
  onAddNew?: () => void;
}

/**
 * VS Code style resize handle — hidden by default, highlights on hover
 */
function ResizeHandle({ direction }: { direction: "horizontal" | "vertical" }) {
  const isHorizontal = direction === "horizontal";
  return (
    <Separator
      className={cn(
        "group relative flex items-center justify-center bg-transparent transition-colors",
        isHorizontal
          ? "w-1 cursor-col-resize hover:bg-primary"
          : "h-1 cursor-row-resize hover:bg-primary"
      )}
    >
      <div
        className={cn(
          "absolute z-10",
          isHorizontal ? "w-3 h-full -left-1" : "h-3 w-full -top-1"
        )}
      />
    </Separator>
  );
}

/**
 * Empty pane slot placeholder
 */
function EmptyPaneSlot({ onAddNew }: { onAddNew?: () => void }) {
  return (
    <div className="flex items-center justify-center h-full bg-terminal-bg-secondary rounded-lg border border-dashed border-terminal-border">
      {onAddNew && (
        <Button
          variant="ghost"
          className="text-terminal-text-muted hover:text-terminal-text hover:bg-terminal-bg-active"
          onClick={onAddNew}
        >
          <Plus className="w-5 h-5 mr-2" />
          Add Terminal
        </Button>
      )}
    </div>
  );
}

/**
 * Recursive renderer for a SplitTreeNode
 */
export function SplitTreeRenderer({ node, onPopout, onAddNew }: SplitTreeRendererProps) {
  const activePane = useWorkspaceStore((s) => s.activePane);
  const removePane = useWorkspaceStore((s) => s.removePane);
  const updateSplitSizes = useWorkspaceStore((s) => s.updateSplitSizes);

  const handleLayoutChange = useCallback(
    (splitId: string, layout: Record<string, number>) => {
      const values = Object.values(layout);
      if (values.length === 2) {
        updateSplitSizes(splitId, [values[0], values[1]]);
      }
    },
    [updateSplitSizes]
  );

  if (node.type === "leaf") {
    if (!node.paneId) {
      return <EmptyPaneSlot onAddNew={onAddNew} />;
    }
    return (
      <LeafPane
        paneId={node.paneId}
        activePane={activePane}
        onClose={removePane}
        onPopout={onPopout}
      />
    );
  }

  // Split node
  const orientation = node.direction === "horizontal" ? "horizontal" : "vertical";

  return (
    <Group
      orientation={orientation}
      className="h-full"
      onLayoutChange={(layout) => handleLayoutChange(node.id, layout)}
    >
      <Panel defaultSize={node.sizes[0]} minSize={10}>
        <SplitTreeRenderer
          node={node.children[0]}
          onPopout={onPopout}
          onAddNew={onAddNew}
        />
      </Panel>
      <ResizeHandle direction={node.direction} />
      <Panel defaultSize={node.sizes[1]} minSize={10}>
        <SplitTreeRenderer
          node={node.children[1]}
          onPopout={onPopout}
          onAddNew={onAddNew}
        />
      </Panel>
    </Group>
  );
}

/**
 * Leaf pane wrapper — subscribes reactively to pane data from the store
 * instead of using getState() which breaks reactivity.
 */
function LeafPane({
  paneId,
  activePane,
  onClose,
  onPopout,
}: {
  paneId: string;
  activePane: string | null;
  onClose: (id: string) => void;
  onPopout?: (paneId: string) => void;
}) {
  const podKey = useWorkspaceStore((s) => s.panes.find((p) => p.id === paneId)?.podKey);
  if (!podKey) return null;

  return (
    <TerminalPane
      paneId={paneId}
      podKey={podKey}
      isActive={paneId === activePane}
      onClose={() => onClose(paneId)}
      onPopout={onPopout ? () => onPopout(paneId) : undefined}
      showHeader={true}
    />
  );
}

export default SplitTreeRenderer;
