"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useAutopilotStore, AutopilotController } from "@/stores/autopilot";
import { AlertTriangle, Play, Square, Plus, Minus } from "lucide-react";

interface CircuitBreakerAlertProps {
  autopilotController: AutopilotController;
  className?: string;
  approvalTimeoutMin?: number;
}

export function CircuitBreakerAlert({
  autopilotController,
  className,
  approvalTimeoutMin = 30,
}: CircuitBreakerAlertProps) {
  const { approveAutopilotController } = useAutopilotStore();
  const [additionalIterations, setAdditionalIterations] = React.useState(5);
  const [timeRemaining, setTimeRemaining] = React.useState<string | null>(null);

  const isWaitingApproval = autopilotController.phase === "waiting_approval";

  React.useEffect(() => {
    if (!isWaitingApproval) return;

    const startTime = autopilotController.last_iteration_at
      ? new Date(autopilotController.last_iteration_at).getTime()
      : Date.now();
    const timeoutMs = approvalTimeoutMin * 60 * 1000;

    const updateTimer = () => {
      const elapsed = Date.now() - startTime;
      const remaining = Math.max(0, timeoutMs - elapsed);
      const minutes = Math.floor(remaining / 60000);
      const seconds = Math.floor((remaining % 60000) / 1000);
      setTimeRemaining(`${minutes}:${seconds.toString().padStart(2, "0")}`);
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [isWaitingApproval, autopilotController.last_iteration_at, approvalTimeoutMin]);

  if (!isWaitingApproval) {
    return null;
  }

  const handleContinue = () => {
    approveAutopilotController(autopilotController.autopilot_controller_key, {
      continue_execution: true,
      additional_iterations: additionalIterations,
    });
  };

  const handleStop = () => {
    approveAutopilotController(autopilotController.autopilot_controller_key, {
      continue_execution: false,
    });
  };

  return (
    <div
      className={cn(
        "rounded-lg border-2 border-orange-500 bg-orange-50 dark:bg-orange-950/20 p-4",
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2 text-orange-600 dark:text-orange-400">
          <AlertTriangle className="h-5 w-5" />
          <span className="font-semibold">Circuit Breaker Triggered</span>
        </div>
        {timeRemaining && (
          <span className="text-sm text-muted-foreground">
            Timeout: {timeRemaining}
          </span>
        )}
      </div>

      {/* Reason */}
      <div className="mb-4">
        <p className="text-sm text-muted-foreground mb-1">Reason:</p>
        <p className="text-sm font-medium">
          {autopilotController.circuit_breaker.reason || "Unknown reason"}
        </p>
      </div>

      {/* Additional Iterations Selector */}
      <div className="mb-4 flex items-center gap-3">
        <span className="text-sm text-muted-foreground">
          Add iterations:
        </span>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            className="h-7 w-7 p-0"
            onClick={() => setAdditionalIterations(Math.max(0, additionalIterations - 1))}
          >
            <Minus className="h-3 w-3" />
          </Button>
          <span className="w-8 text-center font-medium">
            {additionalIterations}
          </span>
          <Button
            size="sm"
            variant="outline"
            className="h-7 w-7 p-0"
            onClick={() => setAdditionalIterations(additionalIterations + 1)}
          >
            <Plus className="h-3 w-3" />
          </Button>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <Button
          onClick={handleContinue}
          className="flex-1 bg-green-600 hover:bg-green-700"
        >
          <Play className="h-4 w-4 mr-2" />
          Continue Execution
        </Button>
        <Button
          onClick={handleStop}
          variant="destructive"
          className="flex-1"
        >
          <Square className="h-4 w-4 mr-2" />
          Stop Task
        </Button>
      </div>
    </div>
  );
}

export default CircuitBreakerAlert;
