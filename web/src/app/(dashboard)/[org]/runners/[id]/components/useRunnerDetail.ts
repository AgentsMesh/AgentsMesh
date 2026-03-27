"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  runnerApi,
  podApi,
  type RunnerData,
  type RunnerPodData,
  type SandboxStatus,
  type RelayConnectionInfo,
} from "@/lib/api";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { useConfirmDialog } from "@/components/ui/confirm-dialog";
import { toast } from "sonner";

export function useRunnerDetail(t: (key: string) => string) {
  const params = useParams();
  const router = useRouter();
  const runnerId = Number(params.id);

  const [runner, setRunner] = useState<RunnerData | null>(null);
  const [latestRunnerVersion, setLatestRunnerVersion] = useState<string | undefined>();
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
      setLatestRunnerVersion(res.latest_runner_version);
    } catch (error) {
      console.error("Failed to load runner:", error);
    } finally {
      setLoading(false);
    }
  }, [runnerId]);

  const loadPods = useCallback(async () => {
    setLoadingPods(true);
    try {
      const res = await runnerApi.listPods(runnerId, { status: podFilter || undefined, limit, offset });
      setPods(res.pods || []);
      setTotal(res.total);
    } catch (error) {
      console.error("Failed to load pods:", error);
    } finally {
      setLoadingPods(false);
    }
  }, [runnerId, podFilter, offset]);

  useEffect(() => { loadRunner(); }, [loadRunner]);
  useEffect(() => { if (activeTab === "pods") loadPods(); }, [activeTab, loadPods]);

  const handleRefreshSandboxStatus = async () => {
    if (!runner || runner.status !== "online") return;
    const inactivePodKeys = pods.filter(p => p.status !== "running" && p.status !== "initializing").map(p => p.pod_key);
    if (inactivePodKeys.length === 0) return;
    setLoadingSandbox(true);
    try {
      const res = await runnerApi.querySandboxes(runnerId, inactivePodKeys);
      const newStatuses = new Map<string, SandboxStatus>();
      for (const status of res.sandboxes || []) newStatuses.set(status.pod_key, status);
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
        runner_id: runner.id, source_pod_key: resumingPod.pod_key,
        resume_agent_session: true, cols: 120, rows: 30,
      });
      setResumeDialogOpen(false);
      setResumingPod(null);
      router.push(`/${params.org}/workspace?pod=${res.pod.pod_key}`);
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("common.error")));
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
      toast.error(getLocalizedErrorMessage(error, t, t("common.error")));
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
      toast.error(getLocalizedErrorMessage(error, t, t("common.error")));
    }
  };

  return {
    runner, latestRunnerVersion, relayConnections, pods, sandboxStatuses,
    loading, loadingPods, loadingSandbox, activeTab, setActiveTab,
    podFilter, setPodFilter, total, offset, setOffset, limit,
    resumeDialogOpen, setResumeDialogOpen, resumingPod, setResumingPod, resumeLoading,
    deleteDialog, loadRunner, loadPods,
    handleRefreshSandboxStatus, handleConfirmResume, handleToggleEnabled, handleDelete,
  };
}
