"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import {
  Server, ArrowLeft, RefreshCw, Trash2, Power, PowerOff,
  CheckCircle, Activity, AlertCircle, Clock,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { getLocalRunnerService } from "@agentsmesh/service-runtime";
import {
  RunnerOverviewTab,
  RunnerPodsTab,
  ResumeDialog,
  useRunnerDetail,
} from "@/app/(dashboard)/[org]/runners/[id]/components";

interface Props {
  runnerId: number;
  onBack: () => void;
}

function statusIcon(status: string) {
  switch (status) {
    case "online": return <CheckCircle className="h-4 w-4 text-green-500" />;
    case "offline": return <PowerOff className="h-4 w-4 text-gray-400" />;
    case "busy": return <Activity className="h-4 w-4 text-yellow-500" />;
    case "maintenance": return <AlertCircle className="h-4 w-4 text-orange-500" />;
    default: return <Clock className="h-4 w-4 text-gray-400" />;
  }
}

export function InfraRunnerDetail({ runnerId, onBack }: Props) {
  const t = useTranslations();
  const state = useRunnerDetail(t, runnerId);
  const [localNodeId, setLocalNodeId] = useState<string | null>(null);

  useEffect(() => {
    const svc = getLocalRunnerService();
    if (!svc) return;
    let cancelled = false;
    void svc.local_node_id().then((id: string | null) => { if (!cancelled) setLocalNodeId(id); }).catch(() => {});
    return () => { cancelled = true; };
  }, []);

  if (state.loading) return <CenteredSpinner className="h-64" />;

  if (!state.runner) {
    return (
      <div className="py-6">
        <p className="text-muted-foreground">{t("runners.detail.notFound")}</p>
        <Button variant="outline" className="mt-4" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          {t("common.back")}
        </Button>
      </div>
    );
  }

  const { runner } = state;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Server className="h-8 w-8 text-muted-foreground" />
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-foreground">{runner.node_id}</h1>
              {localNodeId && runner.node_id === localNodeId && (
                <span className="rounded bg-blue-500/15 px-2 py-0.5 text-xs font-medium uppercase tracking-wide text-blue-600 dark:text-blue-400">
                  This Mac
                </span>
              )}
            </div>
            <div className="flex items-center space-x-2 text-sm text-muted-foreground">
              {statusIcon(runner.status)}
              <span className="capitalize">{runner.status}</span>
              {!runner.is_enabled && <span className="text-red-500">({t("runners.detail.disabled")})</span>}
            </div>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={state.loadRunner}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t("common.refresh")}
          </Button>
          <Button
            variant={runner.is_enabled ? "outline" : "default"}
            onClick={state.handleToggleEnabled}
          >
            {runner.is_enabled ? (
              <>
                <PowerOff className="mr-2 h-4 w-4" />
                {t("runners.detail.disable")}
              </>
            ) : (
              <>
                <Power className="mr-2 h-4 w-4" />
                {t("runners.detail.enable")}
              </>
            )}
          </Button>
          <Button variant="destructive" onClick={state.handleDelete}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("common.delete")}
          </Button>
        </div>
      </div>

      <div className="border-b border-border">
        <nav className="flex space-x-8">
          {(["overview", "pods"] as const).map((tab) => (
            <button
              key={tab}
              data-testid={`runner-detail-tab-${tab}`}
              onClick={() => state.setActiveTab(tab)}
              className={cn(
                "border-b-2 px-1 py-4 text-sm font-medium transition-colors",
                state.activeTab === tab
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground",
              )}
            >
              {t(`runners.detail.tabs.${tab}`)}
            </button>
          ))}
        </nav>
      </div>

      {state.activeTab === "overview" && (
        <RunnerOverviewTab
          runner={runner}
          relayConnections={state.relayConnections}
          latestRunnerVersion={state.latestRunnerVersion}
        />
      )}
      {state.activeTab === "pods" && (
        <RunnerPodsTab
          runner={runner}
          pods={state.pods}
          sandboxStatuses={state.sandboxStatuses}
          loadingPods={state.loadingPods}
          loadingSandbox={state.loadingSandbox}
          podFilter={state.podFilter}
          total={state.total}
          offset={state.offset}
          limit={state.limit}
          onFilterChange={state.setPodFilter}
          onOffsetChange={state.setOffset}
          onRefresh={state.loadPods}
          onRefreshSandbox={state.handleRefreshSandboxStatus}
          onResume={(pod) => {
            state.setResumingPod(pod);
            state.setResumeError(null);
            state.setResumeDialogOpen(true);
          }}
        />
      )}

      <ResumeDialog
        open={state.resumeDialogOpen}
        onOpenChange={(open) => {
          state.setResumeDialogOpen(open);
          if (!open) {
            state.setResumingPod(null);
            state.setResumeError(null);
          }
        }}
        pod={state.resumingPod}
        loading={state.resumeLoading}
        error={state.resumeError}
        onConfirm={state.handleConfirmResume}
      />
      <ConfirmDialog {...state.deleteDialog.dialogProps} />
    </div>
  );
}
