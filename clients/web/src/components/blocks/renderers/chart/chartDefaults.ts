import type { ChartSubType, JSONMap } from "@/lib/api/blockstoreTypes";

// Single source of truth for chart block seed data — the slash menu and any
// other entry point that creates a chart must use these defaults so the shape
// matches what ChartPreview expects.
// Each sub-type has its own data shape because recharts expects different
// fields per chart family (pie uses {name,value}, scatter uses {x,y}, radar
// uses {axis,value}). Kept explicit rather than runtime-coerced so authors see
// the real shape in the JSON editor the first time they open it.

export const CHART_SUB_TYPES: readonly ChartSubType[] = [
  "bar",
  "line",
  "pie",
  "area",
  "scatter",
  "radar",
] as const;

const HINTS: Record<ChartSubType, string> = {
  bar: "Vertical bar chart",
  line: "Line chart over a sequence",
  pie: "Proportions across a whole",
  area: "Stacked area over time",
  scatter: "Two-dimensional points",
  radar: "Multi-axis radar / spider",
};

export function chartHint(sub: ChartSubType): string {
  return HINTS[sub];
}

export function chartLabel(sub: ChartSubType): string {
  return `Chart / ${sub[0].toUpperCase()}${sub.slice(1)}`;
}

export function chartInitialData(sub: ChartSubType): JSONMap {
  switch (sub) {
    case "bar":
    case "line":
    case "area":
      return {
        type: sub,
        title: `Sample ${sub} chart`,
        x_key: "month",
        series: [
          {
            name: "Revenue",
            color: "#4f46e5",
            data: [
              { month: "Jan", value: 12000 },
              { month: "Feb", value: 15000 },
              { month: "Mar", value: 18200 },
              { month: "Apr", value: 16500 },
            ],
          },
        ],
      };
    case "pie":
      return {
        type: "pie",
        title: "Market share",
        series: [
          {
            name: "Share",
            data: [
              { name: "A", value: 40 },
              { name: "B", value: 30 },
              { name: "C", value: 20 },
              { name: "D", value: 10 },
            ],
          },
        ],
      };
    case "scatter":
      return {
        type: "scatter",
        title: "Sample scatter",
        x_key: "x",
        y_key: "y",
        series: [
          {
            name: "Points",
            color: "#059669",
            data: [
              { x: 10, y: 20 },
              { x: 15, y: 35 },
              { x: 25, y: 30 },
              { x: 40, y: 55 },
            ],
          },
        ],
      };
    case "radar":
      return {
        type: "radar",
        title: "Capability radar",
        x_key: "axis",
        series: [
          {
            name: "Team A",
            color: "#4f46e5",
            data: [
              { axis: "Speed", value: 80 },
              { axis: "Quality", value: 70 },
              { axis: "Coverage", value: 60 },
              { axis: "Cost", value: 90 },
            ],
          },
        ],
      };
  }
}
