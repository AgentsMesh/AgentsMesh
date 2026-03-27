"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import {
  Server, ArrowLeft, RefreshCw, Trash2, Power, PowerOff,
  CheckCircle, Activity, AlertCircle, Clock,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { RunnerOverviewTab, RunnerPodsTab, ResumeDialog, useRunnerDetail } from "./components";

function getStatusIcon(status: string) {
  switch (status) {
    case "online": return <CheckCircle className="w-4 h-4 text-green-500" />;
    case "offline": return <PowerOff className="w-4 h-4 text-gray-400" />;
    case "busy": return <Activity className="w-4 h-4 text-yellow-500" />;
    case "maintenance": return <AlertCircle className="w-4 h-4 text-orange-500" />;
    default: return <Clock className="w-4 h-4 text-gray-400" />;
  }
}

export default function RunnerDetailPage() {
  const t = useTranslations();
  const state = useRunnerDetail(t);

  if (state.loading) return <CenteredSpinner className="h-64" />;

  if (!state.runner) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("runners.detail.notFound")}</p>
        <Link href="../runners">
          <Button variant="outline" className="mt-4">
            <ArrowLeft className="w-4 h-4 mr-2" />{t("common.back")}
          </Button>
        </Link>
      </div>
    );
  }

  const { runner } = state;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link href="../runners">
            <Button variant="ghost" size="icon"><ArrowLeft className="w-5 h-5" /></Button>
          </Link>
          <div className="flex items-center space-x-3">
            <Server className="w-8 h-8 text-muted-foreground" />
            <div>
              <h1 className="text-2xl font-bold text-foreground">{runner.node_id}</h1>
              <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                {getStatusIcon(runner.status)}
                <span className="capitalize">{runner.status}</span>
                {!runner.is_enabled && <span className="text-red-500">({t("runners.detail.disabled")})</span>}
              </div>
            </div>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={state.loadRunner}>
            <RefreshCw className="w-4 h-4 mr-2" />{t("common.refresh")}
          </Button>
          <Button variant={runner.is_enabled ? "outline" : "default"} onClick={state.handleToggleEnabled}>
            {runner.is_enabled
              ? <><PowerOff className="w-4 h-4 mr-2" />{t("runners.detail.disable")}</>
              : <><Power className="w-4 h-4 mr-2" />{t("runners.detail.enable")}</>}
          </Button>
          <Button variant="destructive" onClick={state.handleDelete}>
            <Trash2 className="w-4 h-4 mr-2" />{t("common.delete")}
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-border">
        <nav className="flex space-x-8">
          {(["overview", "pods"] as const).map(tab => (
            <button key={tab} onClick={() => state.setActiveTab(tab)}
              className={cn("py-4 px-1 border-b-2 font-medium text-sm transition-colors",
                state.activeTab === tab ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground")}>
              {t(`runners.detail.tabs.${tab}`)}
            </button>
          ))}
        </nav>
      </div>

      {state.activeTab === "overview" && (
        <RunnerOverviewTab runner={runner} relayConnections={state.relayConnections} latestRunnerVersion={state.latestRunnerVersion} />
      )}
      {state.activeTab === "pods" && (
        <RunnerPodsTab runner={runner} pods={state.pods} sandboxStatuses={state.sandboxStatuses}
          loadingPods={state.loadingPods} loadingSandbox={state.loadingSandbox}
          podFilter={state.podFilter} total={state.total} offset={state.offset} limit={state.limit}
          onFilterChange={state.setPodFilter} onOffsetChange={state.setOffset}
          onRefresh={state.loadPods} onRefreshSandbox={state.handleRefreshSandboxStatus}
          onResume={(pod) => { state.setResumingPod(pod); state.setResumeDialogOpen(true); }} />
      )}

      <ResumeDialog open={state.resumeDialogOpen}
        onOpenChange={(open) => { state.setResumeDialogOpen(open); if (!open) state.setResumingPod(null); }}
        pod={state.resumingPod} loading={state.resumeLoading} onConfirm={state.handleConfirmResume} />
      <ConfirmDialog {...state.deleteDialog.dialogProps} />
    </div>
  );
}
