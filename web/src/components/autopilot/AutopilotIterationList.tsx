"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { useAutopilotStore, AutopilotIteration } from "@/stores/autopilot";

// Stable empty array to avoid infinite re-render when key is missing from iterations map.
const EMPTY_ITERATIONS: AutopilotIteration[] = [];
import {
  CheckCircle,
  XCircle,
  Play,
  Send,
  FileText,
  Loader2,
  ChevronDown,
  ChevronRight,
} from "lucide-react";

interface AutopilotIterationListProps {
  autopilotControllerKey: string;
  className?: string;
  maxItems?: number;
}

// Iteration phase display configuration
const iterationPhaseConfig: Record<
  string,
  { label: string; color: string; icon: React.ReactNode }
> = {
  initial_prompt: {
    label: "Initial Prompt",
    color: "bg-blue-500",
    icon: <Send className="h-3 w-3" />,
  },
  started: {
    label: "Started",
    color: "bg-blue-400",
    icon: <Play className="h-3 w-3" />,
  },
  control_running: {
    label: "Control Running",
    color: "bg-yellow-500",
    icon: <Loader2 className="h-3 w-3 animate-spin" />,
  },
  action_sent: {
    label: "Action Sent",
    color: "bg-green-500",
    icon: <Send className="h-3 w-3" />,
  },
  completed: {
    label: "Completed",
    color: "bg-green-600",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  error: {
    label: "Error",
    color: "bg-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
};

function IterationItem({ iteration }: { iteration: AutopilotIteration }) {
  const [expanded, setExpanded] = React.useState(false);
  const phaseInfo = iterationPhaseConfig[iteration.phase] || {
    label: iteration.phase,
    color: "bg-gray-500",
    icon: <FileText className="h-3 w-3" />,
  };

  const hasDetails =
    iteration.summary ||
    (iteration.files_changed && iteration.files_changed.length > 0);

  return (
    <div className="border-b last:border-b-0 py-2">
      <div
        className={cn(
          "flex items-center gap-2",
          hasDetails && "cursor-pointer"
        )}
        onClick={() => hasDetails && setExpanded(!expanded)}
      >
        {hasDetails ? (
          expanded ? (
            <ChevronDown className="h-3 w-3 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-3 w-3 text-muted-foreground" />
          )
        ) : (
          <div className="w-3" />
        )}

        <Badge
          variant="outline"
          className={cn(
            "flex items-center gap-1 text-xs",
            phaseInfo.color,
            "text-white"
          )}
        >
          {phaseInfo.icon}
          <span>{phaseInfo.label}</span>
        </Badge>

        <span className="text-xs text-muted-foreground">
          #{iteration.iteration}
        </span>

        {iteration.duration_ms && (
          <span className="text-xs text-muted-foreground ml-auto">
            {(iteration.duration_ms / 1000).toFixed(1)}s
          </span>
        )}
      </div>

      {expanded && hasDetails && (
        <div className="mt-2 ml-5 pl-3 border-l-2 border-muted">
          {iteration.summary && (
            <p className="text-sm text-muted-foreground mb-2">
              {iteration.summary}
            </p>
          )}

          {iteration.files_changed && iteration.files_changed.length > 0 && (
            <div className="text-xs">
              <span className="text-muted-foreground">Files changed: </span>
              <span className="font-mono">
                {iteration.files_changed.join(", ")}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function AutopilotIterationList({
  autopilotControllerKey,
  className,
  maxItems,
}: AutopilotIterationListProps) {
  const controllerIterations = useAutopilotStore(
    (s) => s.iterations[autopilotControllerKey] ?? EMPTY_ITERATIONS
  );
  const fetchIterations = useAutopilotStore((s) => s.fetchIterations);

  React.useEffect(() => {
    if (autopilotControllerKey) {
      fetchIterations(autopilotControllerKey);
    }
  }, [autopilotControllerKey, fetchIterations]);

  const displayIterations = maxItems
    ? controllerIterations.slice(-maxItems)
    : controllerIterations;

  if (displayIterations.length === 0) {
    return (
      <div className={cn("text-sm text-muted-foreground p-4", className)}>
        No iterations yet
      </div>
    );
  }

  return (
    <div className={cn("rounded-lg border bg-card", className)}>
      <div className="px-4 py-2 border-b bg-muted/50">
        <h3 className="text-sm font-medium">Iteration History</h3>
      </div>
      <div className="max-h-64 overflow-y-auto px-4">
        {displayIterations.map((iteration) => (
          <IterationItem key={iteration.id} iteration={iteration} />
        ))}
      </div>
    </div>
  );
}

export default AutopilotIterationList;
