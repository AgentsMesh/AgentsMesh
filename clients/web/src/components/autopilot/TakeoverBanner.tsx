"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useAutopilotStore, AutopilotController } from "@/stores/autopilot";
import { Gamepad2, RotateCcw } from "lucide-react";

interface TakeoverBannerProps {
  autopilotController: AutopilotController;
  className?: string;
}

export function TakeoverBanner({ autopilotController, className }: TakeoverBannerProps) {
  const { handbackAutopilotController } = useAutopilotStore();

  if (!autopilotController.user_takeover && autopilotController.phase !== "user_takeover") {
    return null;
  }

  return (
    <div
      className={cn(
        "flex items-center justify-between px-4 py-2 bg-purple-600 text-white rounded-t-lg",
        className
      )}
    >
      <div className="flex items-center gap-2">
        <Gamepad2 className="h-4 w-4" />
        <span className="text-sm font-medium">
          You have taken control
        </span>
      </div>
      <Button
        size="sm"
        variant="secondary"
        className="h-7 text-xs"
        onClick={() => handbackAutopilotController(autopilotController.autopilot_controller_key)}
      >
        <RotateCcw className="h-3 w-3 mr-1" />
        Handback to Autopilot
      </Button>
    </div>
  );
}

export default TakeoverBanner;
