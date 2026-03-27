"use client";

import { type RunnerData } from "@/lib/api";
import type { useTranslations } from "next-intl";
import { isVersionOutdated } from "@/lib/utils/version";
import { Button } from "@/components/ui/button";
import { Lock, Building2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { getStatusIcon, getStatusColor } from "./runnerStatusUtils";

interface RunnerTableProps {
  runners: RunnerData[];
  latestVersion?: string;
  t: ReturnType<typeof useTranslations>;
  onNavigate: (id: number) => void;
  onConfigure: (runner: RunnerData) => void;
  onToggleEnabled: (runner: RunnerData) => void;
  onDelete: (runner: RunnerData) => void;
}

export function RunnerTable({
  runners,
  latestVersion,
  t,
  onNavigate,
  onConfigure,
  onToggleEnabled,
  onDelete,
}: RunnerTableProps) {
  return (
    <div className="hidden md:block border border-border rounded-lg overflow-hidden">
      <table className="w-full">
        <thead className="bg-muted">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("runners.page.runnerColumn")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("runners.page.statusColumn")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("runners.page.podsColumn")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("runners.page.hostInfoColumn")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("runners.page.versionColumn")}</th>
            <th className="px-4 py-3 text-right text-sm font-medium">{t("runners.page.actionsColumn")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {runners.map((runner) => (
            <tr
              key={runner.id}
              className="hover:bg-muted/50 cursor-pointer"
              onClick={() => onNavigate(runner.id)}
            >
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  {getStatusIcon(runner.status)}
                  <code className="text-sm bg-muted px-2 py-1 rounded">
                    {runner.node_id}
                  </code>
                  {runner.visibility === "private" ? (
                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                      <Lock className="w-3 h-3" />
                      {t("runners.page.visibilityPrivate")}
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium rounded bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                      <Building2 className="w-3 h-3" />
                      {t("runners.page.visibilityOrganization")}
                    </span>
                  )}
                </div>
              </td>
              <td className="px-4 py-3">
                <span
                  className={cn(
                    "px-2 py-1 text-xs rounded-full",
                    getStatusColor(runner.status)
                  )}
                >
                  {runner.status}
                </span>
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {runner.current_pods} / {runner.max_concurrent_pods}
              </td>
              <td className="px-4 py-3 text-muted-foreground text-sm">
                {runner.host_info ? (
                  <span>
                    {runner.host_info.os} · {runner.host_info.cpu_cores} {t("runners.page.cores")}
                  </span>
                ) : (
                  "-"
                )}
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  {runner.runner_version || "-"}
                  {isVersionOutdated(runner.runner_version, latestVersion) && (
                    <span className="px-1.5 py-0.5 text-[10px] font-medium rounded bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                      {t("runners.page.upgradeAvailable")}
                    </span>
                  )}
                </span>
              </td>
              <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                <Button
                  size="sm"
                  variant="outline"
                  className="mr-2"
                  onClick={() => onConfigure(runner)}
                >
                  {t("runners.page.configure")}
                </Button>
                <Button
                  size="sm"
                  variant={runner.is_enabled ? "outline" : "default"}
                  className="mr-2"
                  onClick={() => onToggleEnabled(runner)}
                >
                  {runner.is_enabled ? t("runners.page.disable") : t("runners.page.enable")}
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => onDelete(runner)}
                >
                  {t("runners.page.delete")}
                </Button>
              </td>
            </tr>
          ))}
          {runners.length === 0 && (
            <tr>
              <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                {t("runners.page.noRunners")} {t("runners.page.noRunnersHint")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
