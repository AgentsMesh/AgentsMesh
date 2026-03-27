"use client";

import { useMemo } from "react";
import { useTheme } from "next-themes";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import type { TokenUsageByAgent } from "@/lib/api/token-usage";
import type { TranslationFn } from "../GeneralSettings";
import { formatTokenCount } from "./format";
import { resolveChartColors } from "./chart-colors";

interface UsageByAgentChartProps {
  data: TokenUsageByAgent[];
  t: TranslationFn;
}

export function UsageByAgentChart({ data, t }: UsageByAgentChartProps) {
  const { resolvedTheme } = useTheme();
  const colors = useMemo(() => resolveChartColors(resolvedTheme === "dark"), [resolvedTheme]);
  const chartData = useMemo(
    () =>
      data.map((item) => ({
        name: item.agent_slug,
        input_tokens: item.input_tokens,
        output_tokens: item.output_tokens,
        cache_creation_tokens: item.cache_creation_tokens,
        cache_read_tokens: item.cache_read_tokens,
      })),
    [data]
  );

  if (data.length === 0) {
    return (
      <div className="border border-border rounded-lg p-6">
        <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.byAgentTitle")}</h3>
        <p className="text-sm text-muted-foreground text-center py-8">
          {t("settings.usagePage.noData")}
        </p>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.byAgentTitle")}</h3>
      <div role="img" aria-label={t("settings.usagePage.byAgentTitle")}>
        <ResponsiveContainer width="100%" height={Math.min(600, Math.max(200, data.length * 50 + 60))}>
          <BarChart data={chartData} layout="vertical" margin={{ top: 5, right: 20, left: 80, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
            <XAxis type="number" tickFormatter={formatTokenCount} tick={{ fontSize: 12 }} />
            <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} width={70} />
            <Tooltip
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              formatter={(value: any, name: any) => [formatTokenCount(Number(value)), name]}
              contentStyle={{
                backgroundColor: "hsl(var(--popover))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "6px",
                fontSize: "12px",
              }}
            />
            <Legend />
            <Bar dataKey="input_tokens" name={t("settings.usagePage.inputTokens")} fill={colors.input} stackId="stack" />
            <Bar dataKey="output_tokens" name={t("settings.usagePage.outputTokens")} fill={colors.output} stackId="stack" />
            <Bar dataKey="cache_creation_tokens" name={t("settings.usagePage.cacheCreationTokens")} fill={colors.cacheCreation} stackId="stack" />
            <Bar dataKey="cache_read_tokens" name={t("settings.usagePage.cacheReadTokens")} fill={colors.cacheRead} stackId="stack" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
