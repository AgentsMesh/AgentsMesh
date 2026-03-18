"use client";

import { MutableRefObject, useEffect, useState } from "react";
import { usePodTitle } from "@/hooks/usePodTitle";
import { CreateAutopilotControllerModal } from "@/components/autopilot";

interface AutopilotStartButtonProps {
  podKey: string;
  triggerRef: MutableRefObject<(() => void) | null>;
}

/**
 * Manages the CreateAutopilotControllerModal and exposes a trigger
 * function via triggerRef so the parent can open the modal.
 * Extracted from TerminalPane for SRP.
 */
export function AutopilotStartButton({ podKey, triggerRef }: AutopilotStartButtonProps) {
  const [showModal, setShowModal] = useState(false);
  const podTitle = usePodTitle(podKey);

  // Expose trigger to parent via ref
  useEffect(() => {
    triggerRef.current = () => setShowModal(true);
    return () => {
      triggerRef.current = null;
    };
  }, [triggerRef]);

  return (
    <CreateAutopilotControllerModal
      open={showModal}
      onClose={() => setShowModal(false)}
      podKey={podKey}
      podTitle={podTitle}
    />
  );
}
