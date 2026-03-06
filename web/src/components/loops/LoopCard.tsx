"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { LoopData } from "@/stores/loop";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Clock,
  MoreHorizontal,
  Play,
  Pencil,
  Power,
  Trash2,
  Loader2,
  Bot,
  Zap,
  ArrowUpRight,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { formatTimeAgo, formatTimeUntil } from "@/lib/utils/time";

// --- Component ---

interface LoopCardProps {
  loop: LoopData;
  onClick: (slug: string) => void;
  onTrigger: (slug: string) => void;
  onEnable: (slug: string) => void;
  onDisable: (slug: string) => void;
  onEdit: (loop: LoopData) => void;
  onDelete: (slug: string) => void;
  triggering?: boolean;
}

export function LoopCard({
  loop,
  onClick,
  onTrigger,
  onEnable,
  onDisable,
  onEdit,
  onDelete,
  triggering,
}: LoopCardProps) {
  const t = useTranslations();
  const isEnabled = loop.status === "enabled";
  const isRunning = loop.active_run_count > 0;
  const successRate =
    loop.total_runs > 0
      ? Math.round((loop.successful_runs / loop.total_runs) * 100)
      : null;

  return (
    <div
      role="button"
      tabIndex={0}
      className={cn(
        "group relative border rounded-xl p-4 cursor-pointer",
        "bg-card hover:bg-accent/50",
        "transition-all duration-200 ease-out",
        "hover:shadow-md hover:border-primary/30",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        !isEnabled && "opacity-50"
      )}
      onClick={() => onClick(loop.slug)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick(loop.slug);
        }
      }}
      aria-label={loop.name}
    >
      {/* Header row */}
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex items-center gap-2.5 min-w-0">
          {/* Status indicator */}
          <div className="relative flex-shrink-0">
            <span
              className={cn(
                "block w-2.5 h-2.5 rounded-full",
                isRunning ? "bg-blue-500" : isEnabled ? "bg-emerald-500" : "bg-gray-400 dark:bg-gray-600"
              )}
            />
            {/* Pulse animation for running loops only */}
            {isRunning && (
              <span className="absolute inset-0 w-2.5 h-2.5 rounded-full animate-ping opacity-30 bg-blue-500" />
            )}
          </div>
          <h3 className="font-semibold text-sm truncate leading-tight">{loop.name}</h3>
        </div>

        {/* More menu */}
        <div onClick={(e) => e.stopPropagation()}>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 p-0 opacity-60 md:opacity-0 md:group-hover:opacity-100 focus-visible:opacity-100 transition-opacity"
              >
                <MoreHorizontal className="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onEdit(loop)}>
                <Pencil className="w-4 h-4 mr-2" />
                {t("common.edit")}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {isEnabled ? (
                <DropdownMenuItem onClick={() => onDisable(loop.slug)}>
                  <Power className="w-4 h-4 mr-2" />
                  {t("loops.disable")}
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem onClick={() => onEnable(loop.slug)}>
                  <Power className="w-4 h-4 mr-2" />
                  {t("loops.enable")}
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => onDelete(loop.slug)}
              >
                <Trash2 className="w-4 h-4 mr-2" />
                {t("common.delete")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Tags row */}
      <div className="flex flex-wrap items-center gap-1.5 mb-3">
        {/* Trigger type badges */}
        {loop.cron_expression ? (
          <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-amber-500/10 text-amber-600 dark:text-amber-400 text-[10px] font-medium font-mono">
            <Clock className="w-3 h-3" />
            {loop.cron_expression}
          </span>
        ) : (
          <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-muted text-muted-foreground text-[10px] font-medium">
            <Play className="w-3 h-3" />
            {t("loops.onDemand")}
          </span>
        )}
        {/* Mode badge */}
        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-muted text-muted-foreground text-[10px] font-medium">
          {loop.execution_mode === "autopilot" ? (
            <Bot className="w-3 h-3" />
          ) : (
            <Zap className="w-3 h-3" />
          )}
          {loop.execution_mode === "autopilot" ? t("loops.modeAutopilot") : t("loops.modeDirect")}
        </span>
      </div>

      {/* Running indicator */}
      {isRunning && (
        <div className="flex items-center gap-1.5 mb-3 px-2 py-1 rounded-md bg-blue-500/10 text-blue-600 dark:text-blue-400 text-xs">
          <Loader2 className="w-3 h-3 animate-spin" />
          <span>{t("loops.runningCount", { count: loop.active_run_count })}</span>
        </div>
      )}

      {/* Stats row */}
      <div className="flex items-center gap-3 mb-3">
        {loop.total_runs > 0 ? (
          <>
            <span className="text-xs text-muted-foreground">
              {t("loops.runCount", { count: loop.total_runs })}
            </span>
            <span className="text-xs text-muted-foreground">|</span>
            {/* Success rate mini bar */}
            <div className="flex items-center gap-1.5">
              <div className="w-16 h-1.5 rounded-full bg-muted overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all duration-500",
                    successRate !== null && successRate >= 80
                      ? "bg-emerald-500"
                      : successRate !== null && successRate >= 50
                        ? "bg-amber-500"
                        : "bg-red-500"
                  )}
                  style={{ width: `${successRate ?? 0}%` }}
                />
              </div>
              <span className="text-xs font-medium tabular-nums">
                {successRate}%
              </span>
            </div>
          </>
        ) : (
          <span className="text-xs text-muted-foreground italic">
            {t("loops.noRunsYet")}
          </span>
        )}
      </div>

      {/* Timing row */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        {loop.last_run_at && (
          <div className="flex items-center gap-1">
            <Clock className="w-3 h-3 flex-shrink-0" />
            <span>{formatTimeAgo(loop.last_run_at, t)}</span>
          </div>
        )}
        {loop.next_run_at && isEnabled && loop.cron_expression && (
          <div className="flex items-center gap-1">
            <Clock className="w-3 h-3 flex-shrink-0" />
            <span>{formatTimeUntil(loop.next_run_at, t)}</span>
          </div>
        )}
      </div>

      {/* Footer actions */}
      <div
        className="flex items-center justify-between mt-3 pt-3 border-t border-border/50"
        onClick={(e) => e.stopPropagation()}
      >
        {isEnabled ? (
          <Button
            size="sm"
            variant="default"
            className="h-7 text-xs gap-1.5"
            onClick={() => onTrigger(loop.slug)}
            disabled={triggering || loop.active_run_count >= loop.max_concurrent_runs}
          >
            {triggering ? (
              <Loader2 className="w-3 h-3 animate-spin" />
            ) : (
              <Play className="w-3 h-3" />
            )}
            {t("loops.trigger")}
          </Button>
        ) : (
          <Button
            size="sm"
            variant="outline"
            className="h-7 text-xs gap-1.5"
            onClick={() => onEnable(loop.slug)}
          >
            <Power className="w-3 h-3" />
            {t("loops.enable")}
          </Button>
        )}

        <Button
          size="sm"
          variant="ghost"
          tabIndex={-1}
          className="h-7 text-xs gap-1 text-muted-foreground opacity-60 md:opacity-0 md:group-hover:opacity-100 transition-opacity"
          onClick={() => onClick(loop.slug)}
        >
          {t("loops.details")}
          <ArrowUpRight className="w-3 h-3" />
        </Button>
      </div>
    </div>
  );
}
