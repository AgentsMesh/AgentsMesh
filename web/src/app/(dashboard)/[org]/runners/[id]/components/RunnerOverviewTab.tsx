"use client";

import { format, formatDistanceToNow } from "date-fns";
import { Cpu, HardDrive, Terminal, Radio } from "lucide-react";
import type { RunnerData, RelayConnectionInfo } from "@/lib/api";
import { useTranslations } from "next-intl";

interface RunnerOverviewTabProps {
  runner: RunnerData;
  relayConnections?: RelayConnectionInfo[];
}

/**
 * Overview tab content showing runner basic info, capacity, available agents, and relay connections
 */
export function RunnerOverviewTab({ runner, relayConnections }: RunnerOverviewTabProps) {
  const t = useTranslations();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {/* Basic Info */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
          {t("runners.detail.basicInfo")}
        </h3>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">
              {t("runners.detail.nodeId")}
            </dt>
            <dd className="text-sm font-medium text-gray-900 dark:text-white">
              {runner.node_id}
            </dd>
          </div>
          {runner.description && (
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">
                {t("runners.detail.description")}
              </dt>
              <dd className="text-sm text-gray-900 dark:text-white">
                {runner.description}
              </dd>
            </div>
          )}
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">
              {t("runners.detail.version")}
            </dt>
            <dd className="text-sm text-gray-900 dark:text-white">
              {runner.runner_version || "-"}
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">
              {t("runners.detail.lastHeartbeat")}
            </dt>
            <dd className="text-sm text-gray-900 dark:text-white">
              {runner.last_heartbeat
                ? formatDistanceToNow(new Date(runner.last_heartbeat), { addSuffix: true })
                : "-"}
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">
              {t("runners.detail.createdAt")}
            </dt>
            <dd className="text-sm text-gray-900 dark:text-white">
              {format(new Date(runner.created_at), "PPpp")}
            </dd>
          </div>
        </dl>
      </div>

      {/* Capacity */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
          {t("runners.detail.capacity")}
        </h3>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">
              {t("runners.detail.currentPods")}
            </dt>
            <dd className="text-sm font-medium text-gray-900 dark:text-white">
              {runner.current_pods} / {runner.max_concurrent_pods}
            </dd>
          </div>
          {runner.host_info && (
            <>
              <div>
                <dt className="text-sm text-gray-500 dark:text-gray-400 flex items-center">
                  <Cpu className="w-4 h-4 mr-1" />
                  {t("runners.detail.cpu")}
                </dt>
                <dd className="text-sm text-gray-900 dark:text-white">
                  {runner.host_info.cpu_cores} cores ({runner.host_info.arch})
                </dd>
              </div>
              <div>
                <dt className="text-sm text-gray-500 dark:text-gray-400 flex items-center">
                  <HardDrive className="w-4 h-4 mr-1" />
                  {t("runners.detail.memory")}
                </dt>
                <dd className="text-sm text-gray-900 dark:text-white">
                  {runner.host_info.memory
                    ? `${(runner.host_info.memory / 1024 / 1024 / 1024).toFixed(1)} GB`
                    : "-"}
                </dd>
              </div>
              <div>
                <dt className="text-sm text-gray-500 dark:text-gray-400">
                  {t("runners.detail.os")}
                </dt>
                <dd className="text-sm text-gray-900 dark:text-white">
                  {runner.host_info.os || "-"}
                </dd>
              </div>
            </>
          )}
        </dl>
      </div>

      {/* Available Agents */}
      {runner.available_agents && runner.available_agents.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 md:col-span-2">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
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
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 md:col-span-2">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4 flex items-center">
            <Radio className="w-5 h-5 mr-2 text-green-500" />
            {t("runners.detail.relayConnections")}
            <span className="ml-2 text-sm font-normal text-gray-500">
              ({relayConnections.length})
            </span>
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead>
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Pod
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Relay
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    {t("runners.detail.status")}
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    {t("runners.detail.connectedSince")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {relayConnections.map((conn) => (
                  <tr key={conn.pod_key}>
                    <td className="px-4 py-3 text-sm font-mono text-gray-900 dark:text-white">
                      {conn.pod_key}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {extractRelayHost(conn.relay_url)}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        conn.connected
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                      }`}>
                        {conn.connected ? t("common.connected") : t("common.disconnected")}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
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
