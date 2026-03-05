"use client";

import * as React from "react";
import { Button } from "@/components/ui/button";
import { Play, Pause, Square, Hand, RotateCcw } from "lucide-react";
import { useAutopilotStore, type AutopilotController } from "@/stores/autopilot";

interface ControlButtonsProps {
  autopilotController: AutopilotController;
}

export function ControlButtons({ autopilotController }: ControlButtonsProps) {
  const pauseAutopilotController = useAutopilotStore((s) => s.pauseAutopilotController);
  const resumeAutopilotController = useAutopilotStore((s) => s.resumeAutopilotController);
  const stopAutopilotController = useAutopilotStore((s) => s.stopAutopilotController);
  const takeoverAutopilotController = useAutopilotStore((s) => s.takeoverAutopilotController);
  const handbackAutopilotController = useAutopilotStore((s) => s.handbackAutopilotController);

  const isActive = ["initializing", "running", "paused", "user_takeover", "waiting_approval"].includes(
    autopilotController.phase
  );

  return (
    <div className="flex items-center gap-1">
      {/* Pause/Resume Button */}
      {autopilotController.phase === "running" && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={() => pauseAutopilotController(autopilotController.autopilot_controller_key)}
        >
          <Pause className="h-3 w-3 mr-1" />
          Pause
        </Button>
      )}

      {autopilotController.phase === "paused" && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={() => resumeAutopilotController(autopilotController.autopilot_controller_key)}
        >
          <Play className="h-3 w-3 mr-1" />
          Resume
        </Button>
      )}

      {/* Takeover/Handback Button */}
      {autopilotController.phase === "running" && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={() => takeoverAutopilotController(autopilotController.autopilot_controller_key)}
        >
          <Hand className="h-3 w-3 mr-1" />
          Takeover
        </Button>
      )}

      {autopilotController.phase === "user_takeover" && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={() => handbackAutopilotController(autopilotController.autopilot_controller_key)}
        >
          <RotateCcw className="h-3 w-3 mr-1" />
          Handback
        </Button>
      )}

      {/* Stop Button */}
      {isActive && autopilotController.phase !== "waiting_approval" && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs text-red-500 hover:text-red-600 hover:bg-red-500/10"
          onClick={() => stopAutopilotController(autopilotController.autopilot_controller_key)}
        >
          <Square className="h-3 w-3 mr-1" />
          Stop
        </Button>
      )}
    </div>
  );
}

export default ControlButtons;
