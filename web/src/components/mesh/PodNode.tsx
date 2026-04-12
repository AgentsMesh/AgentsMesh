"use client";

import { memo } from "react";
import { Handle, Position } from "@xyflow/react";
import { useParams, useRouter } from "next/navigation";
import { getPodStatusInfo, type MeshNode } from "@/stores/mesh";
import { getPodDisplayName } from "@/lib/pod-utils";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import PodContextMenu from "./PodContextMenu";

interface PodNodeProps {
  data: {
    node: MeshNode;
    isSelected?: boolean;
  };
}

function PodNode({ data }: PodNodeProps) {
  const { node, isSelected } = data;
  const statusInfo = getPodStatusInfo(node.status);
  const params = useParams();
  const router = useRouter();
  const orgSlug = params.org as string;

  const displayName = getPodDisplayName(
    {
      pod_key: node.pod_key,
      alias: node.alias,
      title: node.title,
      ticket: node.ticket_slug
        ? { slug: node.ticket_slug, title: node.ticket_title }
        : undefined,
    },
    16
  );

  const handleTicketClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (node.ticket_slug) {
      router.push(`/${orgSlug}/tickets/${node.ticket_slug}`);
    }
  };

  const startedTitle = node.started_at
    ? new Date(node.started_at).toLocaleString()
    : undefined;

  return (
    <PodContextMenu node={node}>
      <div
        title={startedTitle}
        className={`px-4 py-3 rounded-lg border-2 bg-background shadow-md min-w-[180px] transition-all ${
          isSelected
            ? "border-primary ring-2 ring-primary/20"
            : "border-border hover:border-primary/50"
        }`}
      >
        <Handle
          type="target"
          position={Position.Left}
          className="w-3 h-3 !bg-primary"
        />
        <Handle
          type="source"
          position={Position.Right}
          className="w-3 h-3 !bg-primary"
        />

        <div className="flex items-center justify-between mb-2">
          <code className="text-xs font-mono text-muted-foreground truncate mr-2">
            {displayName}
          </code>
          <span
            className={`w-2.5 h-2.5 rounded-full shrink-0 ${statusInfo.bgColor.replace("bg-", "bg-").replace("/30", "")} ${
              node.status === "running" ? "bg-green-500" :
              node.status === "initializing" ? "bg-blue-500" :
              node.status === "failed" ? "bg-red-500" : "bg-gray-400"
            }`}
            title={statusInfo.label}
          />
        </div>

        <AgentStatusBadge
          agentStatus={node.agent_status}
          podStatus={node.status}
          variant="badge"
        />

        {(node.model || node.ticket_slug) && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground mt-1">
            {node.model && (
              <span className="truncate">{node.model}</span>
            )}
            {node.model && node.ticket_slug && (
              <span className="text-border">|</span>
            )}
            {node.ticket_slug && (
              <button
                type="button"
                onClick={handleTicketClick}
                className="nodrag nopan font-medium text-primary hover:underline cursor-pointer truncate"
              >
                {node.ticket_slug}
              </button>
            )}
          </div>
        )}
      </div>
    </PodContextMenu>
  );
}

export default memo(PodNode);
