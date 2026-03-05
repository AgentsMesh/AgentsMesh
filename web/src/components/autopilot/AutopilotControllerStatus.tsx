"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useAutopilotStore, AutopilotController } from "@/stores/autopilot";
import {
  Play,
  Pause,
  Square,
  Hand,
  RotateCcw,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
} from "lucide-react";

interface AutopilotControllerStatusProps {
  autopilotController: AutopilotController;
  className?: string;
  compact?: boolean;
}

// Phase display configuration
const phaseConfig: Record<
  AutopilotController["phase"],
  { label: string; color: string; icon: React.ReactNode }
> = {
  initializing: {
    label: "Initializing",
    color: "bg-blue-500",
    icon: <Loader2 className="h-3 w-3 animate-spin" />,
  },
  running: {
    label: "Running",
    color: "bg-green-500",
    icon: <Play className="h-3 w-3" />,
  },
  paused: {
    label: "Paused",
    color: "bg-yellow-500",
    icon: <Pause className="h-3 w-3" />,
  },
  user_takeover: {
    label: "User Control",
    color: "bg-purple-500",
    icon: <Hand className="h-3 w-3" />,
  },
  waiting_approval: {
    label: "Waiting Approval",
    color: "bg-orange-500",
    icon: <AlertTriangle className="h-3 w-3" />,
  },
  completed: {
    label: "Completed",
    color: "bg-green-600",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  failed: {
    label: "Failed",
    color: "bg-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
  stopped: {
    label: "Stopped",
    color: "bg-gray-500",
    icon: <Square className="h-3 w-3" />,
  },
  max_iterations: {
    label: "Max Iterations",
    color: "bg-orange-600",
    icon: <Clock className="h-3 w-3" />,
  },
};

// Circuit breaker state display
const circuitBreakerConfig: Record<
  AutopilotController["circuit_breaker"]["state"],
  { label: string; color: string }
> = {
  closed: { label: "OK", color: "text-green-500" },
  half_open: { label: "Warning", color: "text-yellow-500" },
  open: { label: "Triggered", color: "text-red-500" },
};

export function AutopilotControllerStatus({
  autopilotController,
  className,
  compact = false,
}: AutopilotControllerStatusProps) {
  const pauseAutopilotController = useAutopilotStore((s) => s.pauseAutopilotController);
  const resumeAutopilotController = useAutopilotStore((s) => s.resumeAutopilotController);
  const stopAutopilotController = useAutopilotStore((s) => s.stopAutopilotController);
  const takeoverAutopilotController = useAutopilotStore((s) => s.takeoverAutopilotController);
  const handbackAutopilotController = useAutopilotStore((s) => s.handbackAutopilotController);

  const phaseInfo = phaseConfig[autopilotController.phase];
  const cbInfo = circuitBreakerConfig[autopilotController.circuit_breaker.state];
  const progress =
    (autopilotController.current_iteration / autopilotController.max_iterations) * 100;

  const isActive = ["initializing", "running", "paused", "user_takeover", "waiting_approval"].includes(
    autopilotController.phase
  );

  if (compact) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <Badge
          variant="outline"
          className={cn("flex items-center gap-1", phaseInfo.color, "text-white")}
        >
          {phaseInfo.icon}
          <span>{phaseInfo.label}</span>
        </Badge>
        <span className="text-xs text-muted-foreground">
          {autopilotController.current_iteration}/{autopilotController.max_iterations}
        </span>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "rounded-lg border bg-card p-4 shadow-sm",
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className={cn(
              "flex items-center gap-1",
              phaseInfo.color,
              "text-white"
            )}
          >
            {phaseInfo.icon}
            <span>{phaseInfo.label}</span>
          </Badge>
          <span className="text-sm text-muted-foreground">
            Autopilot Mode
          </span>
        </div>

        {/* Circuit Breaker Status */}
        <div className="flex items-center gap-1 text-xs">
          <span className="text-muted-foreground">Circuit:</span>
          <span className={cn("font-medium", cbInfo.color)}>
            {cbInfo.label}
          </span>
        </div>
      </div>

      {/* Progress */}
      <div className="mb-3">
        <div className="flex justify-between text-xs mb-1">
          <span className="text-muted-foreground">Iteration Progress</span>
          <span className="font-medium">
            {autopilotController.current_iteration} / {autopilotController.max_iterations}
          </span>
        </div>
        <Progress value={progress} className="h-1.5" />
      </div>

      {/* Actions */}
      {isActive && (
        <div className="flex items-center gap-2 mt-3 pt-3 border-t">
          {autopilotController.phase === "running" && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => pauseAutopilotController(autopilotController.autopilot_controller_key)}
              >
                <Pause className="h-3 w-3 mr-1" />
                Pause
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => takeoverAutopilotController(autopilotController.autopilot_controller_key)}
              >
                <Hand className="h-3 w-3 mr-1" />
                Takeover
              </Button>
            </>
          )}

          {autopilotController.phase === "paused" && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => resumeAutopilotController(autopilotController.autopilot_controller_key)}
            >
              <Play className="h-3 w-3 mr-1" />
              Resume
            </Button>
          )}

          {autopilotController.phase === "user_takeover" && (
            <Button
              size="sm"
              variant="default"
              onClick={() => handbackAutopilotController(autopilotController.autopilot_controller_key)}
            >
              <RotateCcw className="h-3 w-3 mr-1" />
              Handback to Autopilot
            </Button>
          )}

          {isActive && autopilotController.phase !== "waiting_approval" && (
            <Button
              size="sm"
              variant="destructive"
              onClick={() => stopAutopilotController(autopilotController.autopilot_controller_key)}
            >
              <Square className="h-3 w-3 mr-1" />
              Stop
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

export default AutopilotControllerStatus;
