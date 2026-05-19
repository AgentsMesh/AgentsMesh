"use client";

import React from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarAngleAxis,
  PolarGrid,
  PolarRadiusAxis,
  Radar,
  RadarChart,
  Scatter,
  ScatterChart,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { type ChartData, colorAt, mergeSeries, tooltipStyle } from "./chartUtils";

export function BarVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const xKey = d.x_key ?? "name";
  return (
    <BarChart data={mergeSeries(d.series, xKey)}>
      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
      <XAxis dataKey={xKey} label={axisLabel(d.x_label, "bottom")} />
      <YAxis label={axisLabel(d.y_label, "left")} />
      <Tooltip contentStyle={tooltipStyle} />
      <Legend />
      {d.series.map((s, i) => (
        <Bar key={s.name ?? i} dataKey={s.name ?? `series_${i}`} fill={colorAt(s, i, palette)} />
      ))}
    </BarChart>
  );
}

export function LineVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const xKey = d.x_key ?? "name";
  return (
    <LineChart data={mergeSeries(d.series, xKey)}>
      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
      <XAxis dataKey={xKey} />
      <YAxis />
      <Tooltip contentStyle={tooltipStyle} />
      <Legend />
      {d.series.map((s, i) => (
        <Line key={s.name ?? i} type="monotone" dataKey={s.name ?? `series_${i}`} stroke={colorAt(s, i, palette)} dot={false} />
      ))}
    </LineChart>
  );
}

export function AreaVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const xKey = d.x_key ?? "name";
  return (
    <AreaChart data={mergeSeries(d.series, xKey)}>
      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
      <XAxis dataKey={xKey} />
      <YAxis />
      <Tooltip contentStyle={tooltipStyle} />
      <Legend />
      {d.series.map((s, i) => (
        <Area
          key={s.name ?? i}
          type="monotone"
          dataKey={s.name ?? `series_${i}`}
          stroke={colorAt(s, i, palette)}
          fill={colorAt(s, i, palette)}
          fillOpacity={0.35}
        />
      ))}
    </AreaChart>
  );
}

export function PieVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const slices = d.series[0]?.data ?? [];
  return (
    <PieChart>
      <Tooltip contentStyle={tooltipStyle} />
      <Legend />
      <Pie data={slices} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius="75%" label>
        {slices.map((entry, i) => (
          <Cell key={`cell-${i}`} fill={(entry.color as string | undefined) ?? palette[i % palette.length]} />
        ))}
      </Pie>
    </PieChart>
  );
}

export function ScatterVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const xKey = d.x_key ?? "x";
  const yKey = d.y_key ?? "y";
  return (
    <ScatterChart>
      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
      <XAxis type="number" dataKey={xKey} name={d.x_label ?? xKey} />
      <YAxis type="number" dataKey={yKey} name={d.y_label ?? yKey} />
      <Tooltip cursor={{ strokeDasharray: "3 3" }} contentStyle={tooltipStyle} />
      <Legend />
      {d.series.map((s, i) => (
        <Scatter key={s.name ?? i} name={s.name ?? `Series ${i + 1}`} data={s.data} fill={colorAt(s, i, palette)} />
      ))}
    </ScatterChart>
  );
}

export function RadarVariant({ d, palette }: { d: ChartData; palette: string[] }) {
  const axisKey = d.x_key ?? "axis";
  return (
    <RadarChart data={mergeSeries(d.series, axisKey)}>
      <PolarGrid />
      <PolarAngleAxis dataKey={axisKey} />
      <PolarRadiusAxis />
      <Tooltip contentStyle={tooltipStyle} />
      <Legend />
      {d.series.map((s, i) => (
        <Radar
          key={s.name ?? i}
          name={s.name ?? `Series ${i + 1}`}
          dataKey={s.name ?? `series_${i}`}
          stroke={colorAt(s, i, palette)}
          fill={colorAt(s, i, palette)}
          fillOpacity={0.35}
        />
      ))}
    </RadarChart>
  );
}

function axisLabel(label: string | undefined, position: "bottom" | "left") {
  if (!label) return undefined;
  return position === "bottom"
    ? { value: label, position: "insideBottom" as const }
    : { value: label, angle: -90, position: "insideLeft" as const };
}
