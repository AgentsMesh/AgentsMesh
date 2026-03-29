"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { POD_MODE_PTY, POD_MODE_ACP } from "@/lib/pod-modes";
import type { PodMode } from "@/lib/pod-modes";

interface InteractionModeToggleProps {
  supportedModes: string[];
  interactionMode: string;
  onModeChange: (mode: PodMode) => void;
}

export function InteractionModeToggle({
  supportedModes,
  interactionMode,
  onModeChange,
}: InteractionModeToggleProps) {
  const t = useTranslations();

  if (supportedModes.length <= 1) return null;

  return (
    <div>
      <label className="block text-sm font-medium mb-1.5">
        {t("ide.createPod.interactionMode")}
      </label>
      <div className="flex gap-2">
        {supportedModes.includes(POD_MODE_PTY) && (
          <button
            type="button"
            onClick={() => onModeChange(POD_MODE_PTY)}
            className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
              interactionMode === POD_MODE_PTY
                ? "border-primary bg-primary/10 text-primary font-medium"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            {t("ide.createPod.modePty")}
          </button>
        )}
        {supportedModes.includes(POD_MODE_ACP) && (
          <button
            type="button"
            onClick={() => onModeChange(POD_MODE_ACP)}
            className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
              interactionMode === POD_MODE_ACP
                ? "border-primary bg-primary/10 text-primary font-medium"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            {t("ide.createPod.modeAcp")}
          </button>
        )}
      </div>
    </div>
  );
}
