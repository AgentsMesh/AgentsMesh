"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { getTokenUsageService } from "@/lib/wasm-core";
import type {
  TokenUsageSummary,
  TokenUsageTimeSeriesPoint,
  TokenUsageByAgent,
  TokenUsageByUser,
  TokenUsageByModel,
  TokenUsageQueryParams,
} from "@/lib/api";
import type { TranslationFn } from "./GeneralSettings";
import {
  UsageOverviewCards,
  UsageTimeSeriesChart,
  UsageByAgentChart,
  UsageByUserTable,
  UsageByModelTable,
  UsageFilters,
  type TimeRange,
  type Granularity,
} from "./usage";

interface UsageSettingsProps {
  t: TranslationFn;
}

const validTimeRanges: TimeRange[] = ["7d", "30d", "90d"];
const validGranularities: Granularity[] = ["day", "week", "month"];

function isValidTimeRange(v: string | null): v is TimeRange {
  return v !== null && validTimeRanges.includes(v as TimeRange);
}

function isValidGranularity(v: string | null): v is Granularity {
  return v !== null && validGranularities.includes(v as Granularity);
}

function getTimeRangeDates(tr: TimeRange): { start: string; end: string } {
  const now = new Date();
  const end = now.toISOString();
  const days = tr === "7d" ? 7 : tr === "30d" ? 30 : 90;
  const start = new Date(now.getTime() - days * 24 * 60 * 60 * 1000).toISOString();
  return { start, end };
}

function UsageLoadingSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="h-8 bg-muted rounded w-48" />
      <div className="h-4 bg-muted rounded w-96" />
      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-5">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="border border-border rounded-lg p-4">
            <div className="h-4 bg-muted rounded w-20 mb-2" />
            <div className="h-8 bg-muted rounded w-24" />
          </div>
        ))}
      </div>
      <div className="border border-border rounded-lg p-6">
        <div className="h-4 bg-muted rounded w-40 mb-4" />
        <div className="h-64 bg-muted rounded" />
      </div>
    </div>
  );
}

export function UsageSettings({ t }: UsageSettingsProps) {
  const router = useRouter();
  const searchParams = useSearchParams();

  // Initialize filter state from URL params (fallback to defaults).
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    const v = searchParams.get("timeRange");
    return isValidTimeRange(v) ? v : "30d";
  });
  const [granularity, setGranularity] = useState<Granularity>(() => {
    const v = searchParams.get("granularity");
    return isValidGranularity(v) ? v : "day";
  });
  const [agent, setAgent] = useState(() => searchParams.get("agent") || "");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [summary, setSummary] = useState<TokenUsageSummary | null>(null);
  const [timeSeries, setTimeSeries] = useState<TokenUsageTimeSeriesPoint[]>([]);
  const [byAgent, setByAgent] = useState<TokenUsageByAgent[]>([]);
  const [byUser, setByUser] = useState<TokenUsageByUser[]>([]);
  const [byModel, setByModel] = useState<TokenUsageByModel[]>([]);

  // Agent list derived from unfiltered data to avoid circular dependency:
  // selecting an agent filter would otherwise shrink the filter options list.
  const [allAgents, setAllAgents] = useState<string[]>([]);

  const abortRef = useRef<AbortController | null>(null);

  const tRef = useRef(t);
  tRef.current = t;

  useEffect(() => {
    const params = new URLSearchParams(searchParams.toString());
    if (timeRange !== "30d") {
      params.set("timeRange", timeRange);
    } else {
      params.delete("timeRange");
    }
    if (granularity !== "day") {
      params.set("granularity", granularity);
    } else {
      params.delete("granularity");
    }
    if (agent) {
      params.set("agent", agent);
    } else {
      params.delete("agent");
    }

    const newQuery = params.toString();
    const currentQuery = searchParams.toString();
    if (newQuery !== currentQuery) {
      router.replace(`?${newQuery}`, { scroll: false });
    }
  }, [timeRange, granularity, agent, searchParams, router]);

  const loadData = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    setError(null);
    const { start, end } = getTimeRangeDates(timeRange);
    const params: TokenUsageQueryParams = {
      start_time: start,
      end_time: end,
      granularity,
      agent_slug: agent || undefined,
    };

    try {
      const raw = await getTokenUsageService().get_dashboard(
        params.start_time ?? null,
        params.end_time ?? null,
        params.agent_slug ?? null,
        params.user_id != null ? BigInt(params.user_id) : null,
        params.model ?? null,
        params.granularity ?? null,
      );
      const data = JSON.parse(raw);

      if (controller.signal.aborted) return;

      setSummary(data.summary ?? null);
      setTimeSeries(data.time_series ?? []);
      setByAgent(data.by_agent ?? []);
      setByUser(data.by_user ?? []);
      setByModel(data.by_model ?? []);

      if (!agent && data.by_agent) {
        setAllAgents(
          [...new Set(data.by_agent.map((a: TokenUsageByAgent) => a.agent_slug))].filter(Boolean) as string[]
        );
      }
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setError(tRef.current("settings.usagePage.loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [timeRange, granularity, agent]);

  useEffect(() => {
    loadData();
    return () => abortRef.current?.abort();
  }, [loadData]);

  // Show full skeleton only on initial load. For subsequent filter changes,
  // keep the existing data visible to avoid jarring flicker.
  if (loading && !summary) {
    return <UsageLoadingSkeleton />;
  }

  if (error && !summary) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <p className="text-destructive">{error}</p>
          <Button variant="outline" className="mt-4" onClick={loadData}>
            {t("settings.usagePage.retry")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">{t("settings.usagePage.title")}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("settings.usagePage.description")}
        </p>
      </div>

      {error && (
        <div className="p-4 rounded-lg bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800">
          {error}
        </div>
      )}

      <UsageFilters
        timeRange={timeRange}
        granularity={granularity}
        agent={agent}
        onTimeRangeChange={setTimeRange}
        onGranularityChange={setGranularity}
        onAgentChange={setAgent}
        agents={allAgents}
        t={t}
      />

      <UsageOverviewCards summary={summary} t={t} />

      <UsageTimeSeriesChart data={timeSeries} t={t} />

      <UsageByAgentChart data={byAgent} t={t} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <UsageByUserTable data={byUser} t={t} />
        <UsageByModelTable data={byModel} t={t} />
      </div>
    </div>
  );
}
