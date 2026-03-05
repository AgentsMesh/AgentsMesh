"use client";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import { usePodStore } from "@/stores/pod";
import { getPodDisplayName } from "@/lib/pod-utils";
import {
  X,
  Maximize2,
  Minimize2,
  ExternalLink,
  Circle,
  Scaling,
  Bot,
} from "lucide-react";

type ConnectionStatus = "connected" | "connecting" | "disconnected" | "error";

interface TerminalPaneHeaderProps {
  podKey: string;
  connectionStatus: ConnectionStatus;
  isMaximized: boolean;
  isPodReady: boolean;
  hasAutopilot: boolean;
  onSyncSize: () => void;
  onStartAutopilot: () => void;
  onPopout?: () => void;
  onMaximize: () => void;
  onClose?: () => void;
}

/**
 * Header component for TerminalPane.
 * Title is derived from podStore (single source of truth), not passed as prop.
 */
export function TerminalPaneHeader({
  podKey,
  connectionStatus,
  isMaximized,
  isPodReady,
  hasAutopilot,
  onSyncSize,
  onStartAutopilot,
  onPopout,
  onMaximize,
  onClose,
}: TerminalPaneHeaderProps) {
  // Derive title from podStore — single source of truth
  const title = usePodStore((state) => {
    const pod = state.pods.find((p) => p.pod_key === podKey);
    return pod ? getPodDisplayName(pod) : podKey.substring(0, 8);
  });

  const getStatusColor = () => {
    switch (connectionStatus) {
      case "connected":
        return "text-green-500 dark:text-green-400";
      case "connecting":
        return "text-yellow-500 dark:text-yellow-400 animate-pulse";
      case "disconnected":
        return "text-gray-500 dark:text-gray-400";
      case "error":
        return "text-red-500 dark:text-red-400";
      default:
        return "text-gray-500 dark:text-gray-400";
    }
  };

  return (
    <div className="h-8 flex items-center justify-between px-2 bg-terminal-bg-secondary border-b border-terminal-border">
      <div className="flex items-center gap-2 min-w-0">
        <Circle className={cn("w-2 h-2 flex-shrink-0", getStatusColor())} />
        <span className="text-xs text-terminal-text truncate">{title}</span>
        <code className="text-[10px] text-terminal-text-muted truncate">
          {podKey.substring(0, 8)}
        </code>
        <TerminalAgentStatus podKey={podKey} />
      </div>
      <div className="flex items-center gap-1 flex-shrink-0">
        <Button
          variant="ghost"
          size="sm"
          className="h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text"
          onClick={(e) => {
            e.stopPropagation();
            onSyncSize();
          }}
          title="Sync terminal size"
        >
          <Scaling className="w-3 h-3" />
        </Button>
        {/* Start Autopilot button - only show when no active AutopilotController */}
        {!hasAutopilot && isPodReady && (
          <Button
            variant="ghost"
            size="sm"
            className="h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text hover:text-primary"
            onClick={(e) => {
              e.stopPropagation();
              onStartAutopilot();
            }}
            title="Start Autopilot Mode"
          >
            <Bot className="w-3 h-3" />
          </Button>
        )}
        {onPopout && (
          <Button
            variant="ghost"
            size="sm"
            className="h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text"
            onClick={(e) => {
              e.stopPropagation();
              onPopout();
            }}
            title="Popout"
          >
            <ExternalLink className="w-3 h-3" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text"
          onClick={(e) => {
            e.stopPropagation();
            onMaximize();
          }}
          title={isMaximized ? "Restore" : "Maximize"}
        >
          {isMaximized ? (
            <Minimize2 className="w-3 h-3" />
          ) : (
            <Maximize2 className="w-3 h-3" />
          )}
        </Button>
        {onClose && (
          <Button
            variant="ghost"
            size="sm"
            className="h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text hover:text-red-500 dark:hover:text-red-400"
            onClick={(e) => {
              e.stopPropagation();
              onClose();
            }}
            title="Close"
          >
            <X className="w-3 h-3" />
          </Button>
        )}
      </div>
    </div>
  );
}

/**
 * Internal component that reads agent status from the pod store
 */
function TerminalAgentStatus({ podKey }: { podKey: string }) {
  const pod = usePodStore((state) => state.pods.find((p) => p.pod_key === podKey));
  if (!pod) return null;
  return (
    <AgentStatusBadge
      agentStatus={pod.agent_status}
      podStatus={pod.status}
      variant="inline"
    />
  );
}

export default TerminalPaneHeader;
