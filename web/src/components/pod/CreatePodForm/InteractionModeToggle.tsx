"use client";

import React from "react";
import { useTranslations } from "next-intl";

interface InteractionModeToggleProps {
  supportedModes: string[];
  interactionMode: string;
  onModeChange: (mode: "pty" | "acp") => void;
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
        {supportedModes.includes("pty") && (
          <button
            type="button"
            onClick={() => onModeChange("pty")}
            className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
              interactionMode === "pty"
                ? "border-primary bg-primary/10 text-primary font-medium"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            {t("ide.createPod.modePty")}
          </button>
        )}
        {supportedModes.includes("acp") && (
          <button
            type="button"
            onClick={() => onModeChange("acp")}
            className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
              interactionMode === "acp"
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
