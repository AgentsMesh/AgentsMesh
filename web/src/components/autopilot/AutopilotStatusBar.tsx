"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useAutopilotStore, AutopilotController, AutopilotThinking } from "@/stores/autopilot";
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
  ExternalLink,
  Brain,
} from "lucide-react";

interface AutopilotStatusBarProps {
  autopilotController: AutopilotController;
  className?: string;
  onTogglePanel?: () => void;
}

// Phase display configuration with pulse animation states
const phaseConfig: Record<
  AutopilotController["phase"],
  { label: string; bgColor: string; textColor: string; icon: React.ReactNode; animate?: boolean }
> = {
  initializing: {
    label: "Initializing",
    bgColor: "bg-blue-500/10",
    textColor: "text-blue-500",
    icon: <Loader2 className="h-3 w-3 animate-spin" />,
    animate: true,
  },
  running: {
    label: "Running",
    bgColor: "bg-green-500/10",
    textColor: "text-green-500",
    icon: <Play className="h-3 w-3" />,
    animate: true,
  },
  paused: {
    label: "Paused",
    bgColor: "bg-yellow-500/10",
    textColor: "text-yellow-500",
    icon: <Pause className="h-3 w-3" />,
  },
  user_takeover: {
    label: "User Control",
    bgColor: "bg-purple-500/10",
    textColor: "text-purple-500",
    icon: <Hand className="h-3 w-3" />,
  },
  waiting_approval: {
    label: "Waiting Approval",
    bgColor: "bg-orange-500/10",
    textColor: "text-orange-500",
    icon: <AlertTriangle className="h-3 w-3" />,
    animate: true,
  },
  completed: {
    label: "Completed",
    bgColor: "bg-green-600/10",
    textColor: "text-green-600",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  failed: {
    label: "Failed",
    bgColor: "bg-red-500/10",
    textColor: "text-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
  stopped: {
    label: "Stopped",
    bgColor: "bg-gray-500/10",
    textColor: "text-gray-500",
    icon: <Square className="h-3 w-3" />,
  },
  max_iterations: {
    label: "Max Iterations",
    bgColor: "bg-orange-600/10",
    textColor: "text-orange-600",
    icon: <Clock className="h-3 w-3" />,
  },
};

// Truncate reasoning text for status bar display
function truncateReasoning(text: string | undefined, maxLength: number = 60): string {
  if (!text) return "Waiting for Control Agent...";
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength) + "...";
}

export function AutopilotStatusBar({
  autopilotController,
  className,
  onTogglePanel,
}: AutopilotStatusBarProps) {
  const pauseAutopilotController = useAutopilotStore((s) => s.pauseAutopilotController);
  const resumeAutopilotController = useAutopilotStore((s) => s.resumeAutopilotController);
  const stopAutopilotController = useAutopilotStore((s) => s.stopAutopilotController);
  const takeoverAutopilotController = useAutopilotStore((s) => s.takeoverAutopilotController);
  const handbackAutopilotController = useAutopilotStore((s) => s.handbackAutopilotController);

  // Reactive thinking selector — re-renders when this controller's thinking changes
  const thinking: AutopilotThinking | null = useAutopilotStore(
    (s) => s.thinking[autopilotController.autopilot_controller_key] ?? null
  );

  const phaseInfo = phaseConfig[autopilotController.phase];
  const progress =
    (autopilotController.current_iteration / autopilotController.max_iterations) * 100;

  const isActive = ["initializing", "running", "paused", "user_takeover", "waiting_approval"].includes(
    autopilotController.phase
  );
  const reasoningText = thinking?.reasoning;

  // Is needing help?
  // Backend sends NEED_HUMAN_HELP, frontend expects need_help
  const needsHelp =
    thinking?.decision_type === "need_help" ||
    thinking?.decision_type === "NEED_HUMAN_HELP" ||
    autopilotController.phase === "waiting_approval";

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 border-b",
        needsHelp ? "bg-orange-500/5 border-orange-500/30" : phaseInfo.bgColor,
        className
      )}
    >
      {/* Status Indicator with Pulse Animation */}
      <div className="flex items-center gap-2 min-w-0 flex-shrink-0">
        {/* Animated status dot */}
        <div className="relative flex items-center justify-center">
          {phaseInfo.animate && (
            <span
              className={cn(
                "absolute inline-flex h-4 w-4 rounded-full opacity-75 animate-ping",
                needsHelp ? "bg-orange-400" : phaseInfo.textColor.replace("text-", "bg-")
              )}
            />
          )}
          <span
            className={cn(
              "relative inline-flex items-center justify-center h-4 w-4 rounded-full",
              needsHelp ? "bg-orange-500 text-white" : `${phaseInfo.textColor.replace("text-", "bg-")} text-white`
            )}
          >
            {needsHelp ? <AlertTriangle className="h-2.5 w-2.5" /> : phaseInfo.icon}
          </span>
        </div>

        {/* Phase Label */}
        <span
          className={cn(
            "text-xs font-medium whitespace-nowrap",
            needsHelp ? "text-orange-500" : phaseInfo.textColor
          )}
        >
          {needsHelp ? "Need Help" : phaseInfo.label}
        </span>
      </div>

      {/* Progress Bar */}
      <div className="flex items-center gap-1.5 flex-shrink-0">
        <div className="w-16 md:w-24">
          <Progress
            value={progress}
            className={cn(
              "h-1.5",
              needsHelp && "[&>div]:bg-orange-500"
            )}
          />
        </div>
        <span className="text-[10px] text-muted-foreground whitespace-nowrap">
          {autopilotController.current_iteration}/{autopilotController.max_iterations}
        </span>
      </div>

      {/* Reasoning Summary (with scrolling effect on hover) */}
      <div className="hidden md:flex items-center gap-1.5 min-w-0 flex-1">
        <Brain className="h-3 w-3 text-muted-foreground flex-shrink-0" />
        <span
          className="text-xs text-muted-foreground truncate"
          title={reasoningText || "Waiting for Control Agent..."}
        >
          {truncateReasoning(reasoningText)}
        </span>
      </div>

      {/* Control Buttons */}
      <div className="flex items-center gap-0.5 flex-shrink-0 ml-auto">
        {/* View Details Button - opens BottomPanel */}
        <Button
          size="sm"
          variant="ghost"
          className="h-6 w-6 p-0 hover:bg-background/50"
          onClick={onTogglePanel}
          title="View Details"
        >
          <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
        </Button>

        {/* Pause/Resume Button */}
        {autopilotController.phase === "running" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 hover:bg-yellow-500/20"
            onClick={() => pauseAutopilotController(autopilotController.autopilot_controller_key)}
            title="Pause"
          >
            <Pause className="h-3.5 w-3.5 text-yellow-500" />
          </Button>
        )}

        {autopilotController.phase === "paused" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 hover:bg-green-500/20"
            onClick={() => resumeAutopilotController(autopilotController.autopilot_controller_key)}
            title="Resume"
          >
            <Play className="h-3.5 w-3.5 text-green-500" />
          </Button>
        )}

        {/* Takeover/Handback Button */}
        {autopilotController.phase === "running" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 hover:bg-purple-500/20"
            onClick={() => takeoverAutopilotController(autopilotController.autopilot_controller_key)}
            title="Takeover Control"
          >
            <Hand className="h-3.5 w-3.5 text-purple-500" />
          </Button>
        )}

        {autopilotController.phase === "user_takeover" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 hover:bg-purple-500/20"
            onClick={() => handbackAutopilotController(autopilotController.autopilot_controller_key)}
            title="Handback to Autopilot"
          >
            <RotateCcw className="h-3.5 w-3.5 text-purple-500" />
          </Button>
        )}

        {/* Stop Button */}
        {isActive && autopilotController.phase !== "waiting_approval" && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 hover:bg-red-500/20"
            onClick={() => stopAutopilotController(autopilotController.autopilot_controller_key)}
            title="Stop Autopilot"
          >
            <Square className="h-3.5 w-3.5 text-red-500" />
          </Button>
        )}
      </div>
    </div>
  );
}

export default AutopilotStatusBar;
