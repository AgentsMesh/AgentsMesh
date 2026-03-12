"use client";

import { useMemo } from "react";
import { useTheme } from "next-themes";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import type { TokenUsageTimeSeriesPoint } from "@/lib/api/token-usage";
import type { TranslationFn } from "../GeneralSettings";
import { formatTokenCount } from "./format";
import { resolveChartColors } from "./chart-colors";

interface UsageTimeSeriesChartProps {
  data: TokenUsageTimeSeriesPoint[];
  t: TranslationFn;
}

function formatDate(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  // Include year when timestamp is in a different year (e.g., 90-day ranges crossing year boundary).
  const opts: Intl.DateTimeFormatOptions =
    date.getFullYear() !== now.getFullYear()
      ? { year: "numeric", month: "short", day: "numeric" }
      : { month: "short", day: "numeric" };
  return new Intl.DateTimeFormat(undefined, opts).format(date);
}

export function UsageTimeSeriesChart({ data, t }: UsageTimeSeriesChartProps) {
  const { resolvedTheme } = useTheme();
  const colors = useMemo(() => resolveChartColors(resolvedTheme === "dark"), [resolvedTheme]);
  const chartData = useMemo(
    () => data.map((point) => ({ ...point, date: formatDate(point.period) })),
    [data]
  );

  if (data.length === 0) {
    return (
      <div className="border border-border rounded-lg p-6">
        <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.timeSeriesTitle")}</h3>
        <p className="text-sm text-muted-foreground text-center py-8">
          {t("settings.usagePage.noData")}
        </p>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-6">
      <h3 className="text-sm font-medium mb-4">{t("settings.usagePage.timeSeriesTitle")}</h3>
      <div role="img" aria-label={t("settings.usagePage.timeSeriesTitle")}>
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={chartData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
            <XAxis dataKey="date" className="text-xs" tick={{ fontSize: 12 }} />
            <YAxis tickFormatter={formatTokenCount} className="text-xs" tick={{ fontSize: 12 }} />
            <Tooltip
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              formatter={(value: any, name: any) => [formatTokenCount(Number(value)), name]}
              labelFormatter={(label: string) => label}
              contentStyle={{
                backgroundColor: "hsl(var(--popover))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "6px",
                fontSize: "12px",
              }}
            />
            <Legend />
            <Area
              type="monotone"
              dataKey="input_tokens"
              name={t("settings.usagePage.inputTokens")}
              stackId="1"
              stroke={colors.input}
              fill={colors.input}
              fillOpacity={0.3}
            />
            <Area
              type="monotone"
              dataKey="output_tokens"
              name={t("settings.usagePage.outputTokens")}
              stackId="1"
              stroke={colors.output}
              fill={colors.output}
              fillOpacity={0.3}
            />
            <Area
              type="monotone"
              dataKey="cache_creation_tokens"
              name={t("settings.usagePage.cacheCreationTokens")}
              stackId="1"
              stroke={colors.cacheCreation}
              fill={colors.cacheCreation}
              fillOpacity={0.3}
            />
            <Area
              type="monotone"
              dataKey="cache_read_tokens"
              name={t("settings.usagePage.cacheReadTokens")}
              stackId="1"
              stroke={colors.cacheRead}
              fill={colors.cacheRead}
              fillOpacity={0.3}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
