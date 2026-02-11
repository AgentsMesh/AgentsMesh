"use client";

import { useState, useEffect } from "react";
import { runnerApi, type RunnerData } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Server,
  Plus,
  Trash2,
  RefreshCw,
  Settings2,
  Power,
  PowerOff,
  AlertCircle,
  CheckCircle,
  Clock,
  Cpu,
  HardDrive,
  Activity,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";
import { StatCard, AddRunnerModal, RunnerConfigModal } from "./components";

/**
 * RunnersPage - Displays a list of runners with their status and actions
 */
export default function RunnersPage() {
  const t = useTranslations();
  const [runners, setRunners] = useState<RunnerData[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddRunnerModal, setShowAddRunnerModal] = useState(false);
  const [selectedRunner, setSelectedRunner] = useState<RunnerData | null>(null);
  const serverUrl = useServerUrl();

  const deleteDialog = useConfirmDialog({
    title: t("runners.page.deleteDialog.title"),
    description: t("runners.page.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const runnersRes = await runnerApi.list();
      setRunners(runnersRes.runners || []);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: RunnerData["status"]) => {
    switch (status) {
      case "online":
        return <CheckCircle className="w-4 h-4 text-green-500 dark:text-green-400" />;
      case "offline":
        return <PowerOff className="w-4 h-4 text-gray-500 dark:text-gray-400" />;
      case "busy":
        return <Activity className="w-4 h-4 text-yellow-500 dark:text-yellow-400" />;
      case "maintenance":
        return <AlertCircle className="w-4 h-4 text-orange-500 dark:text-orange-400" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400 dark:text-gray-500" />;
    }
  };

  const getStatusColor = (status: RunnerData["status"]) => {
    switch (status) {
      case "online":
        return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
      case "offline":
        return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
      case "busy":
        return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400";
      case "maintenance":
        return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400";
      default:
        return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
    }
  };

  const handleToggleEnabled = async (runner: RunnerData) => {
    try {
      await runnerApi.update(runner.id, { is_enabled: !runner.is_enabled });
      loadData();
    } catch (error) {
      console.error("Failed to update runner:", error);
    }
  };

  const handleDeleteRunner = async (runner: RunnerData) => {
    const confirmed = await deleteDialog.confirm();
    if (!confirmed) return;
    try {
      await runnerApi.delete(runner.id);
      loadData();
    } catch (error) {
      console.error("Failed to delete runner:", error);
    }
  };

  if (loading) {
    return <CenteredSpinner />;
  }

  const onlineCount = runners.filter((r) => r.status === "online").length;
  const totalPods = runners.reduce((sum, r) => sum + r.current_pods, 0);
  const totalCapacity = runners.reduce((sum, r) => sum + r.max_concurrent_pods, 0);

  return (
    <div className="p-4 md:p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-xl md:text-2xl font-bold text-foreground">{t("runners.page.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("runners.page.subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadData}>
            <RefreshCw className="w-4 h-4 mr-2" />
            {t("runners.page.refresh")}
          </Button>
          <Button onClick={() => setShowAddRunnerModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t("runners.page.addRunner")}
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 md:gap-4">
        <StatCard
          title={t("runners.page.totalRunners")}
          value={runners.length}
          icon={<Server className="w-5 h-5" />}
        />
        <StatCard
          title={t("runners.page.online")}
          value={onlineCount}
          icon={<Power className="w-5 h-5" />}
          variant="success"
        />
        <StatCard
          title={t("runners.page.activePods")}
          value={totalPods}
          icon={<Cpu className="w-5 h-5" />}
        />
        <StatCard
          title={t("runners.page.totalCapacity")}
          value={totalCapacity}
          icon={<HardDrive className="w-5 h-5" />}
        />
      </div>

      {/* Runners List */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">{t("runners.page.activeRunners")}</h2>

        {/* Mobile: Card view */}
        <div className="block md:hidden space-y-3">
          {runners.map((runner) => (
            <div
              key={runner.id}
              className="p-4 border border-border rounded-lg bg-card"
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  {getStatusIcon(runner.status)}
                  <span className="font-medium truncate">{runner.node_id}</span>
                </div>
                <span
                  className={cn(
                    "px-2 py-1 text-xs rounded-full",
                    getStatusColor(runner.status)
                  )}
                >
                  {runner.status}
                </span>
              </div>

              <div className="space-y-2 text-sm text-muted-foreground mb-3">
                <div className="flex justify-between">
                  <span>{t("runners.page.mobilePodsLabel")}</span>
                  <span>
                    {runner.current_pods} / {runner.max_concurrent_pods}
                  </span>
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
                  <span>{runner.runner_version || "-"}</span>
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  className="flex-1"
                  onClick={() => setSelectedRunner(runner)}
                >
                  <Settings2 className="w-4 h-4 mr-1" />
                  {t("runners.page.configure")}
                </Button>
                <Button
                  size="sm"
                  variant={runner.is_enabled ? "outline" : "default"}
                  onClick={() => handleToggleEnabled(runner)}
                >
                  {runner.is_enabled ? (
                    <PowerOff className="w-4 h-4" />
                  ) : (
                    <Power className="w-4 h-4" />
                  )}
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => handleDeleteRunner(runner)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
          {runners.length === 0 && (
            <div className="text-center py-8 text-muted-foreground border border-dashed border-border rounded-lg">
              <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>{t("runners.page.noRunners")}</p>
              <p className="text-sm mt-1">{t("runners.page.noRunnersHint")}</p>
            </div>
          )}
        </div>

        {/* Desktop: Table view */}
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
                <tr key={runner.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {getStatusIcon(runner.status)}
                      <code className="text-sm bg-muted px-2 py-1 rounded">
                        {runner.node_id}
                      </code>
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
                    {runner.runner_version || "-"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      size="sm"
                      variant="outline"
                      className="mr-2"
                      onClick={() => setSelectedRunner(runner)}
                    >
                      {t("runners.page.configure")}
                    </Button>
                    <Button
                      size="sm"
                      variant={runner.is_enabled ? "outline" : "default"}
                      className="mr-2"
                      onClick={() => handleToggleEnabled(runner)}
                    >
                      {runner.is_enabled ? t("runners.page.disable") : t("runners.page.enable")}
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleDeleteRunner(runner)}
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
      </div>

      {/* Modals */}
      {showAddRunnerModal && (
        <AddRunnerModal
          t={t}
          onClose={() => setShowAddRunnerModal(false)}
          onCreated={() => {
            setShowAddRunnerModal(false);
            loadData();
          }}
          serverUrl={serverUrl}
        />
      )}

      {selectedRunner && (
        <RunnerConfigModal
          t={t}
          runner={selectedRunner}
          onClose={() => setSelectedRunner(null)}
          onUpdated={() => {
            setSelectedRunner(null);
            loadData();
          }}
        />
      )}

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog {...deleteDialog.dialogProps} />
    </div>
  );
}
