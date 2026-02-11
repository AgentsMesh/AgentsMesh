"use client";

import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import {
  RefreshCw,
  CheckCircle,
  XCircle,
  GitBranch,
  FolderOpen,
  RotateCcw,
} from "lucide-react";
import type { RunnerData, RunnerPodData, SandboxStatus } from "@/lib/api";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";

interface RunnerPodsTabProps {
  runner: RunnerData;
  pods: RunnerPodData[];
  sandboxStatuses: Map<string, SandboxStatus>;
  loadingPods: boolean;
  loadingSandbox: boolean;
  podFilter: string;
  total: number;
  offset: number;
  limit: number;
  onFilterChange: (filter: string) => void;
  onOffsetChange: (offset: number) => void;
  onRefresh: () => void;
  onRefreshSandbox: () => void;
  onResume: (pod: RunnerPodData) => void;
}

/**
 * Pods tab content showing pod list with filtering and pagination
 */
export function RunnerPodsTab({
  runner,
  pods,
  sandboxStatuses,
  loadingPods,
  loadingSandbox,
  podFilter,
  total,
  offset,
  limit,
  onFilterChange,
  onOffsetChange,
  onRefresh,
  onRefreshSandbox,
  onResume,
}: RunnerPodsTabProps) {
  const t = useTranslations();

  const getPodStatusBadge = (status: string) => {
    const statusColors: Record<string, string> = {
      running: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
      initializing: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
      terminated: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400",
      error: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
      paused: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
    };
    return statusColors[status] || "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
  };

  return (
    <div className="space-y-4">
      {/* Filters and Actions */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <select
            value={podFilter}
            onChange={(e) => {
              onFilterChange(e.target.value);
              onOffsetChange(0);
            }}
            className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm"
          >
            <option value="">{t("runners.detail.allStatus")}</option>
            <option value="running">{t("pods.status.running")}</option>
            <option value="terminated">{t("pods.status.terminated")}</option>
            <option value="error">{t("pods.status.error")}</option>
          </select>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            onClick={onRefreshSandbox}
            disabled={loadingSandbox || runner.status !== "online"}
          >
            {loadingSandbox ? (
              <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <FolderOpen className="w-4 h-4 mr-2" />
            )}
            {t("runners.detail.refreshSandbox")}
          </Button>
          <Button variant="outline" onClick={onRefresh} disabled={loadingPods}>
            {loadingPods ? (
              <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <RefreshCw className="w-4 h-4 mr-2" />
            )}
            {t("common.refresh")}
          </Button>
        </div>
      </div>

      {/* Pods Table */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-900">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.podKey")}
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.status")}
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Agent Status
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.sandbox")}
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.branch")}
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.createdAt")}
              </th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                {t("runners.detail.actions")}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {pods.map((pod) => {
              const sandboxStatus = sandboxStatuses.get(pod.pod_key);
              const isInactive = pod.status !== "running" && pod.status !== "initializing";
              const canResume = isInactive && sandboxStatus?.can_resume;

              return (
                <tr key={pod.pod_key} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3">
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {pod.pod_key}
                    </span>
                    {pod.source_pod_key && (
                      <span className="ml-2 text-xs text-gray-400">
                        (resumed from {pod.source_pod_key.slice(0, 8)}...)
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={cn(
                        "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium",
                        getPodStatusBadge(pod.status)
                      )}
                    >
                      {pod.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <AgentStatusBadge
                      agentStatus={pod.agent_status}
                      podStatus={pod.status}
                      variant="badge"
                    />
                  </td>
                  <td className="px-4 py-3">
                    {pod.status === "running" ? (
                      <span className="flex items-center text-green-600 dark:text-green-400 text-sm">
                        <CheckCircle className="w-4 h-4 mr-1" />
                        {t("runners.detail.active")}
                      </span>
                    ) : isInactive ? (
                      sandboxStatus === undefined ? (
                        <span className="text-gray-400 text-sm">-</span>
                      ) : sandboxStatus.exists ? (
                        <span className="flex items-center text-green-600 dark:text-green-400 text-sm">
                          <CheckCircle className="w-4 h-4 mr-1" />
                          {sandboxStatus.can_resume ? t("runners.detail.canResume") : t("runners.detail.exists")}
                        </span>
                      ) : (
                        <span className="flex items-center text-gray-400 text-sm">
                          <XCircle className="w-4 h-4 mr-1" />
                          {t("runners.detail.notExists")}
                        </span>
                      )
                    ) : (
                      <span className="text-gray-400 text-sm">-</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                    {pod.branch_name ? (
                      <span className="flex items-center">
                        <GitBranch className="w-4 h-4 mr-1" />
                        {pod.branch_name}
                      </span>
                    ) : (
                      "-"
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                    {formatDistanceToNow(new Date(pod.created_at), { addSuffix: true })}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end space-x-2">
                      {canResume && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onResume(pod)}
                          title={t("runners.detail.resumeTooltip")}
                        >
                          <RotateCcw className="w-4 h-4 mr-1" />
                          {t("runners.detail.resume")}
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
            {pods.length === 0 && (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-gray-500 dark:text-gray-400">
                  {t("runners.detail.noPods")}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {total > limit && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("runners.detail.showing", {
              from: offset + 1,
              to: Math.min(offset + limit, total),
              total,
            })}
          </p>
          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => onOffsetChange(Math.max(0, offset - limit))}
            >
              {t("common.previous")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + limit >= total}
              onClick={() => onOffsetChange(offset + limit)}
            >
              {t("common.next")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
