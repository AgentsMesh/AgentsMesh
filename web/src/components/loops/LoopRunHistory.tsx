"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { LoopRunData } from "@/stores/loop";
import { Button } from "@/components/ui/button";
import {
  CheckCircle2,
  XCircle,
  Clock,
  Ban,
  SkipForward,
  Loader2,
  Terminal,
  FileText,
  AlertTriangle,
  User,
  Link2,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { formatDuration, formatTimeAgo } from "@/lib/utils/time";

interface StatusConfig {
  icon: React.ElementType;
  color: string;
  bg: string;
  labelKey: string;
}

const STATUS_CONFIG: Record<string, StatusConfig> = {
  completed: {
    icon: CheckCircle2,
    color: "text-emerald-600 dark:text-emerald-400",
    bg: "bg-emerald-500/10",
    labelKey: "loops.statusCompleted",
  },
  failed: {
    icon: XCircle,
    color: "text-red-600 dark:text-red-400",
    bg: "bg-red-500/10",
    labelKey: "loops.statusFailed",
  },
  timeout: {
    icon: AlertTriangle,
    color: "text-amber-600 dark:text-amber-400",
    bg: "bg-amber-500/10",
    labelKey: "loops.statusTimeout",
  },
  cancelled: {
    icon: Ban,
    color: "text-gray-500",
    bg: "bg-gray-500/10",
    labelKey: "loops.statusCancelled",
  },
  skipped: {
    icon: SkipForward,
    color: "text-gray-400",
    bg: "bg-gray-500/10",
    labelKey: "loops.statusSkipped",
  },
  running: {
    icon: Loader2,
    color: "text-blue-600 dark:text-blue-400",
    bg: "bg-blue-500/10",
    labelKey: "loops.statusRunning",
  },
  pending: {
    icon: Clock,
    color: "text-yellow-600 dark:text-yellow-400",
    bg: "bg-yellow-500/10",
    labelKey: "loops.statusPending",
  },
};

interface LoopRunHistoryProps {
  runs: LoopRunData[];
  loading: boolean;
  total: number;
  onLoadMore?: () => void;
  onViewTerminal?: (podKey: string) => void;
  onCancel?: (runId: number) => void;
  className?: string;
}

export function LoopRunHistory({
  runs,
  loading,
  total,
  onLoadMore,
  onViewTerminal,
  onCancel,
  className,
}: LoopRunHistoryProps) {
  const t = useTranslations();

  if (loading && runs.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center mb-3">
          <Clock className="w-5 h-5 text-muted-foreground" />
        </div>
        <p className="text-sm text-muted-foreground">{t("loops.noRuns")}</p>
      </div>
    );
  }

  return (
    <div className={cn("space-y-1.5", className)}>
      {runs.map((run) => (
        <RunRow
          key={run.id}
          run={run}
          onViewTerminal={onViewTerminal}
          onCancel={onCancel}
        />
      ))}

      {/* Load more */}
      {runs.length < total && onLoadMore && (
        <div className="flex justify-center pt-2">
          <Button
            size="sm"
            variant="ghost"
            className="text-xs text-muted-foreground"
            onClick={onLoadMore}
            disabled={loading}
          >
            {loading ? (
              <Loader2 className="w-3 h-3 animate-spin mr-1" />
            ) : null}
            {t("loops.loadMore")}
          </Button>
        </div>
      )}
    </div>
  );
}

