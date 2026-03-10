"use client";

import { useState } from "react";
import { format, formatDistanceToNow } from "date-fns";
import { Cpu, HardDrive, Terminal, Radio, ArrowUpCircle } from "lucide-react";
import type { RunnerData, RelayConnectionInfo } from "@/lib/api";
import { runnerApi } from "@/lib/api";
import { isVersionOutdated } from "@/lib/utils/version";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

interface RunnerOverviewTabProps {
  runner: RunnerData;
  relayConnections?: RelayConnectionInfo[];
  latestRunnerVersion?: string;
}

/**
 * Overview tab content showing runner basic info, capacity, available agents, and relay connections
 */
export function RunnerOverviewTab({ runner, relayConnections, latestRunnerVersion }: RunnerOverviewTabProps) {
  const t = useTranslations();
  const [upgrading, setUpgrading] = useState(false);

  const hasUpdate = !!latestRunnerVersion && isVersionOutdated(runner.runner_version, latestRunnerVersion);
  const canUpgrade = hasUpdate
    && runner.status === "online"
    && runner.current_pods === 0;

  const handleUpgrade = async () => {
    if (!confirm(t("runners.detail.upgradeConfirm"))) return;
    setUpgrading(true);
    try {
      await runnerApi.upgrade(runner.id);
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
                          : runner.current_pods > 0
                            ? t("runners.detail.upgradeHasPods")
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
              {format(new Date(runner.created_at), "PPpp")}
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

      {/* Relay Connections */}
      {relayConnections && relayConnections.length > 0 && (
        <div className="bg-card rounded-lg border border-border p-6 md:col-span-2">
          <h3 className="text-lg font-medium text-foreground mb-4 flex items-center">
            <Radio className="w-5 h-5 mr-2 text-green-500" />
            {t("runners.detail.relayConnections")}
            <span className="ml-2 text-sm font-normal text-muted-foreground">
              ({relayConnections.length})
            </span>
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-border">
              <thead>
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Pod
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Relay
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {t("runners.detail.status")}
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {t("runners.detail.connectedSince")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {relayConnections.map((conn) => (
                  <tr key={conn.pod_key}>
                    <td className="px-4 py-3 text-sm font-mono text-foreground">
                      {conn.pod_key}
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {extractRelayHost(conn.relay_url)}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        conn.connected
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : "bg-muted text-muted-foreground"
                      }`}>
                        {conn.connected ? t("common.connected") : t("common.disconnected")}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {conn.connected_at
                        ? formatDistanceToNow(new Date(conn.connected_at), { addSuffix: true })
                        : "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Extract host from relay URL for display
 */
function extractRelayHost(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.host;
  } catch {
    return url;
  }
}
