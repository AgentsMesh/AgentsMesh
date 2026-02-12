"use client";

import Link from "next/link";
import { CenteredSpinner } from "@/components/ui/spinner";
import { GitProviderIcon } from "@/components/icons/GitProviderIcon";
import type { StepProps } from "../types";

/**
 * Step 1: Select Source - Choose provider or manual entry
 */
export function SourceStep({ state, actions, t }: StepProps) {
  const { providers, loadingProviders } = state;

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {t("repositories.modal.selectSourceHint")}
      </p>

      {loadingProviders ? (
        <CenteredSpinner className="py-8" />
      ) : (
        <>
          <div className="space-y-2">
            <p className="text-sm font-medium">{t("repositories.modal.yourConnections")}</p>
            {providers.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">
                {t("repositories.modal.noConnections")}{" "}
                <Link
                  href="/settings/git"
                  className="text-primary hover:underline"
                >
                  {t("repositories.modal.addOne")}
                </Link>{" "}
                {t("repositories.modal.toBrowse")}
              </p>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                {providers.map((provider) => (
                  <button
                    key={provider.id}
                    onClick={() => actions.selectProvider(provider)}
                    className="flex items-center gap-3 p-4 border border-border rounded-lg hover:bg-muted/50 text-left"
                  >
                    <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                      <GitProviderIcon provider={provider.provider_type} />
                    </div>
                    <div>
                      <div className="font-medium">{provider.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {provider.base_url}
                      </div>
                      {provider.has_identity && (
                        <div className="text-xs text-green-600 dark:text-green-400">
                          OAuth
                        </div>
                      )}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-border"></div>
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">
                {t("repositories.modal.or")}
              </span>
            </div>
          </div>

          <button
            onClick={() => actions.setStep("manual")}
            className="w-full flex items-center gap-3 p-4 border border-dashed border-border rounded-lg hover:bg-muted/50"
          >
            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
              <svg
                className="w-5 h-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                />
              </svg>
            </div>
            <div className="text-left">
              <div className="font-medium">{t("repositories.modal.enterManually")}</div>
              <div className="text-xs text-muted-foreground">
                {t("repositories.modal.enterManuallyHint")}
              </div>
            </div>
          </button>
        </>
      )}
    </div>
  );
}

export default SourceStep;
