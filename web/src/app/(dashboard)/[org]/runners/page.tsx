"use client";

import { useState, useEffect } from "react";
import { runnerApi, type RunnerData } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Server,
  Plus,
  Copy,
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
  Terminal,
  Check,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n/client";

export default function RunnersPage() {
  const t = useTranslations();
  const [runners, setRunners] = useState<RunnerData[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddRunnerModal, setShowAddRunnerModal] = useState(false);
  const [selectedRunner, setSelectedRunner] = useState<RunnerData | null>(null);

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
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "offline":
        return <PowerOff className="w-4 h-4 text-gray-500" />;
      case "busy":
        return <Activity className="w-4 h-4 text-yellow-500" />;
      case "maintenance":
        return <AlertCircle className="w-4 h-4 text-orange-500" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
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
    if (!confirm(t("runners.page.confirmDelete", { nodeId: runner.node_id }))) {
      return;
    }
    try {
      await runnerApi.delete(runner.id);
      loadData();
    } catch (error) {
      console.error("Failed to delete runner:", error);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
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

      {/* Add Runner Modal */}
      {showAddRunnerModal && (
        <AddRunnerModal
          t={t}
          onClose={() => setShowAddRunnerModal(false)}
          onCreated={() => {
            setShowAddRunnerModal(false);
            loadData();
          }}
        />
      )}

      {/* Runner Config Modal */}
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
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
  variant,
}: {
  title: string;
  value: number;
  icon: React.ReactNode;
  variant?: "success" | "warning" | "error";
}) {
  return (
    <div className="p-3 md:p-4 border border-border rounded-lg bg-card">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs md:text-sm text-muted-foreground">{title}</p>
          <p className="text-xl md:text-2xl font-bold">{value}</p>
        </div>
        <div
          className={cn(
            "w-8 h-8 md:w-10 md:h-10 rounded-lg flex items-center justify-center",
            variant === "success"
              ? "bg-green-500/10 text-green-500"
              : variant === "warning"
                ? "bg-yellow-500/10 text-yellow-500"
                : variant === "error"
                  ? "bg-red-500/10 text-red-500"
                  : "bg-primary/10 text-primary"
          )}
        >
          {icon}
        </div>
      </div>
    </div>
  );
}

function AddRunnerModal({
  t,
  onClose,
  onCreated,
}: {
  t: (key: string, params?: Record<string, string | number>) => string;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [loading, setLoading] = useState(false);
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleGenerate = async () => {
    setLoading(true);
    try {
      const res = await runnerApi.createToken();
      setGeneratedToken(res.token);
    } catch (error) {
      console.error("Failed to generate token:", error);
    } finally {
      setLoading(false);
    }
  };

  const copyToken = () => {
    if (generatedToken) {
      navigator.clipboard.writeText(generatedToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const copyCommand = () => {
    if (generatedToken) {
      const command = `agentsmesh-runner --token ${generatedToken}`;
      navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-lg p-4 md:p-6">
        <h2 className="text-lg md:text-xl font-semibold mb-2">
          {t("runners.addRunnerModal.title")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("runners.addRunnerModal.subtitle")}
        </p>

        {generatedToken ? (
          <div className="space-y-4">
            {/* Warning */}
            <div className="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <AlertCircle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-yellow-800 dark:text-yellow-200">
                {t("runners.addRunnerModal.tokenWarning")}
              </p>
            </div>

            {/* Token display */}
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.tokenLabel")}
              </label>
              <div className="flex gap-2">
                <code className="flex-1 p-3 bg-muted rounded text-sm break-all font-mono">
                  {generatedToken}
                </code>
                <Button variant="outline" size="sm" onClick={copyToken} className="flex-shrink-0">
                  {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                </Button>
              </div>
            </div>

            {/* Usage instructions */}
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.usageTitle")}
              </label>
              <div className="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 relative">
                <div className="flex items-center gap-2 text-zinc-400 text-xs mb-2">
                  <Terminal className="w-4 h-4" />
                  <span>Terminal</span>
                </div>
                <code className="text-green-400 text-sm font-mono block">
                  agentsmesh-runner --token {generatedToken.substring(0, 16)}...
                </code>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={copyCommand}
                  className="absolute top-2 right-2 h-7 text-xs text-zinc-400 hover:text-white"
                >
                  {t("runners.addRunnerModal.copyCommand")}
                </Button>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <Button onClick={onCreated}>{t("runners.addRunnerModal.done")}</Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t("runners.addRunnerModal.generateHint")}
            </p>

            <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
              <Button variant="outline" onClick={onClose}>
                {t("runners.addRunnerModal.cancel")}
              </Button>
              <Button onClick={handleGenerate} disabled={loading}>
                {loading ? t("runners.addRunnerModal.generating") : t("runners.addRunnerModal.generate")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function RunnerConfigModal({
  t,
  runner,
  onClose,
  onUpdated,
}: {
  t: (key: string, params?: Record<string, string | number>) => string;
  runner: RunnerData;
  onClose: () => void;
  onUpdated: () => void;
}) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods);
  const [loading, setLoading] = useState(false);

  const handleUpdate = async () => {
    setLoading(true);
    try {
      await runnerApi.update(runner.id, {
        description: description || undefined,
        max_concurrent_pods: maxPods,
      });
      onUpdated();
    } catch (error) {
      console.error("Failed to update runner:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6">
        <h2 className="text-lg md:text-xl font-semibold mb-4">
          {t("runners.configModal.title")}
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">{t("runners.configModal.nodeIdLabel")}</label>
            <code className="block w-full p-3 bg-muted rounded text-sm">
              {runner.node_id}
            </code>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t("runners.configModal.descriptionLabel")}</label>
            <input
              type="text"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              placeholder={t("runners.configModal.descriptionPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("runners.configModal.maxPodsLabel")}
            </label>
            <input
              type="number"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              value={maxPods}
              onChange={(e) => setMaxPods(parseInt(e.target.value) || 1)}
              min={1}
              max={100}
            />
          </div>

          {runner.active_pods && runner.active_pods.length > 0 && (
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.configModal.activePodsLabel", { count: runner.active_pods.length })}
              </label>
              <div className="space-y-2 max-h-32 overflow-y-auto">
                {runner.active_pods.map((pod) => (
                  <div
                    key={pod.pod_key}
                    className="flex items-center justify-between p-2 bg-muted rounded text-sm"
                  >
                    <code>{pod.pod_key.substring(0, 12)}...</code>
                    <span className="text-muted-foreground">{pod.status}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
            <Button variant="outline" onClick={onClose}>
              {t("runners.configModal.cancel")}
            </Button>
            <Button onClick={handleUpdate} disabled={loading}>
              {loading ? t("runners.configModal.saving") : t("runners.configModal.save")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
