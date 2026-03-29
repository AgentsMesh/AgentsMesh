"use client";

import React, { useCallback } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import type { SplitTreeNode } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { POD_MODE_ACP } from "@/lib/pod-modes";
import { TerminalPane } from "./TerminalPane";
import { AgentPanel } from "./AgentPanel";

interface SplitTreeRendererProps {
  node: SplitTreeNode;
  onPopout?: (paneId: string) => void;
}

/**
 * VS Code style resize handle — hidden by default, highlights on hover
 */
function ResizeHandle({ direction }: { direction: "horizontal" | "vertical" }) {
  const isHorizontal = direction === "horizontal";
  return (
    <Separator
      className={cn(
        "bg-transparent transition-colors",
        isHorizontal
          ? "w-1 cursor-col-resize hover:bg-primary"
          : "h-1 cursor-row-resize hover:bg-primary"
      )}
    />
  );
}

/**
 * Recursive renderer for a SplitTreeNode
 */
export function SplitTreeRenderer({ node, onPopout }: SplitTreeRendererProps) {
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
        />
      </Panel>
      <ResizeHandle direction={node.direction} />
      <Panel defaultSize={node.sizes[1]} minSize={10}>
        <SplitTreeRenderer
          node={node.children[1]}
          onPopout={onPopout}
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
  const interactionMode = usePodStore(
    (s) => s.pods.find((p) => p.pod_key === podKey)?.interaction_mode
  );
  if (!podKey) return null;

  const sharedProps = {
    paneId,
    podKey,
    isActive: paneId === activePane,
    onClose: () => onClose(paneId),
    onPopout: onPopout ? () => onPopout(paneId) : undefined,
    showHeader: true,
  };

  if (interactionMode === POD_MODE_ACP) {
    return <AgentPanel {...sharedProps} />;
  }
  return <TerminalPane {...sharedProps} />;
}

export default SplitTreeRenderer;
