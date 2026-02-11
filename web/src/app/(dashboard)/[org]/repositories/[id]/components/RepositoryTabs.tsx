"use client";

import { useTranslations } from "next-intl";
import { RepositoryTab } from "./useRepositoryDetail";

interface RepositoryTabsProps {
  activeTab: RepositoryTab;
  onTabChange: (tab: RepositoryTab) => void;
}

export function RepositoryTabs({ activeTab, onTabChange }: RepositoryTabsProps) {
  const t = useTranslations();

  const tabs: { key: RepositoryTab; label: string }[] = [
    { key: "info", label: t("repositories.detail.information") },
    { key: "branches", label: t("repositories.detail.branches") },
  ];

  return (
    <div className="border-b border-border mb-6">
      <div className="flex gap-4">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.key
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => onTabChange(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>
    </div>
  );
}
