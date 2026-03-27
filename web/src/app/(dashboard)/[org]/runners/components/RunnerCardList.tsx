"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { isVersionOutdated } from "@/lib/utils/version";
import type { RunnerData } from "@/lib/api";
import type { useTranslations } from "next-intl";
import {
  Settings2, Power, PowerOff, Trash2, Lock, Building2, Server,
} from "lucide-react";
import { getStatusIcon, getStatusColor } from "./runnerStatusUtils";

interface RunnerCardListProps {
  runners: RunnerData[];
  latestVersion?: string;
  t: ReturnType<typeof useTranslations>;
  onNavigate: (runnerId: number) => void;
  onConfigure: (runner: RunnerData) => void;
  onToggleEnabled: (runner: RunnerData) => void;
  onDelete: (runner: RunnerData) => void;
}

export function RunnerCardList({
  runners, latestVersion, t, onNavigate, onConfigure, onToggleEnabled, onDelete,
}: RunnerCardListProps) {
  if (runners.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground border border-dashed border-border rounded-lg">
        <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
        <p>{t("runners.page.noRunners")}</p>
        <p className="text-sm mt-1">{t("runners.page.noRunnersHint")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {runners.map((runner) => (
        <div key={runner.id}
          className="p-4 border border-border rounded-lg bg-card cursor-pointer hover:bg-muted/50 transition-colors"
          onClick={() => onNavigate(runner.id)}>
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              {getStatusIcon(runner.status)}
              <span className="font-medium truncate">{runner.node_id}</span>
              {runner.visibility === "private" ? (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                  <Lock className="w-3 h-3" />{t("runners.page.visibilityPrivate")}
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium rounded bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                  <Building2 className="w-3 h-3" />{t("runners.page.visibilityOrganization")}
                </span>
              )}
            </div>
            <span className={cn("px-2 py-1 text-xs rounded-full", getStatusColor(runner.status))}>
              {runner.status}
            </span>
          </div>
          <div className="space-y-2 text-sm text-muted-foreground mb-3">
            <div className="flex justify-between">
              <span>{t("runners.page.mobilePodsLabel")}</span>
              <span>{runner.current_pods} / {runner.max_concurrent_pods}</span>
            </div>
            {runner.host_info && (
              <>
                <div className="flex justify-between">
                  <span>{t("runners.page.mobileOsLabel")}</span>
                  <span>{runner.host_info.os || "-"}</span>
                </div>
                <div className="flex justify-between">
                  <span>{t("runners.page.mobileCpuLabel")}</span>
                  <span>{runner.host_info.cpu_cores || "-"} {t("runners.page.cores")}</span>
                </div>
              </>
            )}
            <div className="flex justify-between">
              <span>{t("runners.page.mobileVersionLabel")}</span>
              <span className="flex items-center gap-1">
                {runner.runner_version || "-"}
                {isVersionOutdated(runner.runner_version, latestVersion) && (
                  <span className="px-1.5 py-0.5 text-[10px] font-medium rounded bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                    {t("runners.page.upgradeAvailable")}
                  </span>
                )}
              </span>
            </div>
          </div>
          <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
            <Button size="sm" variant="outline" className="flex-1" onClick={() => onConfigure(runner)}>
              <Settings2 className="w-4 h-4 mr-1" />{t("runners.page.configure")}
            </Button>
            <Button size="sm" variant={runner.is_enabled ? "outline" : "default"} onClick={() => onToggleEnabled(runner)}>
              {runner.is_enabled ? <PowerOff className="w-4 h-4" /> : <Power className="w-4 h-4" />}
            </Button>
            <Button size="sm" variant="destructive" onClick={() => onDelete(runner)}>
              <Trash2 className="w-4 h-4" />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
