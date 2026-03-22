"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import { usePodStore } from "@/stores/pod";
import { useAcpSessionStore } from "@/stores/acpSession";
import { usePodTitle } from "@/hooks/usePodTitle";
import { getShortPodKey } from "@/lib/pod-utils";
import {
  X,
  Maximize2,
  Minimize2,
  ExternalLink,
  Circle,
  PanelRight,
  PanelBottom,
} from "lucide-react";

interface AgentPanelHeaderProps {
  podKey: string;
  isMaximized: boolean;
  onPopout?: () => void;
  onSplitRight?: () => void;
  onSplitDown?: () => void;
  onMaximize: () => void;
  onClose?: () => void;
}

export function AgentPanelHeader({
  podKey,
  isMaximized,
  onPopout,
  onSplitRight,
  onSplitDown,
  onMaximize,
  onClose,
}: AgentPanelHeaderProps) {
  const title = usePodTitle(podKey);
  const sessionState = useAcpSessionStore(
    (s) => s.sessions[podKey]?.state ?? "idle"
  );

  const statusColor = (() => {
    switch (sessionState) {
      case "processing":
        return "text-blue-500 dark:text-blue-400 animate-pulse";
      case "waiting_permission":
        return "text-amber-500 dark:text-amber-400 animate-pulse";
      default:
        return "text-green-500 dark:text-green-400";
    }
  })();

  return (
    <div className="h-8 flex items-center justify-between px-2 bg-terminal-bg-secondary border-b border-terminal-border">
      <div className="flex items-center gap-2 min-w-0">
        <Circle className={cn("w-2 h-2 flex-shrink-0", statusColor)} />
        <span className="text-xs text-terminal-text truncate">{title}</span>
        <code className="text-[10px] text-terminal-text-muted truncate">
          {getShortPodKey(podKey)}
        </code>
        <AgentPanelAgentStatus podKey={podKey} />
      </div>
      <div className="flex items-center gap-1 flex-shrink-0">
        {onSplitRight && (
          <IconButton
            onClick={onSplitRight}
            title="Split Right"
            icon={<PanelRight className="w-3 h-3" />}
          />
        )}
        {onSplitDown && (
          <IconButton
            onClick={onSplitDown}
            title="Split Down"
            icon={<PanelBottom className="w-3 h-3" />}
          />
        )}
        {onPopout && (
          <IconButton
            onClick={onPopout}
            title="Popout"
            icon={<ExternalLink className="w-3 h-3" />}
          />
        )}
        <IconButton
          onClick={onMaximize}
          title={isMaximized ? "Restore" : "Maximize"}
          icon={
            isMaximized ? (
              <Minimize2 className="w-3 h-3" />
            ) : (
              <Maximize2 className="w-3 h-3" />
            )
          }
        />
        {onClose && (
          <IconButton
            onClick={onClose}
            title="Close"
            icon={<X className="w-3 h-3" />}
            hoverClass="hover:text-red-500 dark:hover:text-red-400"
          />
        )}
      </div>
    </div>
  );
}

function IconButton({
  onClick,
  title,
  icon,
  hoverClass,
}: {
  onClick: () => void;
  title: string;
  icon: React.ReactNode;
  hoverClass?: string;
}) {
  return (
    <Button
      variant="ghost"
      size="sm"
      className={cn(
        "h-5 w-5 p-0 hover:bg-terminal-bg-active text-terminal-text",
        hoverClass
      )}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      title={title}
    >
      {icon}
    </Button>
  );
}

function AgentPanelAgentStatus({ podKey }: { podKey: string }) {
  const pod = usePodStore(
    (state) => state.pods.find((p) => p.pod_key === podKey)
  );
  if (!pod) return null;
  return (
    <AgentStatusBadge
      agentStatus={pod.agent_status}
      podStatus={pod.status}
      variant="inline"
    />
  );
}
