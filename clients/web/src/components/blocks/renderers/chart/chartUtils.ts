import type React from "react";

import type { ChartSubType, JSONMap } from "@/lib/api/blockstoreTypes";

export interface ChartSeries {
  name?: string;
  color?: string;
  data: JSONMap[];
}

export interface ChartData {
  type: ChartSubType;
  title?: string;
  x_key?: string;
  y_key?: string;
  x_label?: string;
  y_label?: string;
  series: ChartSeries[];
}

const VALID_TYPES: ChartSubType[] = ["bar", "line", "pie", "area", "scatter", "radar"];

// normalize validates just enough of `data` to decide whether we can render
// anything. Unknown type or missing series returns null so the caller can
// fall back to a placeholder without surfacing errors to recharts.
export function normalize(raw: JSONMap): ChartData | null {
  const type = raw?.type as ChartSubType | undefined;
  if (!type || !VALID_TYPES.includes(type)) return null;
  const series = Array.isArray(raw?.series) ? (raw.series as ChartSeries[]) : null;
  if (!series) return null;
  return {
    type,
    title: raw?.title as string | undefined,
    x_key: raw?.x_key as string | undefined,
    y_key: raw?.y_key as string | undefined,
    x_label: raw?.x_label as string | undefined,
    y_label: raw?.y_label as string | undefined,
    series: series.map((s) => ({
      name: s?.name,
      color: s?.color,
      data: Array.isArray(s?.data) ? s.data : [],
    })),
  };
}

export function mergeSeries(series: ChartSeries[], xKey: string): JSONMap[] {
  const byX = new Map<string, JSONMap>();
  series.forEach((s, i) => {
    const key = s.name ?? `series_${i}`;
    s.data.forEach((row) => {
      const x = String(row[xKey] ?? "");
      const bucket = byX.get(x) ?? { [xKey]: row[xKey] };
      bucket[key] = row.value ?? row[key] ?? null;
      byX.set(x, bucket);
    });
  });
  return Array.from(byX.values());
}

export const tooltipStyle: React.CSSProperties = {
  backgroundColor: "hsl(var(--popover))",
  border: "1px solid hsl(var(--border))",
  borderRadius: "6px",
  fontSize: "12px",
};

export function colorAt(series: ChartSeries, index: number, palette: string[]): string {
  return series.color ?? palette[index % palette.length];
}

// Default palettes cycle 6 hues. Tuned for light vs dark using 400/600-level
// Tailwind hex values — matches the usage dashboard convention in the org
// settings area so charts look visually consistent across the product.
export const PALETTE_LIGHT = ["#2563eb", "#059669", "#d97706", "#9333ea", "#dc2626", "#0891b2"];
export const PALETTE_DARK = ["#60a5fa", "#34d399", "#fbbf24", "#c084fc", "#f87171", "#22d3ee"];
