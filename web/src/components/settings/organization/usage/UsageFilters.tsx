"use client";

import type { TranslationFn } from "../GeneralSettings";

export type TimeRange = "7d" | "30d" | "90d";
export type Granularity = "day" | "week" | "month";

interface UsageFiltersProps {
  timeRange: TimeRange;
  granularity: Granularity;
  agent: string;
  onTimeRangeChange: (range: TimeRange) => void;
  onGranularityChange: (granularity: Granularity) => void;
  onAgentChange: (agent: string) => void;
  agents: string[];
  t: TranslationFn;
}

const timeRangeOptions: { value: TimeRange; labelKey: string }[] = [
  { value: "7d", labelKey: "settings.usagePage.filters.last7Days" },
  { value: "30d", labelKey: "settings.usagePage.filters.last30Days" },
  { value: "90d", labelKey: "settings.usagePage.filters.last90Days" },
];

const granularityOptions: { value: Granularity; labelKey: string }[] = [
  { value: "day", labelKey: "settings.usagePage.filters.day" },
  { value: "week", labelKey: "settings.usagePage.filters.week" },
  { value: "month", labelKey: "settings.usagePage.filters.month" },
];

export function UsageFilters({
  timeRange,
  granularity,
  agent,
  onTimeRangeChange,
  onGranularityChange,
  onAgentChange,
  agents,
  t,
}: UsageFiltersProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      {/* Time Range */}
      <div className="flex items-center gap-1.5">
        <span className="text-sm text-muted-foreground">
          {t("settings.usagePage.filters.timeRange")}:
        </span>
        <div className="inline-flex rounded-md border border-border overflow-hidden">
          {timeRangeOptions.map((opt) => (
            <button
              key={opt.value}
              className={`px-3 py-1 text-sm transition-colors ${
                timeRange === opt.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-background text-muted-foreground hover:bg-muted"
              }`}
              onClick={() => onTimeRangeChange(opt.value)}
            >
              {t(opt.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Granularity */}
      <div className="flex items-center gap-1.5">
        <span className="text-sm text-muted-foreground">
          {t("settings.usagePage.filters.granularity")}:
        </span>
        <div className="inline-flex rounded-md border border-border overflow-hidden">
          {granularityOptions.map((opt) => (
            <button
              key={opt.value}
              className={`px-3 py-1 text-sm transition-colors ${
                granularity === opt.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-background text-muted-foreground hover:bg-muted"
              }`}
              onClick={() => onGranularityChange(opt.value)}
            >
              {t(opt.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Agent Filter */}
      <div className="flex items-center gap-1.5">
        <span className="text-sm text-muted-foreground">
          {t("settings.usagePage.filters.agent")}:
        </span>
        <select
          value={agent}
          onChange={(e) => onAgentChange(e.target.value)}
          disabled={agents.length === 0}
          className="rounded-md border border-border bg-background px-3 py-1 text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <option value="">{t("settings.usagePage.filters.allAgents")}</option>
          {agents.map((agentSlug) => (
            <option key={agentSlug} value={agentSlug}>
              {agentSlug}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
