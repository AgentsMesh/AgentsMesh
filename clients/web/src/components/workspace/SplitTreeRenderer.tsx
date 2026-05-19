"use client";

import React, { useCallback } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import type { SplitTreeNode } from "@/stores/workspace";
import { usePod } from "@/stores/pod";
import { POD_MODE_ACP } from "@/lib/pod-modes";
import { TerminalPane } from "./TerminalPane";
import { AgentPanel } from "./AgentPanel";

interface SplitTreeRendererProps {
  node: SplitTreeNode;
  onPopout?: (paneId: string) => void;
}

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

export function SplitTreeRenderer({ node, onPopout }: SplitTreeRendererProps) {
  const activePane = useWorkspaceStore((s) => s.activePane);
  const removePane = useWorkspaceStore((s) => s.removePane);
  const updateSplitSizes = useWorkspaceStore((s) => s.updateSplitSizes);

  const handleLayoutChange = useCallback(
    (splitId: string, layout: Record<string, number>) => {
      const values = Object.values(layout);
      if (values.length >= 2) {
        updateSplitSizes(splitId, values);
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

  const orientation = node.direction === "horizontal" ? "horizontal" : "vertical";

  return (
    <Group
      orientation={orientation}
      className="h-full"
      onLayoutChange={(layout) => handleLayoutChange(node.id, layout)}
    >
      {node.children.map((child, i) => (
        <React.Fragment key={child.id}>
          {i > 0 && <ResizeHandle direction={node.direction} />}
          <Panel defaultSize={node.sizes[i]} minSize={10}>
            <SplitTreeRenderer node={child} onPopout={onPopout} />
          </Panel>
        </React.Fragment>
      ))}
    </Group>
  );
}

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
  const pod = usePod(podKey);
  const interactionMode = pod?.interaction_mode;
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
