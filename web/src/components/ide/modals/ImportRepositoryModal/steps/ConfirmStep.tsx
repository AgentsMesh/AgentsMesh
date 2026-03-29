"use client";

import { Input } from "@/components/ui/input";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";
import type { StepProps } from "../types";

/**
 * Step 4: Confirm - Review and confirm import settings
 */
export function ConfirmStep({ state, actions, t }: StepProps) {
  const {
    manualProviderType,
    manualCloneURL,
    manualName,
    manualSlug,
    manualDefaultBranch,
    ticketPrefix,
    visibility,
  } = state;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <button
          onClick={actions.goBack}
          className="text-muted-foreground hover:text-foreground"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 19l-7-7 7-7"
            />
          </svg>
        </button>
        <span className="text-sm text-muted-foreground">
          {t("repositories.modal.confirmImport")}
        </span>
      </div>

      <div className="p-4 border border-border rounded-lg bg-muted/50">
        <div className="flex items-center gap-3 mb-3">
          <GitProviderIcon provider={manualProviderType} />
          <div>
            <div className="font-medium">{manualName}</div>
            <div className="text-sm text-muted-foreground">{manualSlug}</div>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div className="text-muted-foreground">{t("repositories.modal.cloneUrl")}</div>
          <div className="truncate">{manualCloneURL}</div>
          <div className="text-muted-foreground">{t("repositories.modal.branch")}</div>
          <div>{manualDefaultBranch}</div>
          <div className="text-muted-foreground">{t("repositories.modal.provider")}</div>
          <div className="capitalize">{manualProviderType}</div>
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">
          {t("repositories.modal.ticketPrefixOptional")}
        </label>
        <Input
          value={ticketPrefix}
          onChange={(e) => actions.setTicketPrefix(e.target.value)}
          placeholder="PROJ"
        />
        <p className="text-xs text-muted-foreground mt-1">
          {t("repositories.modal.ticketPrefixHint")}
        </p>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">
          {t("repositories.modal.visibility")}
        </label>
        <div className="flex gap-4">
          <label className="flex items-center gap-2">
            <input
              type="radio"
              checked={visibility === "organization"}
              onChange={() => actions.setVisibility("organization")}
              className="w-4 h-4"
            />
            <span className="text-sm">{t("repositories.modal.organization")}</span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="radio"
              checked={visibility === "private"}
              onChange={() => actions.setVisibility("private")}
              className="w-4 h-4"
            />
            <span className="text-sm">{t("repositories.modal.privateOnly")}</span>
          </label>
        </div>
      </div>
    </div>
  );
}

export default ConfirmStep;
