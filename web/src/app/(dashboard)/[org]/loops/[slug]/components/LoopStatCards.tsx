import { Hash, CheckCircle2, XCircle, Timer } from "lucide-react";
import type { LoopData } from "@/stores/loop";
import { formatDuration } from "@/lib/utils/time";
import { StatCard } from "./StatCard";

interface LoopStatCardsProps {
  loop: LoopData;
  t: (key: string) => string;
}

export function LoopStatCards({ loop, t }: LoopStatCardsProps) {
  const successRate =
    loop.total_runs > 0
      ? Math.round((loop.successful_runs / loop.total_runs) * 100)
      : 0;
  const avgDuration = loop.avg_duration_sec != null ? Math.round(loop.avg_duration_sec) : 0;

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-8">
      <StatCard icon={Hash} label={t("loops.totalRuns")} value={loop.total_runs.toString()} />
      <StatCard
        icon={CheckCircle2}
        iconColor="text-emerald-500"
        label={t("loops.success")}
        value={loop.successful_runs.toString()}
        suffix={loop.total_runs > 0 ? `${successRate}%` : undefined}
      />
      <StatCard icon={XCircle} iconColor="text-red-500" label={t("loops.failed")} value={loop.failed_runs.toString()} />
      <StatCard icon={Timer} label={t("loops.avgDuration")} value={avgDuration > 0 ? formatDuration(avgDuration) : "-"} />
    </div>
  );
}
