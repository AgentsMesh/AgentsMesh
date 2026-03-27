"use client";

import { useEffect, useCallback, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useLoopStore } from "@/stores/loop";
import { CenteredSpinner } from "@/components/ui/spinner";
import { LoopRunHistory } from "@/components/loops/LoopRunHistory";
import { LoopCreateDialog } from "@/components/loops/LoopCreateDialog";
import { Button } from "@/components/ui/button";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { AlertCircle, XCircle, RefreshCw } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { LoopHeader, LoopStatCards, LoopConfigSection } from "./components";

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
  const setCurrentLoop = useLoopStore((s) => s.setCurrentLoop);

  const [editOpen, setEditOpen] = useState(false);
  const [triggering, setTriggering] = useState(false);

  const deleteDialog = useConfirmDialog({
    title: t("loops.deleteConfirm"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => {
    fetchLoop(slug);
    fetchRuns(slug, { limit: 20, offset: 0 });
    return () => { setCurrentLoop(null); };
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
    } catch { toast.error(t("loops.triggerFailed")); }
    finally { setTriggering(false); }
  }, [slug, triggerLoop, t]);

  const handleLoadMore = useCallback(() => { loadMoreRuns(slug); }, [slug, loadMoreRuns]);

  const handleCancelRun = useCallback(async (runId: number) => {
    try { await cancelRun(slug, runId); toast.success(t("loops.runCancelled")); }
    catch { toast.error(t("loops.cancelFailed")); }
  }, [slug, cancelRun, t]);

  const handleViewTerminal = useCallback((podKey: string) => {
    router.push(`/${orgSlug}/workspace?pod=${podKey}`);
  }, [router, orgSlug]);

  const handleEnable = useCallback(async () => {
    try { await enableLoop(slug); toast.success(t("loops.enabled")); }
    catch { toast.error(t("loops.enableFailed")); }
  }, [slug, enableLoop, t]);

  const handleDisable = useCallback(async () => {
    try { await disableLoop(slug); toast.success(t("loops.disabled")); }
    catch { toast.error(t("loops.disableFailed")); }
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

  if (loopLoading && !currentLoop) return <CenteredSpinner className="h-full" />;

  if (error && !currentLoop) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center py-20">
        <div className="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mb-3">
          <AlertCircle className="w-6 h-6 text-destructive" />
        </div>
        <p className="text-sm text-muted-foreground mb-3">{error}</p>
        <Button variant="outline" size="sm" className="gap-1.5" onClick={() => { clearError(); fetchLoop(slug); }}>
          <RefreshCw className="w-3.5 h-3.5" />{t("loops.retry")}
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

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-5">
        <LoopHeader loop={currentLoop} orgSlug={orgSlug} triggering={triggering} t={t}
          onBack={() => router.push(`/${orgSlug}/loops`)} onTrigger={handleTrigger}
          onEdit={() => setEditOpen(true)} onEnable={handleEnable}
          onDisable={handleDisable} onDelete={handleDelete} />
        <LoopStatCards loop={currentLoop} t={t} />
        <LoopConfigSection loop={currentLoop} orgSlug={orgSlug} t={t} />

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
            <LoopRunHistory runs={runs} loading={runsLoading} total={runsTotalCount}
              onLoadMore={handleLoadMore} onViewTerminal={handleViewTerminal} onCancel={handleCancelRun} />
          </div>
        </section>

        <LoopCreateDialog open={editOpen} onOpenChange={setEditOpen}
          onCreated={() => { setEditOpen(false); fetchLoop(slug); }} editLoop={currentLoop} />
        <ConfirmDialog {...deleteDialog.dialogProps} />
      </div>
    </div>
  );
}
