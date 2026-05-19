"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Progress } from "@/components/ui/progress";
import { AutopilotThinking } from "@/stores/autopilot";
import { CheckCircle, Clock, ListChecks } from "lucide-react";

interface AutopilotProgressTrackerProps {
  thinking: AutopilotThinking | null;
  className?: string;
  compact?: boolean;
}

export function AutopilotProgressTracker({
  thinking,
  className,
  compact = false,
}: AutopilotProgressTrackerProps) {
  const progress = thinking?.progress;

  if (!progress) {
    return null;
  }

  const hasSteps =
    (progress.completed_steps && progress.completed_steps.length > 0) ||
    (progress.remaining_steps && progress.remaining_steps.length > 0);

  if (compact) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <ListChecks className="h-3 w-3 text-muted-foreground flex-shrink-0" />
        <span className="text-xs text-muted-foreground truncate">
          {progress.summary || "In progress..."}
        </span>
        {progress.percent > 0 && (
          <span className="text-xs font-medium text-muted-foreground">
            {progress.percent}%
          </span>
        )}
      </div>
    );
  }

  return (
    <div className={cn("rounded-lg border bg-card p-4", className)}>
      {/* Header with Progress */}
      <div className="mb-3">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <ListChecks className="h-4 w-4 text-primary" />
            <span className="text-sm font-medium">Task Progress</span>
          </div>
          {progress.percent > 0 && (
            <span className="text-sm font-medium">{progress.percent}%</span>
          )}
        </div>
        {progress.percent > 0 && (
          <Progress value={progress.percent} className="h-2" />
        )}
      </div>

      {/* Summary */}
      {progress.summary && (
        <p className="text-sm text-muted-foreground mb-3">{progress.summary}</p>
      )}

      {/* Steps */}
      {hasSteps && (
        <div className="space-y-3">
          {/* Completed Steps */}
          {progress.completed_steps && progress.completed_steps.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
                <CheckCircle className="h-3 w-3 text-green-500" />
                <span>Completed ({progress.completed_steps.length})</span>
              </div>
              <ul className="space-y-1.5">
                {progress.completed_steps.map((step, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <CheckCircle className="h-4 w-4 text-green-500 flex-shrink-0 mt-0.5" />
                    <span className="text-foreground">{step}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Remaining Steps */}
          {progress.remaining_steps && progress.remaining_steps.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
                <Clock className="h-3 w-3" />
                <span>Remaining ({progress.remaining_steps.length})</span>
              </div>
              <ul className="space-y-1.5">
                {progress.remaining_steps.map((step, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <div className="h-4 w-4 rounded-full border-2 border-muted-foreground/50 flex-shrink-0 mt-0.5" />
                    <span className="text-muted-foreground">{step}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default AutopilotProgressTracker;
