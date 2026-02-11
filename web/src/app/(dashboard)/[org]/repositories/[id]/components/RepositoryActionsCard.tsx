"use client";

import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";

interface RepositoryActionsCardProps {
  onSetupWebhook: () => void;
}

export function RepositoryActionsCard({ onSetupWebhook }: RepositoryActionsCardProps) {
  const t = useTranslations();

  return (
    <div className="border border-border rounded-lg p-6 md:col-span-2">
      <h3 className="font-semibold mb-4">{t("repositories.detail.actions")}</h3>
      <div className="flex flex-wrap gap-3">
        <Button variant="outline" onClick={onSetupWebhook}>
          <svg
            className="w-4 h-4 mr-2"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
            />
          </svg>
          {t("repositories.detail.setupWebhook")}
        </Button>
      </div>
    </div>
  );
}
