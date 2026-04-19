"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { MeshNode, getPodStatusInfo } from "@/stores/mesh";
import { Button } from "@/components/ui/button";
import {
  Users,
  Loader2,
  ChevronDown,
  ChevronRight,
  Terminal,
} from "lucide-react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

interface MeshNodesListProps {
  nodes: MeshNode[];
  loading: boolean;
  expanded: boolean;
  onToggle: (expanded: boolean) => void;
  selectedNodeId: string | null;
  onNodeClick: (node: MeshNode) => void;
  onOpenTerminal: (podKey: string, e: React.MouseEvent) => void;
  t: (key: string) => string;
}

/**
 * Collapsible nodes (pods) list in mesh sidebar
 */
export function MeshNodesList({
  nodes,
  loading,
  expanded,
  onToggle,
  selectedNodeId,
  onNodeClick,
  onOpenTerminal,
  t,
}: MeshNodesListProps) {
  return (
    <Collapsible open={expanded} onOpenChange={onToggle}>
      <CollapsibleTrigger asChild>
        <div className="flex items-center justify-between px-3 py-2 border-t border-border cursor-pointer hover:bg-muted/50">
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4 text-muted-foreground" />
            <span className="text-sm font-medium">{t("ide.sidebar.mesh.podsSection")}</span>
            <span className="text-xs text-muted-foreground">
              ({nodes.length})
            </span>
          </div>
          {expanded ? (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="w-4 h-4 text-muted-foreground" />
          )}
        </div>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="flex-1 overflow-y-auto max-h-48">
          {loading && nodes.length === 0 ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
            </div>
          ) : nodes.length === 0 ? (
            <div className="px-3 py-4 text-center text-xs text-muted-foreground">
              {t("ide.sidebar.mesh.noPods")}
            </div>
          ) : (
            <div className="py-1">
              {nodes.map((node) => {
                const isSelected = selectedNodeId === node.pod_key;
                const statusInfo = getPodStatusInfo(node.status);
                return (
                  <div
                    key={node.pod_key}
                    className={cn(
                      "group flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-muted/50",
                      isSelected && "bg-muted/30"
                    )}
                    onClick={() => onNodeClick(node)}
                  >
                    <span className={cn("w-2 h-2 rounded-full flex-shrink-0", statusInfo.bgColor)} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{getPodDisplayName(node)}</p>
                      {node.model && (
                        <p className="text-xs text-muted-foreground">{node.model}</p>
                      )}
                    </div>
                    {/* Open terminal button */}
                    {(node.status === "running" || node.status === "initializing") && (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
                        onClick={(e) => onOpenTerminal(node.pod_key, e)}
                      >
                        <Terminal className="w-3 h-3" />
                      </Button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

export default MeshNodesList;
