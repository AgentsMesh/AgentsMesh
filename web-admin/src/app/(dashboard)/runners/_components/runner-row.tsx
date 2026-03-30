"use client";

import { Power, PowerOff, Trash2, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { Runner } from "@/lib/api/admin";
import { formatRelativeTime } from "@/lib/utils";

interface RunnerRowProps {
  runner: Runner;
  onDisable: () => void;
  onEnable: () => void;
  onDelete: () => void;
}

export function RunnerRow({
  runner,
  onDisable,
  onEnable,
  onDelete,
}: RunnerRowProps) {
  const isOnline = runner.status === "online";

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary">
          <Server className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{runner.node_id}</span>
            <Badge variant={isOnline ? "success" : "secondary"}>
              {runner.status}
            </Badge>
            {!runner.is_enabled && (
              <Badge variant="destructive">Disabled</Badge>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            {runner.organization && (
              <span>{runner.organization.name}</span>
            )}
            {runner.runner_version && (
              <span>v{runner.runner_version}</span>
            )}
            <span>
              {runner.current_pods}/{runner.max_concurrent_pods} pods
            </span>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <div className="hidden text-right text-xs text-muted-foreground sm:block">
          {runner.last_heartbeat && (
            <p>Last seen {formatRelativeTime(runner.last_heartbeat)}</p>
          )}
          {runner.available_agents && runner.available_agents.length > 0 && (
            <p>{runner.available_agents.length} agents</p>
          )}
        </div>
        <div className="flex gap-1">
          {runner.is_enabled ? (
            <Button
              variant="ghost"
              size="icon"
              onClick={onDisable}
              title="Disable runner"
            >
              <PowerOff className="h-4 w-4" />
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="icon"
              onClick={onEnable}
              title="Enable runner"
            >
              <Power className="h-4 w-4" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            title="Delete runner"
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
