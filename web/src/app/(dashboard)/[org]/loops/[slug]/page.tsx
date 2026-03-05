"use client";

import { useEffect, useCallback, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useLoopStore } from "@/stores/loop";
import { CenteredSpinner } from "@/components/ui/spinner";
import { LoopRunHistory } from "@/components/loops/LoopRunHistory";
import { LoopCreateDialog } from "@/components/loops/LoopCreateDialog";
import { Button } from "@/components/ui/button";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import Link from "next/link";
import {
  ArrowLeft,
  Play,
  Pencil,
  Loader2,
  CheckCircle2,
  XCircle,
  Timer,
  Hash,
  Bot,
  Zap,
  Shield,
  Layers,
  MoreHorizontal,
  Power,
  Trash2,
  ExternalLink,
  AlertCircle,
  RefreshCw,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { formatDuration } from "@/lib/utils/time";

export default function LoopDetailPage() {
  const t = useTranslations();
  const router = useRouter();
  const params = useParams();
  const slug = params.slug as string;
  const orgSlug = params.org as string;
  const currentLoop = useLoopStore((s) => s.currentLoop);
  const runs = useLoopStore((s) => s.runs);
  const runsLoading = useLoopStore((s) => s.runsLoading);
  const runsTotalCount = useLoopStore((s) => s.runsTotalCount);
  const loopLoading = useLoopStore((s) => s.loopLoading);
  const error = useLoopStore((s) => s.error);
  const fetchLoop = useLoopStore((s) => s.fetchLoop);
  const fetchRuns = useLoopStore((s) => s.fetchRuns);
  const triggerLoop = useLoopStore((s) => s.triggerLoop);
  const cancelRun = useLoopStore((s) => s.cancelRun);
  const enableLoop = useLoopStore((s) => s.enableLoop);
  const disableLoop = useLoopStore((s) => s.disableLoop);
  const deleteLoop = useLoopStore((s) => s.deleteLoop);
  const loadMoreRuns = useLoopStore((s) => s.loadMoreRuns);
  const clearError = useLoopStore((s) => s.clearError);

  const [editOpen, setEditOpen] = useState(false);
  const [triggering, setTriggering] = useState(false);

  const deleteDialog = useConfirmDialog({
    title: t("loops.deleteConfirm"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  const setCurrentLoop = useLoopStore((s) => s.setCurrentLoop);

  useEffect(() => {
    fetchLoop(slug);
    fetchRuns(slug, { limit: 20, offset: 0 });
    return () => {
      // Clear stale data on unmount to prevent flash on next detail page
      setCurrentLoop(null);
    };
  }, [slug, fetchLoop, fetchRuns, setCurrentLoop]);

  const handleTrigger = useCallback(async () => {
    setTriggering(true);
    try {
      const result = await triggerLoop(slug);
      if (result.skipped) {
        toast.info(t("loops.triggerSkipped"), { description: result.reason });
      } else if (result.run) {
        toast.success(t("loops.triggered"), { description: `Run #${result.run.run_number}` });
      }
    } catch {
      toast.error(t("loops.triggerFailed"));
    } finally {
      setTriggering(false);
    }
  }, [slug, triggerLoop, t]);

  const handleLoadMore = useCallback(() => {
    loadMoreRuns(slug);
  }, [slug, loadMoreRuns]);

  const handleCancelRun = useCallback(
    async (runId: number) => {
      try {
        await cancelRun(slug, runId);
        toast.success(t("loops.runCancelled"));
      } catch {
        toast.error(t("loops.cancelFailed"));
      }
    },
    [slug, cancelRun, t]
  );

  const handleViewTerminal = useCallback(
    (podKey: string) => {
      router.push(`/${orgSlug}/workspace?pod=${podKey}`);
    },
    [router, orgSlug]
  );

  const handleEnable = useCallback(async () => {
    try {
      await enableLoop(slug);
      toast.success(t("loops.enabled"));
    } catch {
      toast.error(t("loops.enableFailed"));
    }
  }, [slug, enableLoop, t]);

  const handleDisable = useCallback(async () => {
    try {
      await disableLoop(slug);
      toast.success(t("loops.disabled"));
    } catch {
      toast.error(t("loops.disableFailed"));
    }
  }, [slug, disableLoop, t]);

  const handleDelete = useCallback(async () => {
    const confirmed = await deleteDialog.confirm();
    if (!confirmed) return;
    try {
      await deleteLoop(slug);
      toast.success(t("loops.deleted"));
      router.push(`/${orgSlug}/loops`);
    } catch (err) {
      const message = (err as Error).message;
      const isActiveRunsError = message.includes("active runs");
      toast.error(t("loops.deleteFailed"), {
        description: isActiveRunsError ? t("loops.deleteHasActiveRuns") : message,
      });
    }
  }, [slug, deleteLoop, deleteDialog, router, orgSlug, t]);

  if (loopLoading && !currentLoop) {
    return <CenteredSpinner className="h-full" />;
  }

  if (error && !currentLoop) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center py-20">
        <div className="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mb-3">
          <AlertCircle className="w-6 h-6 text-destructive" />
        </div>
        <p className="text-sm text-muted-foreground mb-3">{error}</p>
        <Button variant="outline" size="sm" className="gap-1.5" onClick={() => { clearError(); fetchLoop(slug); }}>
          <RefreshCw className="w-3.5 h-3.5" />
          {t("loops.retry")}
        </Button>
      </div>
    );
  }

  if (!currentLoop) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center py-20">
        <div className="w-12 h-12 rounded-xl bg-muted flex items-center justify-center mb-3">
          <XCircle className="w-6 h-6 text-muted-foreground" />
        </div>
        <p className="text-sm text-muted-foreground">{t("loops.notFound")}</p>
      </div>
    );
  }

  const loop = currentLoop;
  const isEnabled = loop.status === "enabled";
  const successRate =
    loop.total_runs > 0
      ? Math.round((loop.successful_runs / loop.total_runs) * 100)
      : 0;
  // Use backend-computed avg duration (SSOT) instead of frontend slice calculation
  const avgDuration = loop.avg_duration_sec != null ? Math.round(loop.avg_duration_sec) : 0;

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-5">
        {/* Header */}
        <div className="mb-8">
          <button
            className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-4"
            onClick={() => router.push(`/${orgSlug}/loops`)}
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            {t("loops.back")}
          </button>

          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-3 mb-1.5">
                <h1 className="text-xl font-bold truncate">{loop.name}</h1>
                <span
                  className={cn(
                    "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium flex-shrink-0",
                    isEnabled
                      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                      : "bg-gray-500/10 text-gray-600 dark:text-gray-400"
                  )}
                >
                  <span
                    className={cn(
                      "w-1.5 h-1.5 rounded-full",
                      isEnabled ? "bg-emerald-500" : "bg-gray-400"
                    )}
                  />
                  {isEnabled ? t("loops.statusEnabled") : t("loops.statusDisabled")}
                </span>
              </div>
              {loop.description && (
                <p className="text-sm text-muted-foreground">{loop.description}</p>
              )}
            </div>

            <div className="flex gap-2 flex-shrink-0">
              {isEnabled && (
                <Button size="sm" onClick={handleTrigger} disabled={triggering || loop.active_run_count >= loop.max_concurrent_runs} className="gap-1.5">
                  {triggering ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Play className="w-3.5 h-3.5" />
                  )}
                  {t("loops.trigger")}
                </Button>
              )}
              <Button size="sm" variant="outline" onClick={() => setEditOpen(true)} className="gap-1.5">
                <Pencil className="w-3.5 h-3.5" />
                {t("common.edit")}
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="ghost" className="h-8 w-8 p-0">
                    <MoreHorizontal className="w-4 h-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {isEnabled ? (
                    <DropdownMenuItem onClick={handleDisable}>
                      <Power className="w-4 h-4 mr-2" />
                      {t("loops.disable")}
                    </DropdownMenuItem>
                  ) : (
                    <DropdownMenuItem onClick={handleEnable}>
                      <Power className="w-4 h-4 mr-2" />
                      {t("loops.enable")}
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onClick={handleDelete}
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    {t("common.delete")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>

        {/* Statistics */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-8">
          <StatCard
            icon={Hash}
            label={t("loops.totalRuns")}
            value={loop.total_runs.toString()}
          />
          <StatCard
            icon={CheckCircle2}
            iconColor="text-emerald-500"
            label={t("loops.success")}
            value={loop.successful_runs.toString()}
            suffix={loop.total_runs > 0 ? `${successRate}%` : undefined}
          />
          <StatCard
            icon={XCircle}
            iconColor="text-red-500"
            label={t("loops.failed")}
            value={loop.failed_runs.toString()}
          />
          <StatCard
            icon={Timer}
            label={t("loops.avgDuration")}
            value={avgDuration > 0 ? formatDuration(avgDuration) : "-"}
          />
        </div>

        {/* Configuration + API Trigger — side by side on wide screens */}
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_1fr] gap-3 mb-8">
          {/* Configuration */}
          <section>
            <h2 className="text-sm font-semibold mb-3">{t("loops.configuration")}</h2>
            <div className="border rounded-xl overflow-hidden h-[calc(100%-2rem)]">
              <div className="p-4 space-y-3">
                <ConfigRow
                  icon={<Bot className="w-3.5 h-3.5" />}
                  label={t("loops.mode")}
                  value={loop.execution_mode === "autopilot" ? t("loops.modeAutopilot") : t("loops.modeDirect")}
                />
                <ConfigRow
                  icon={<Layers className="w-3.5 h-3.5" />}
                  label={t("loops.sandbox")}
                  value={loop.sandbox_strategy === "persistent" ? t("loops.sandboxPersistent") : t("loops.sandboxFresh")}
                />
                <ConfigRow
                  icon={<Shield className="w-3.5 h-3.5" />}
                  label={t("loops.concurrency")}
                  value={
                    loop.concurrency_policy === "skip"
                      ? t("loops.policySkip")
                      : loop.concurrency_policy === "queue"
                        ? t("loops.policyQueue")
                        : t("loops.policyReplace")
                  }
                />
                <ConfigRow
                  icon={<Timer className="w-3.5 h-3.5" />}
                  label={t("loops.timeout")}
                  value={`${loop.timeout_minutes} ${t("loops.minutes")}`}
                />
                <ConfigRow
                  icon={<Shield className="w-3.5 h-3.5" />}
                  label={t("loops.sessionLabel")}
                  value={loop.session_persistence ? t("loops.sessionKeep") : t("loops.sessionFresh")}
                />
                <ConfigRow
                  icon={<Hash className="w-3.5 h-3.5" />}
                  label={t("loops.maxConcurrent")}
                  value={loop.max_concurrent_runs.toString()}
                />
                {loop.max_retained_runs > 0 && (
                  <ConfigRow
                    icon={<Hash className="w-3.5 h-3.5" />}
                    label={t("loops.maxRetainedRuns")}
                    value={loop.max_retained_runs.toString()}
                  />
                )}
                <ConfigRow
                  icon={<Timer className="w-3.5 h-3.5" />}
                  label={t("loops.triggerLabel")}
                  value={
                    loop.cron_expression ? (
                      <span className="px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 text-[10px] font-medium font-mono">
                        {loop.cron_expression}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">{t("loops.onDemand")}</span>
                    )
                  }
                />
              </div>

              {/* Callback URL */}
              {loop.callback_url && (
                <div className="px-4 pb-3">
                  <ConfigRow
                    icon={<Zap className="w-3.5 h-3.5" />}
                    label={t("loops.webhookUrl")}
                    value={
                      <span className="text-xs font-mono truncate max-w-[200px] inline-block align-bottom">
                        {loop.callback_url}
                      </span>
                    }
                  />
                </div>
              )}

              {/* Prompt preview */}
              <div className="border-t p-4">
                <div className="text-xs font-medium text-muted-foreground mb-2">
                  {t("loops.prompt")}
                </div>
                <pre className="p-3 bg-muted/50 rounded-lg text-sm whitespace-pre-wrap font-mono leading-relaxed max-h-32 overflow-y-auto text-foreground/80">
                  {loop.prompt_template}
                </pre>
              </div>
            </div>
          </section>

          {/* API Trigger */}
          <section>
            <h2 className="text-sm font-semibold mb-3">{t("loops.apiTrigger")}</h2>
            <div className="border rounded-xl p-4 h-[calc(100%-2rem)]">
              <p className="text-xs text-muted-foreground mb-3">
                {t("loops.apiTriggerDesc")}
              </p>
              <div className="relative">
                <div className="absolute top-2 left-3 text-[10px] text-muted-foreground font-medium uppercase tracking-wider">
                  {t("loops.curlExample")}
                </div>
                <pre suppressHydrationWarning className="pt-7 pb-3 px-3 bg-muted/50 rounded-lg text-xs font-mono overflow-x-auto text-foreground/70 leading-relaxed">
{`curl -X POST \\
  ${typeof window !== "undefined" ? window.location.origin : ""}/api/v1/ext/orgs/${orgSlug}/loops/${loop.slug}/trigger \\
  -H "X-API-Key: amk_your_api_key_here" \\
  -H "Content-Type: application/json"`}
                </pre>
              </div>
              <p className="text-[10px] text-muted-foreground mt-2">
                {t("loops.apiKeyHint")}
              </p>
              <Link
                href={`/${orgSlug}/settings`}
                className="inline-flex items-center gap-1 text-[10px] text-primary hover:underline mt-1"
              >
                {t("loops.manageApiKeys")}
                <ExternalLink className="w-2.5 h-2.5" />
              </Link>
            </div>
          </section>
        </div>

        {/* Run History */}
        <section className="mb-8">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold">{t("loops.runHistory")}</h2>
            {runs.length > 0 && (
              <span className="text-xs text-muted-foreground tabular-nums">
                {runsTotalCount} {t("loops.totalLabel")}
              </span>
            )}
          </div>
          <div className="border rounded-xl p-3">
            <LoopRunHistory
              runs={runs}
              loading={runsLoading}
              total={runsTotalCount}
              onLoadMore={handleLoadMore}
              onViewTerminal={handleViewTerminal}
              onCancel={handleCancelRun}
            />
          </div>
        </section>

        {/* Edit Dialog */}
        <LoopCreateDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          onCreated={() => {
            // Detail page always uses edit mode — refresh current loop
            setEditOpen(false);
            fetchLoop(slug);
          }}
          editLoop={loop}
        />

        <ConfirmDialog {...deleteDialog.dialogProps} />
      </div>
    </div>
  );
}

// --- Sub-components ---

function StatCard({
  icon: Icon,
  iconColor,
  label,
  value,
  suffix,
  note,
}: {
  icon: React.ElementType;
  iconColor?: string;
  label: string;
  value: string;
  suffix?: string;
  note?: string;
}) {
  return (
    <div className="border rounded-xl p-4 bg-card">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
        <Icon className={cn("w-3.5 h-3.5", iconColor)} />
        {label}
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-2xl font-bold tabular-nums tracking-tight">{value}</span>
        {suffix && (
          <span className="text-sm font-medium text-muted-foreground">{suffix}</span>
        )}
      </div>
      {note && (
        <p className="text-[10px] text-muted-foreground/60 mt-1">{note}</p>
      )}
    </div>
  );
}

function ConfigRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="flex items-center gap-2 text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="font-medium capitalize">{value}</span>
    </div>
  );
}
