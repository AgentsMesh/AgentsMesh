"use client";

import { cn } from "@/lib/utils";
import { getPodDisplayName } from "@/lib/pod-utils";
import { Pod } from "@/stores/pod";
import { Button } from "@/components/ui/button";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import {
  Square,
  Terminal,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
} from "lucide-react";

// Status badge colors - matches PodData status type
const statusColors: Record<string, { bg: string; text: string; dot: string }> = {
  initializing: { bg: "bg-yellow-500/10", text: "text-yellow-600 dark:text-yellow-400", dot: "bg-yellow-500" },
  running: { bg: "bg-blue-500/10", text: "text-blue-600 dark:text-blue-400", dot: "bg-blue-500" },
  paused: { bg: "bg-orange-500/10", text: "text-orange-600 dark:text-orange-400", dot: "bg-orange-500" },
  disconnected: { bg: "bg-gray-500/10", text: "text-gray-600 dark:text-gray-400", dot: "bg-gray-500" },
  orphaned: { bg: "bg-purple-500/10", text: "text-purple-600 dark:text-purple-400", dot: "bg-purple-500" },
  completed: { bg: "bg-green-500/10", text: "text-green-600 dark:text-green-400", dot: "bg-green-500" },
  terminated: { bg: "bg-gray-500/10", text: "text-gray-600 dark:text-gray-400", dot: "bg-gray-500" },
  error: { bg: "bg-red-500/10", text: "text-red-600 dark:text-red-400", dot: "bg-red-500" },
  failed: { bg: "bg-red-500/10", text: "text-red-600 dark:text-red-400", dot: "bg-red-500" },
};

function getStatusIcon(status: string) {
  switch (status) {
    case "initializing":
      return <Clock className="w-3 h-3" />;
    case "running":
      return <Loader2 className="w-3 h-3 animate-spin" />;
    case "paused":
      return <Square className="w-3 h-3" />;
    case "terminated":
      return <CheckCircle className="w-3 h-3" />;
    case "failed":
      return <XCircle className="w-3 h-3" />;
    default:
      return <Square className="w-3 h-3" />;
  }
}

interface PodListItemProps {
  pod: Pod;
  isOpen: boolean;
  onClick: () => void;
  onTerminate: (e: React.MouseEvent) => void;
}

/**
 * Single pod item in the workspace sidebar list
 */
export function PodListItem({ pod, isOpen, onClick, onTerminate }: PodListItemProps) {
  const status = statusColors[pod.status] || statusColors.terminated;

  return (
    <div
      className={cn(
        "group flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer",
        isOpen && "bg-muted/30"
      )}
      onClick={onClick}
    >
      {/* Status indicator */}
      <div className={cn("flex items-center justify-center", status.text)}>
        {getStatusIcon(pod.status)}
      </div>

      {/* Pod info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-sm truncate font-mono">
            {getPodDisplayName(pod)}
          </span>
          <AgentStatusBadge
            agentStatus={pod.agent_status}
            podStatus={pod.status}
            variant="dot"
          />
          {isOpen && (
            <Terminal className="w-3 h-3 text-primary flex-shrink-0" />
          )}
        </div>
        {/* Show ticket if title is not from ticket */}
        {!pod.title && pod.ticket?.identifier && (
          <p className="text-xs text-muted-foreground truncate">
            {pod.ticket.identifier}
          </p>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        {pod.status === "running" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 text-destructive hover:text-destructive"
            onClick={onTerminate}
          >
            <Square className="w-3 h-3" />
          </Button>
        )}
      </div>
    </div>
  );
}

export { statusColors };
