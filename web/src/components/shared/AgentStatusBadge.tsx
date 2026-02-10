"use client";

import { getAgentStatusInfo } from "@/stores/mesh";
import { cn } from "@/lib/utils";

interface AgentStatusBadgeProps {
  agentStatus: string;
  podStatus: string;
  variant?: "dot" | "badge" | "inline";
  className?: string;
}

export function AgentStatusBadge({
  agentStatus,
  podStatus,
  variant = "badge",
  className,
}: AgentStatusBadgeProps) {
  // Only show for active (running) pods
  if (podStatus !== "running") {
    return null;
  }

  const info = getAgentStatusInfo(agentStatus);
  const Icon = info.icon;

  if (variant === "dot") {
    return (
      <span
        className={cn(
          "inline-block w-2 h-2 rounded-full",
          info.dotColor,
          agentStatus === "executing" && "animate-pulse",
          className
        )}
        title={info.label}
      />
    );
  }

  if (variant === "inline") {
    return (
      <span className={cn("inline-flex items-center gap-1.5", info.color, className)}>
        <Icon className="w-3.5 h-3.5" />
        <span className="text-sm">{info.label}</span>
      </span>
    );
  }

  // badge variant (default)
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium",
        info.bgColor,
        info.color,
        className
      )}
    >
      <Icon className="w-3 h-3" />
      {info.label}
    </span>
  );
}
