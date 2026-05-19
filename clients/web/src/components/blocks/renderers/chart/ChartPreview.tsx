"use client";

import React from "react";
import { useTheme } from "next-themes";
import { ResponsiveContainer } from "recharts";

import type { JSONMap } from "@/lib/api/blockstoreTypes";

import {
  AreaVariant,
  BarVariant,
  LineVariant,
  PieVariant,
  RadarVariant,
  ScatterVariant,
} from "./chartVariants";
import {
  type ChartData,
  normalize,
  PALETTE_DARK,
  PALETTE_LIGHT,
} from "./chartUtils";

export interface ChartPreviewProps {
  data: JSONMap;
  height?: number;
}

// ChartPreview is a pure renderer: normalize → dispatch on sub-type → wrap in
// ResponsiveContainer. Invalid shapes (missing series, wrong type) fall back
// to a neutral placeholder rather than throwing, so the JSON editor upstream
// can keep showing the draft while the user fixes their input.
export function ChartPreview({ data, height = 300 }: ChartPreviewProps) {
  const { resolvedTheme } = useTheme();
  const palette = resolvedTheme === "dark" ? PALETTE_DARK : PALETTE_LIGHT;

  const parsed = normalize(data);
  if (!parsed || parsed.series.length === 0 || parsed.series.every((s) => s.data.length === 0)) {
    return <Placeholder height={height} message="No data" />;
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      {pickVariant(parsed, palette)}
    </ResponsiveContainer>
  );
}

function pickVariant(d: ChartData, palette: string[]): React.ReactElement {
  switch (d.type) {
    case "bar":
      return <BarVariant d={d} palette={palette} />;
    case "line":
      return <LineVariant d={d} palette={palette} />;
    case "area":
      return <AreaVariant d={d} palette={palette} />;
    case "pie":
      return <PieVariant d={d} palette={palette} />;
    case "scatter":
      return <ScatterVariant d={d} palette={palette} />;
    case "radar":
      return <RadarVariant d={d} palette={palette} />;
  }
}

function Placeholder({ height, message }: { height: number; message: string }) {
  return (
    <div
      className="flex items-center justify-center rounded-md border border-dashed border-border bg-muted/20 text-sm text-muted-foreground"
      style={{ height }}
    >
      {message}
    </div>
  );
}
