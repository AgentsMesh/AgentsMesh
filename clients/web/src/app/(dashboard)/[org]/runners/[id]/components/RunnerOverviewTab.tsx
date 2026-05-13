"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { format, formatDistanceToNow } from "date-fns";
import { Cpu, HardDrive, Terminal, ArrowUpCircle } from "lucide-react";
import type { RunnerData, RelayConnectionInfo } from "@/lib/api/runnerTypes";
import { upgradeRunner } from "@/lib/api/runnerConnect";
import { isVersionOutdated } from "@/lib/utils/version";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { RelayConnectionsCard } from "./RelayConnectionsCard";
import { RunnerLogsCard } from "./RunnerLogsCard";

interface RunnerOverviewTabProps {
  runner: RunnerData;
  relayConnections?: RelayConnectionInfo[];
  latestRunnerVersion?: string;
}

export function RunnerOverviewTab({ runner, relayConnections, latestRunnerVersion }: RunnerOverviewTabProps) {
  const t = useTranslations();
  const params = useParams();
  const orgSlug = String(params.org ?? "");
  const [upgrading, setUpgrading] = useState(false);
  const { dialogProps, confirm } = useConfirmDialog();

  const hasUpdate = !!latestRunnerVersion && isVersionOutdated(runner.runner_version, latestRunnerVersion);
  const canUpgrade = hasUpdate && runner.status === "online";

  const handleUpgrade = async () => {
    const confirmed = await confirm({
      title: t("runners.detail.upgradeDialogTitle"),
      description: runner.current_pods > 0
        ? t("runners.detail.upgradeConfirmWithPods", { count: runner.current_pods })
        : t("runners.detail.upgradeConfirm"),
      confirmText: t("runners.detail.upgrade"),
      cancelText: t("common.cancel"),
      variant: runner.current_pods > 0 ? "warning" : "default",
    });
    if (!confirmed) return;

    setUpgrading(true);
    try {
      await upgradeRunner(orgSlug, runner.id);
      toast.success(t("runners.detail.upgradeSent"));
    } catch {
      toast.error(t("runners.detail.upgradeFailed"));
    } finally {
      setUpgrading(false);
    }
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {/* Basic Info */}
      <div className="bg-card rounded-lg border border-border p-6">
        <h3 className="text-lg font-medium text-foreground mb-4">
          {t("runners.detail.basicInfo")}
        </h3>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-muted-foreground">
              {t("runners.detail.nodeId")}
            </dt>
            <dd className="text-sm font-medium text-foreground">
              {runner.node_id}
            </dd>
          </div>
          {runner.description && (
            <div>
              <dt className="text-sm text-muted-foreground">
                {t("runners.detail.description")}
              </dt>
              <dd className="text-sm text-foreground">
                {runner.description}
              </dd>
            </div>
          )}
          <div>
            <dt className="text-sm text-muted-foreground">
              {t("runners.detail.version")}
            </dt>
            <dd className="text-sm text-foreground">
              <span className="flex items-center gap-2">
                {runner.runner_version || "-"}
                {hasUpdate && (
                  <>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                      {t("runners.detail.upgradeAvailable", { version: latestRunnerVersion })}
                    </span>
                    <button
                      onClick={handleUpgrade}
                      disabled={!canUpgrade || upgrading}
                      title={
                        runner.status !== "online"
                          ? t("runners.detail.upgradeOffline")
                          : t("runners.detail.upgradeTooltip")
                      }
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      <ArrowUpCircle className="w-3.5 h-3.5" />
                      {upgrading ? t("runners.detail.upgrading") : t("runners.detail.upgrade")}
                    </button>
                  </>
                )}
              </span>
            </dd>
          </div>
          <div>
            <dt className="text-sm text-muted-foreground">
              {t("runners.detail.lastHeartbeat")}
            </dt>
            <dd className="text-sm text-foreground">
              {runner.last_heartbeat
                ? formatDistanceToNow(new Date(runner.last_heartbeat), { addSuffix: true })
                : "-"}
            </dd>
          </div>
          <div>
            <dt className="text-sm text-muted-foreground">
              {t("runners.detail.createdAt")}
            </dt>
            <dd className="text-sm text-foreground">
              {format(new Date(runner.created_at ?? ''), "PPpp")}
            </dd>
          </div>
        </dl>
      </div>

      {/* Capacity */}
      <div className="bg-card rounded-lg border border-border p-6">
        <h3 className="text-lg font-medium text-foreground mb-4">
          {t("runners.detail.capacity")}
        </h3>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-muted-foreground">
              {t("runners.detail.currentPods")}
            </dt>
            <dd className="text-sm font-medium text-foreground">
              {runner.current_pods} / {runner.max_concurrent_pods}
            </dd>
          </div>
          {runner.host_info && (
            <>
              <div>
                <dt className="text-sm text-muted-foreground flex items-center">
                  <Cpu className="w-4 h-4 mr-1" />
                  {t("runners.detail.cpu")}
                </dt>
                <dd className="text-sm text-foreground">
                  {runner.host_info.cpu_cores} cores ({runner.host_info.arch})
                </dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground flex items-center">
                  <HardDrive className="w-4 h-4 mr-1" />
                  {t("runners.detail.memory")}
                </dt>
                <dd className="text-sm text-foreground">
                  {runner.host_info.memory
                    ? `${(runner.host_info.memory / 1024 / 1024 / 1024).toFixed(1)} GB`
                    : "-"}
                </dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">
                  {t("runners.detail.os")}
                </dt>
                <dd className="text-sm text-foreground">
                  {runner.host_info.os || "-"}
                </dd>
              </div>
            </>
          )}
        </dl>
      </div>

      {/* Available Agents */}
      {runner.available_agents && runner.available_agents.length > 0 && (
        <div className="bg-card rounded-lg border border-border p-6 md:col-span-2">
          <h3 className="text-lg font-medium text-foreground mb-4">
            {t("runners.detail.availableAgents")}
          </h3>
          <div className="flex flex-wrap gap-2">
            {runner.available_agents.map((agent) => (
              <span
                key={agent}
                className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
              >
                <Terminal className="w-4 h-4 mr-1" />
                {agent}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Tags */}
      {runner.tags && runner.tags.length > 0 && (
        <div className="bg-card rounded-lg border border-border p-6 md:col-span-2">
          <h3 className="text-lg font-medium text-foreground mb-4">
            {t("runners.detail.tags") || "Tags"}
          </h3>
          <div className="flex flex-wrap gap-2">
            {runner.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      {relayConnections && relayConnections.length > 0 && (
        <RelayConnectionsCard connections={relayConnections} />
      )}

      {/* Diagnostic Logs */}
      <RunnerLogsCard runnerId={runner.id} runnerStatus={runner.status} />

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
