"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { LoopRunData } from "@/stores/loop";
import { Button } from "@/components/ui/button";
import { Clock, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { RunRow } from "./RunRow";

interface LoopRunHistoryProps {
  runs: LoopRunData[];
  loading: boolean;
  total: number;
  onLoadMore?: () => void;
  onViewTerminal?: (podKey: string) => void;
  onCancel?: (runId: number) => void;
  className?: string;
}

export function LoopRunHistory({
  runs, loading, total, onLoadMore, onViewTerminal, onCancel, className,
}: LoopRunHistoryProps) {
  const t = useTranslations();

  if (loading && runs.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center mb-3">
          <Clock className="w-5 h-5 text-muted-foreground" />
        </div>
        <p className="text-sm text-muted-foreground">{t("loops.noRuns")}</p>
      </div>
    );
  }

  return (
    <div className={cn("space-y-1.5", className)}>
      {runs.map((run) => (
        <RunRow key={run.id} run={run} onViewTerminal={onViewTerminal} onCancel={onCancel} />
      ))}

      {runs.length < total && onLoadMore && (
        <div className="flex justify-center pt-2">
          <Button size="sm" variant="ghost" className="text-xs text-muted-foreground" onClick={onLoadMore} disabled={loading}>
            {loading ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : null}
            {t("loops.loadMore")}
          </Button>
        </div>
      )}
    </div>
  );
}
