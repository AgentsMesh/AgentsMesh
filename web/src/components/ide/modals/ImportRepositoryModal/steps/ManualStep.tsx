"use client";

import { Input } from "@/components/ui/input";
import type { StepProps } from "../types";

/**
 * Step 3: Manual Entry - Enter repository details manually
 */
export function ManualStep({ state, actions, t }: StepProps) {
  const {
    manualProviderType,
    manualBaseURL,
    manualCloneURL,
    manualName,
    manualSlug,
    manualDefaultBranch,
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
          {t("repositories.modal.manualEntry")}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">
            {t("repositories.modal.providerType")}
          </label>
          <select
            className="w-full px-3 py-2 border border-border rounded-md bg-background"
            value={manualProviderType}
            onChange={(e) => actions.setManualProviderType(e.target.value)}
          >
            <option value="github">GitHub</option>
            <option value="gitlab">GitLab</option>
            <option value="gitee">Gitee</option>
            <option value="generic">{t("repositories.modal.genericGit")}</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">
            {t("repositories.modal.baseUrl")}
          </label>
          <Input
            value={manualBaseURL}
            onChange={(e) => actions.setManualBaseURL(e.target.value)}
            placeholder="https://github.com"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">
          {t("repositories.modal.cloneUrl")} *
        </label>
        <Input
          value={manualCloneURL}
          onChange={(e) => actions.setManualCloneURL(e.target.value)}
          placeholder="https://github.com/org/repo.git"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">
            {t("repositories.modal.repoName")} *
          </label>
          <Input
            value={manualName}
            onChange={(e) => actions.setManualName(e.target.value)}
            placeholder="my-project"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">
            {t("repositories.modal.slug")} *
          </label>
          <Input
            value={manualSlug}
            onChange={(e) => actions.setManualSlug(e.target.value)}
            placeholder="org/my-project"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">
          {t("repositories.modal.defaultBranch")}
        </label>
        <Input
          value={manualDefaultBranch}
          onChange={(e) => actions.setManualDefaultBranch(e.target.value)}
          placeholder="main"
        />
      </div>
    </div>
  );
}

export default ManualStep;
