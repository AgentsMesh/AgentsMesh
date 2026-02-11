"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  runnerApi,
  podApi,
  type RunnerData,
  type RunnerPodData,
  type SandboxStatus,
  type RelayConnectionInfo,
} from "@/lib/api";
import { useTranslations } from "next-intl";
import {
  Server,
  ArrowLeft,
  RefreshCw,
  Trash2,
  Power,
  PowerOff,
  CheckCircle,
  Activity,
  AlertCircle,
  Clock,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { RunnerOverviewTab, RunnerPodsTab, ResumeDialog } from "./components";

export default function RunnerDetailPage() {
  const t = useTranslations();
  const params = useParams();
  const router = useRouter();
  const runnerId = Number(params.id);

  const [runner, setRunner] = useState<RunnerData | null>(null);
  const [relayConnections, setRelayConnections] = useState<RelayConnectionInfo[]>([]);
  const [pods, setPods] = useState<RunnerPodData[]>([]);
  const [sandboxStatuses, setSandboxStatuses] = useState<Map<string, SandboxStatus>>(new Map());
  const [loading, setLoading] = useState(true);
  const [loadingPods, setLoadingPods] = useState(false);
  const [loadingSandbox, setLoadingSandbox] = useState(false);
  const [activeTab, setActiveTab] = useState<"overview" | "pods">("overview");
  const [podFilter, setPodFilter] = useState<string>("");
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const limit = 20;

  // Resume confirmation dialog state
  const [resumeDialogOpen, setResumeDialogOpen] = useState(false);
  const [resumingPod, setResumingPod] = useState<RunnerPodData | null>(null);
  const [resumeLoading, setResumeLoading] = useState(false);

  const deleteDialog = useConfirmDialog({
    title: t("runners.detail.deleteDialog.title"),
    description: t("runners.detail.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  const loadRunner = useCallback(async () => {
    try {
      const res = await runnerApi.get(runnerId);
      setRunner(res.runner);
      setRelayConnections(res.relay_connections || []);
    } catch (error) {
      console.error("Failed to load runner:", error);
    } finally {
      setLoading(false);
    }
  }, [runnerId]);

  const loadPods = useCallback(async () => {
    setLoadingPods(true);
    try {
      const res = await runnerApi.listPods(runnerId, {
        status: podFilter || undefined,
        limit,
        offset,
      });
      setPods(res.pods || []);
      setTotal(res.total);
    } catch (error) {
      console.error("Failed to load pods:", error);
    } finally {
      setLoadingPods(false);
    }
  }, [runnerId, podFilter, offset]);

  useEffect(() => {
    loadRunner();
  }, [loadRunner]);

  useEffect(() => {
    if (activeTab === "pods") {
      loadPods();
    }
  }, [activeTab, loadPods]);

  const handleRefreshSandboxStatus = async () => {
    if (!runner || runner.status !== "online") return;

    const inactivePodKeys = pods
      .filter(p => p.status !== "running" && p.status !== "initializing")
      .map(p => p.pod_key);

    if (inactivePodKeys.length === 0) return;

    setLoadingSandbox(true);
    try {
      const res = await runnerApi.querySandboxes(runnerId, inactivePodKeys);
      const newStatuses = new Map<string, SandboxStatus>();
      for (const status of res.sandboxes || []) {
        newStatuses.set(status.pod_key, status);
      }
      setSandboxStatuses(newStatuses);
    } catch (error) {
      console.error("Failed to query sandbox status:", error);
    } finally {
      setLoadingSandbox(false);
    }
  };

  const handleConfirmResume = async () => {
    if (!runner || !resumingPod) return;

    setResumeLoading(true);
    try {
      const res = await podApi.create({
        runner_id: runner.id,
        source_pod_key: resumingPod.pod_key,
        resume_agent_session: true,
        cols: 120,
        rows: 30,
      });

      setResumeDialogOpen(false);
      setResumingPod(null);
      router.push(`/${params.org}/workspace?pod=${res.pod.pod_key}`);
    } catch (error) {
      console.error("Failed to resume pod:", error);
    } finally {
      setResumeLoading(false);
    }
  };

  const handleToggleEnabled = async () => {
    if (!runner) return;
    try {
      await runnerApi.update(runner.id, { is_enabled: !runner.is_enabled });
      loadRunner();
    } catch (error) {
      console.error("Failed to update runner:", error);
    }
  };

  const handleDelete = async () => {
    if (!runner) return;
    const confirmed = await deleteDialog.confirm();
    if (!confirmed) return;
    try {
      await runnerApi.delete(runner.id);
      router.push("../runners");
    } catch (error) {
      console.error("Failed to delete runner:", error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "online":
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "offline":
        return <PowerOff className="w-4 h-4 text-gray-400" />;
      case "busy":
        return <Activity className="w-4 h-4 text-yellow-500" />;
      case "maintenance":
        return <AlertCircle className="w-4 h-4 text-orange-500" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
    }
  };

  if (loading) {
    return <CenteredSpinner className="h-64" />;
  }

  if (!runner) {
    return (
      <div className="p-6">
        <p className="text-gray-500 dark:text-gray-400">
          {t("runners.detail.notFound")}
        </p>
        <Link href="../runners">
          <Button variant="outline" className="mt-4">
            <ArrowLeft className="w-4 h-4 mr-2" />
            {t("common.back")}
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link href="../runners">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="w-5 h-5" />
            </Button>
          </Link>
          <div className="flex items-center space-x-3">
            <Server className="w-8 h-8 text-gray-400" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {runner.node_id}
              </h1>
              <div className="flex items-center space-x-2 text-sm text-gray-500">
                {getStatusIcon(runner.status)}
                <span className="capitalize">{runner.status}</span>
                {!runner.is_enabled && (
                  <span className="text-red-500">({t("runners.detail.disabled")})</span>
                )}
              </div>
            </div>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={loadRunner}>
            <RefreshCw className="w-4 h-4 mr-2" />
            {t("common.refresh")}
          </Button>
          <Button
            variant={runner.is_enabled ? "outline" : "default"}
            onClick={handleToggleEnabled}
          >
            {runner.is_enabled ? (
              <>
                <PowerOff className="w-4 h-4 mr-2" />
                {t("runners.detail.disable")}
              </>
            ) : (
              <>
                <Power className="w-4 h-4 mr-2" />
                {t("runners.detail.enable")}
              </>
            )}
          </Button>
          <Button variant="destructive" onClick={handleDelete}>
            <Trash2 className="w-4 h-4 mr-2" />
            {t("common.delete")}
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex space-x-8">
          <button
            onClick={() => setActiveTab("overview")}
            className={cn(
              "py-4 px-1 border-b-2 font-medium text-sm transition-colors",
              activeTab === "overview"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
            )}
          >
            {t("runners.detail.tabs.overview")}
          </button>
          <button
            onClick={() => setActiveTab("pods")}
            className={cn(
              "py-4 px-1 border-b-2 font-medium text-sm transition-colors",
              activeTab === "pods"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
            )}
          >
            {t("runners.detail.tabs.pods")}
          </button>
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && <RunnerOverviewTab runner={runner} relayConnections={relayConnections} />}

      {activeTab === "pods" && (
        <RunnerPodsTab
          runner={runner}
          pods={pods}
          sandboxStatuses={sandboxStatuses}
          loadingPods={loadingPods}
          loadingSandbox={loadingSandbox}
          podFilter={podFilter}
          total={total}
          offset={offset}
          limit={limit}
          onFilterChange={setPodFilter}
          onOffsetChange={setOffset}
          onRefresh={loadPods}
          onRefreshSandbox={handleRefreshSandboxStatus}
          onResume={(pod) => {
            setResumingPod(pod);
            setResumeDialogOpen(true);
          }}
        />
      )}

      {/* Resume Confirmation Dialog */}
      <ResumeDialog
        open={resumeDialogOpen}
        onOpenChange={(open) => {
          setResumeDialogOpen(open);
          if (!open) setResumingPod(null);
        }}
        pod={resumingPod}
        loading={resumeLoading}
        onConfirm={handleConfirmResume}
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog {...deleteDialog.dialogProps} />
    </div>
  );
}
