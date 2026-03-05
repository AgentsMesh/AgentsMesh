"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { History, FileText, ChevronDown, ChevronRight } from "lucide-react";
import { useAutopilotStore, type AutopilotIteration } from "@/stores/autopilot";
import { iterationPhaseConfig } from "./config";

// Stable empty array to avoid infinite re-render when key is missing from iterations map.
// Inline `|| []` creates a new reference on every getSnapshot → useSyncExternalStore loop.
const EMPTY_ITERATIONS: AutopilotIteration[] = [];

interface HistoryTabProps {
  autopilotControllerKey: string;
}

// Iteration Item component
function IterationItem({ iteration }: { iteration: AutopilotIteration }) {
  const [expanded, setExpanded] = React.useState(false);
  const phaseInfo = iterationPhaseConfig[iteration.phase] || {
    label: iteration.phase,
    color: "bg-gray-500",
    icon: <FileText className="h-3 w-3" />,
  };

  const hasDetails =
    iteration.summary || (iteration.files_changed && iteration.files_changed.length > 0);

  return (
    <div className="border-b border-border/50 last:border-b-0 py-1.5">
      <div
        className={cn(
          "flex items-center gap-2",
          hasDetails && "cursor-pointer hover:bg-muted/50 -mx-2 px-2 rounded"
        )}
        onClick={() => hasDetails && setExpanded(!expanded)}
      >
        {hasDetails ? (
          expanded ? (
            <ChevronDown className="h-3 w-3 text-muted-foreground flex-shrink-0" />
          ) : (
            <ChevronRight className="h-3 w-3 text-muted-foreground flex-shrink-0" />
          )
        ) : (
          <div className="w-3 flex-shrink-0" />
        )}

        <Badge
          variant="outline"
          className={cn("flex items-center gap-1 text-[10px] h-5", phaseInfo.color, "text-white")}
        >
          {phaseInfo.icon}
          <span>{phaseInfo.label}</span>
        </Badge>

        <span className="text-xs text-muted-foreground">#{iteration.iteration}</span>

        {iteration.duration_ms && (
          <span className="text-[10px] text-muted-foreground ml-auto">
            {(iteration.duration_ms / 1000).toFixed(1)}s
          </span>
        )}
      </div>

      {expanded && hasDetails && (
        <div className="mt-1.5 ml-5 pl-2 border-l-2 border-muted">
          {iteration.summary && (
            <p className="text-xs text-muted-foreground mb-1">{iteration.summary}</p>
          )}

          {iteration.files_changed && iteration.files_changed.length > 0 && (
            <div className="text-[10px]">
              <span className="text-muted-foreground">Files: </span>
              <span className="font-mono">{iteration.files_changed.join(", ")}</span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function HistoryTab({ autopilotControllerKey }: HistoryTabProps) {
  const controllerIterations = useAutopilotStore(
    (s) => s.iterations[autopilotControllerKey] ?? EMPTY_ITERATIONS
  );
  const fetchIterations = useAutopilotStore((s) => s.fetchIterations);

  React.useEffect(() => {
    if (autopilotControllerKey) {
      fetchIterations(autopilotControllerKey);
    }
  }, [autopilotControllerKey, fetchIterations]);

  if (controllerIterations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-6 text-muted-foreground">
        <History className="h-6 w-6 mb-2 opacity-50" />
        <span className="text-xs">No iterations yet</span>
      </div>
    );
  }

  // Show iterations in reverse order (most recent first)
  const displayIterations = [...controllerIterations].reverse();

  return (
    <div className="max-h-40 overflow-y-auto px-3 py-2">
      {displayIterations.map((iteration) => (
        <IterationItem key={iteration.id || iteration.iteration} iteration={iteration} />
      ))}
    </div>
  );
}

export default HistoryTab;
