"use client";

import type { TokenUsageSummary } from "@/lib/api/token-usage";
import type { TranslationFn } from "../GeneralSettings";
import { formatTokenCount, formatNumber } from "./format";

interface UsageOverviewCardsProps {
  summary: TokenUsageSummary | null;
  t: TranslationFn;
}

interface StatCardProps {
  label: string;
  value: number;
  color: string;
}

function StatCard({ label, value, color }: StatCardProps) {
  return (
    <div className="border border-border rounded-lg p-4">
      <p className="text-sm text-muted-foreground mb-1">{label}</p>
      <p className={`text-2xl font-bold ${color}`} title={formatNumber(value)}>
        {formatTokenCount(value)}
      </p>
    </div>
  );
}

export function UsageOverviewCards({ summary, t }: UsageOverviewCardsProps) {
  const cards: StatCardProps[] = [
    {
      label: t("settings.usagePage.inputTokens"),
      value: summary?.input_tokens ?? 0,
      color: "text-blue-600 dark:text-blue-400",
    },
    {
      label: t("settings.usagePage.outputTokens"),
      value: summary?.output_tokens ?? 0,
      color: "text-emerald-600 dark:text-emerald-400",
    },
    {
      label: t("settings.usagePage.cacheCreationTokens"),
      value: summary?.cache_creation_tokens ?? 0,
      color: "text-purple-600 dark:text-purple-400",
    },
    {
      label: t("settings.usagePage.cacheReadTokens"),
      value: summary?.cache_read_tokens ?? 0,
      color: "text-amber-600 dark:text-amber-400",
    },
    {
      label: t("settings.usagePage.totalTokens"),
      value: summary?.total_tokens ?? 0,
      color: "text-foreground",
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-5">
      {cards.map((card) => (
        <StatCard key={card.label} {...card} />
      ))}
    </div>
  );
}
