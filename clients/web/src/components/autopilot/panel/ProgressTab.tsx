"use client";

import * as React from "react";
import { Progress } from "@/components/ui/progress";
import { CheckCircle, Clock, ListChecks } from "lucide-react";
import type { AutopilotThinking } from "@/stores/autopilot";

interface ProgressTabProps {
  thinking: AutopilotThinking | null;
}

export function ProgressTab({ thinking }: ProgressTabProps) {
  const progress = thinking?.progress;

  if (!progress) {
    return (
      <div className="flex flex-col items-center justify-center py-6 text-muted-foreground">
        <ListChecks className="h-6 w-6 mb-2 opacity-50" />
        <span className="text-xs">No progress data available</span>
      </div>
    );
  }

  return (
    <div className="space-y-3 p-3">
      {/* Progress Summary & Percent */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium">{progress.summary || "Task Progress"}</span>
          {(progress.percent ?? 0) > 0 && (
            <span className="text-sm font-medium">{progress.percent}%</span>
          )}
        </div>
        {(progress.percent ?? 0) > 0 && <Progress value={progress.percent} className="h-2" />}
      </div>

      {/* Steps in two columns */}
      <div className="grid grid-cols-2 gap-3">
        {/* Completed Steps */}
        {progress.completed_steps && progress.completed_steps.length > 0 && (
          <div>
            <div className="text-xs text-muted-foreground mb-1.5 flex items-center gap-1">
              <CheckCircle className="h-3 w-3 text-green-500" />
              <span>Completed ({progress.completed_steps.length})</span>
            </div>
            <ul className="space-y-0.5">
              {progress.completed_steps.map((step, i) => (
                <li key={i} className="flex items-start gap-1.5 text-xs">
                  <CheckCircle className="h-3 w-3 text-green-500 flex-shrink-0 mt-0.5" />
                  <span className="line-clamp-2">{step}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Remaining Steps */}
        {progress.remaining_steps && progress.remaining_steps.length > 0 && (
          <div>
            <div className="text-xs text-muted-foreground mb-1.5 flex items-center gap-1">
              <Clock className="h-3 w-3" />
              <span>Remaining ({progress.remaining_steps.length})</span>
            </div>
            <ul className="space-y-0.5">
              {progress.remaining_steps.map((step, i) => (
                <li key={i} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                  <div className="h-3 w-3 rounded-full border border-muted-foreground flex-shrink-0 mt-0.5" />
                  <span className="line-clamp-2">{step}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}

export default ProgressTab;