function RunRow({
  run,
  onViewTerminal,
  onCancel,
}: {
  run: LoopRunData;
  onViewTerminal?: (podKey: string) => void;
  onCancel?: (runId: number) => void;
}) {
  const t = useTranslations();
  const [expanded, setExpanded] = useState(false);
  const config = STATUS_CONFIG[run.status] || STATUS_CONFIG.pending;
  const StatusIcon = config.icon;
  const isRunning = run.status === "running";
  const hasDetails = run.error_message || run.exit_summary || run.resolved_prompt;

  return (
    <div>
      <div
        className={cn(
          "flex items-center flex-wrap md:flex-nowrap gap-x-3 gap-y-1 px-3 py-2.5 rounded-lg",
          "border border-transparent",
          "hover:bg-accent/50 hover:border-border/50",
          "transition-colors duration-150",
          isRunning && "bg-blue-500/5 border-blue-500/20",
          hasDetails && "cursor-pointer",
          hasDetails && "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
        )}
        role={hasDetails ? "button" : undefined}
        tabIndex={hasDetails ? 0 : undefined}
        aria-expanded={hasDetails ? expanded : undefined}
        onClick={() => hasDetails && setExpanded(!expanded)}
        onKeyDown={hasDetails ? (e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setExpanded(!expanded);
          }
        } : undefined}
      >
        {/* Expand indicator */}
        <span className="w-3 flex-shrink-0" aria-hidden="true">
          {hasDetails ? (
            expanded ? (
              <ChevronDown className="w-3 h-3 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-3 h-3 text-muted-foreground" />
            )
          ) : null}
        </span>

        {/* Run number */}
        <span className="text-xs text-muted-foreground tabular-nums w-8 flex-shrink-0">
          #{run.run_number}
        </span>

        {/* Status badge */}
        <div
          className={cn(
            "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium w-24 flex-shrink-0",
            config.bg,
            config.color
          )}
        >
          <StatusIcon
            className={cn("w-3 h-3 flex-shrink-0", isRunning && "animate-spin")}
          />
          <span className="truncate">{t(config.labelKey)}</span>
        </div>

        {/* Trigger type */}
        <span className="hidden sm:inline-flex items-center gap-1 text-xs text-muted-foreground w-16 flex-shrink-0">
          {run.trigger_type === "manual" ? (
            <User className="w-3 h-3 flex-shrink-0" />
          ) : run.trigger_type === "api" ? (
            <Link2 className="w-3 h-3 flex-shrink-0" />
          ) : (
            <Clock className="w-3 h-3 flex-shrink-0" />
          )}
          <span>
            {run.trigger_type === "cron"
              ? t("loops.triggerTypeCron")
              : run.trigger_type === "api"
                ? t("loops.triggerTypeApi")
                : t("loops.triggerTypeManual")}
          </span>
        </span>

        {/* Started time */}
        <span className="text-xs text-muted-foreground flex-1 min-w-0 truncate">
          {run.started_at ? formatTimeAgo(run.started_at, t) : "-"}
        </span>

        {/* Duration */}
        <span className="text-xs text-muted-foreground tabular-nums w-16 text-right flex-shrink-0">
          {run.duration_sec ? formatDuration(run.duration_sec) : "-"}
        </span>

        {/* Actions */}
        <div
          className="flex items-center gap-1 flex-shrink-0 justify-end"
          onClick={(e) => e.stopPropagation()}
        >
          {run.pod_key && onViewTerminal && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 text-xs gap-1"
              onClick={(e) => {
                e.stopPropagation();
                onViewTerminal(run.pod_key!);
              }}
            >
              {run.status === "running" || run.status === "pending" ? (
                <Terminal className="w-3 h-3" />
              ) : (
                <FileText className="w-3 h-3" />
              )}
              {run.status === "running" || run.status === "pending"
                ? t("loops.viewTerminal")
                : t("loops.viewLogs")}
            </Button>
          )}
          {isRunning && onCancel && (
            <Button
              size="sm"
              variant="ghost"
              className="h-6 px-1.5 text-xs text-destructive hover:text-destructive"
              onClick={() => onCancel(run.id)}
            >
              {t("common.cancel")}
            </Button>
          )}
        </div>
      </div>

      {/* Expanded details */}
      {expanded && hasDetails && (
        <div className="ml-6 mr-3 mb-1 p-3 rounded-lg bg-muted/30 border border-border/50 space-y-2 text-xs">
          {run.error_message && (
            <div>
              <span className="font-medium text-red-500">{t("loops.errorMessage")}:</span>
              <span className="ml-1.5 text-foreground/80">{run.error_message}</span>
            </div>
          )}
          {run.exit_summary && (
            <div>
              <span className="font-medium text-muted-foreground">{t("loops.exitSummary")}:</span>
              <span className="ml-1.5 text-foreground/80">{run.exit_summary}</span>
            </div>
          )}
          {run.resolved_prompt && (
            <div>
              <span className="font-medium text-muted-foreground">{t("loops.resolvedPrompt")}:</span>
              <pre className="mt-1 p-2 bg-muted/50 rounded text-foreground/70 font-mono whitespace-pre-wrap max-h-24 overflow-y-auto">
                {run.resolved_prompt}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
