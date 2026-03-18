"use client";

import React, { useEffect } from "react";
import { useAutopilotStore } from "@/stores/autopilot";
import { useIDEStore } from "@/stores/ide";
import {
  CircuitBreakerAlert,
  TakeoverBanner,
  AutopilotStatusBar,
} from "@/components/autopilot";

interface AutopilotOverlayProps {
  podKey: string;
}

/**
 * Renders autopilot UI elements (TakeoverBanner, CircuitBreakerAlert,
 * AutopilotStatusBar) and manages the auto-open BottomPanel effect.
 * Extracted from TerminalPane for SRP.
 */
export function AutopilotOverlay({ podKey }: AutopilotOverlayProps) {
  const autopilotController = useAutopilotStore((state) =>
    state.getAutopilotControllerByPodKey(podKey)
  );
  const setBottomPanelOpen = useIDEStore((s) => s.setBottomPanelOpen);
  const setBottomPanelTab = useIDEStore((s) => s.setBottomPanelTab);

  const autopilotControllerKey = autopilotController?.autopilot_controller_key;
  const thinking = useAutopilotStore((state) =>
    autopilotControllerKey
      ? (state.thinking[autopilotControllerKey] ?? null)
      : null
  );

  // Auto-open BottomPanel Autopilot tab when help is needed
  useEffect(() => {
    if (
      thinking?.decision_type === "need_help" ||
      thinking?.decision_type === "NEED_HUMAN_HELP" ||
      autopilotController?.phase === "waiting_approval"
    ) {
      setBottomPanelTab("autopilot");
      setBottomPanelOpen(true);
    }
  }, [thinking?.decision_type, autopilotController?.phase, setBottomPanelTab, setBottomPanelOpen]);

  if (!autopilotController) return null;

  return (
    <>
      <TakeoverBanner autopilotController={autopilotController} className="rounded-none" />
      <CircuitBreakerAlert autopilotController={autopilotController} className="mx-2 mt-2 rounded-md" />
      <AutopilotStatusBar
        autopilotController={autopilotController}
        onTogglePanel={() => {
          setBottomPanelTab("autopilot");
          setBottomPanelOpen(true);
        }}
      />
    </>
  );
}
