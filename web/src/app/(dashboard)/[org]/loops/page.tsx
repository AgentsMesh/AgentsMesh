"use client";

import { useEffect, useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { useLoopStore, LoopData } from "@/stores/loop";
import { useAuthStore } from "@/stores/auth";
import { CenteredSpinner } from "@/components/ui/spinner";
import { LoopCard } from "@/components/loops/LoopCard";
import { LoopCreateDialog } from "@/components/loops/LoopCreateDialog";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Plus, Repeat, AlertCircle, RefreshCw } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

export default function LoopsPage() {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const loops = useLoopStore((s) => s.loops);
  const loading = useLoopStore((s) => s.loading);
  const error = useLoopStore((s) => s.error);
  const fetchLoops = useLoopStore((s) => s.fetchLoops);
  const triggerLoop = useLoopStore((s) => s.triggerLoop);
  const enableLoop = useLoopStore((s) => s.enableLoop);
  const disableLoop = useLoopStore((s) => s.disableLoop);
  const deleteLoop = useLoopStore((s) => s.deleteLoop);
  const clearError = useLoopStore((s) => s.clearError);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editLoop, setEditLoop] = useState<LoopData | null>(null);
  const [triggeringSlug, setTriggeringSlug] = useState<string | null>(null);

  const deleteDialog = useConfirmDialog({
    title: t("loops.deleteConfirm"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => {
    fetchLoops();
  }, [fetchLoops]);

  const handleClick = useCallback(
    (slug: string) => {
      router.push(`/${currentOrg?.slug}/loops/${slug}`);
    },
    [router, currentOrg]
  );

  const handleTrigger = useCallback(
    async (slug: string) => {
      setTriggeringSlug(slug);
      try {
        const result = await triggerLoop(slug);
        if (result.skipped) {
          toast.info(t("loops.triggerSkipped"), {
            description: result.reason,
          });
        } else if (result.run) {
          toast.success(t("loops.triggered"), {
            description: `Run #${result.run.run_number}`,
          });
        }
      } catch (err) {
        toast.error(t("loops.triggerFailed"), { description: (err as Error).message });
      } finally {
        setTriggeringSlug(null);
      }
    },
    [triggerLoop, t]
  );

  const handleEnable = useCallback(
    async (slug: string) => {
      try {
        await enableLoop(slug);
        toast.success(t("loops.enabled"));
      } catch {
        toast.error(t("loops.enableFailed"));
      }
    },
    [enableLoop, t]
  );

  const handleDisable = useCallback(
    async (slug: string) => {
      try {
        await disableLoop(slug);
        toast.success(t("loops.disabled"));
      } catch {
        toast.error(t("loops.disableFailed"));
      }
    },
    [disableLoop, t]
  );

  const handleDelete = useCallback(
    async (slug: string) => {
      const confirmed = await deleteDialog.confirm();
      if (!confirmed) return;
      try {
        await deleteLoop(slug);
        toast.success(t("loops.deleted"));
      } catch (err) {
        toast.error(t("loops.deleteFailed"), { description: (err as Error).message });
      }
    },
    [deleteLoop, deleteDialog, t]
  );

  const handleEdit = useCallback((loop: LoopData) => {
    setEditLoop(loop);
    setCreateDialogOpen(true);
  }, []);

  const handleCreated = useCallback((createdLoop?: import("@/lib/api/loop").LoopData) => {
    setCreateDialogOpen(false);
    setEditLoop(null);
    if (createdLoop) {
      // Navigate to the newly created loop's detail page
      router.push(`/${currentOrg?.slug}/loops/${createdLoop.slug}`);
    } else {
      // Edit mode — refresh list
      fetchLoops();
    }
  }, [fetchLoops, router, currentOrg]);

  if (loading && loops.length === 0) {
    return <CenteredSpinner className="h-full" />;
  }

  if (error && loops.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center py-20">
        <div className="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mb-3">
          <AlertCircle className="w-6 h-6 text-destructive" />
        </div>
        <p className="text-sm text-muted-foreground mb-3">{error}</p>
        <Button variant="outline" size="sm" className="gap-1.5" onClick={() => { clearError(); fetchLoops(); }}>
          <RefreshCw className="w-3.5 h-3.5" />
          {t("loops.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-5">
        {/* Empty state */}
        {loops.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mb-5">
              <Repeat className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">{t("loops.emptyTitle")}</h3>
            <p className="text-sm text-muted-foreground mb-6 max-w-md leading-relaxed">
              {t("loops.emptyDescription")}
            </p>
            <Button onClick={() => setCreateDialogOpen(true)} className="gap-1.5">
              <Plus className="w-4 h-4" />
              {t("loops.createFirstLoop")}
            </Button>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-end mb-3">
              <Button size="sm" onClick={() => setCreateDialogOpen(true)} className="gap-1.5">
                <Plus className="w-3.5 h-3.5" />
                {t("loops.createLoop")}
              </Button>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            {loops.map((loop) => (
              <LoopCard
                key={loop.id}
                loop={loop}
                onClick={handleClick}
                onTrigger={handleTrigger}
                onEnable={handleEnable}
                onDisable={handleDisable}
                onEdit={handleEdit}
                onDelete={handleDelete}
                triggering={triggeringSlug === loop.slug}
              />
            ))}
            </div>
          </>
        )}

        <LoopCreateDialog
          open={createDialogOpen}
          onOpenChange={(open) => {
            setCreateDialogOpen(open);
            if (!open) setEditLoop(null);
          }}
          onCreated={handleCreated}
          editLoop={editLoop || undefined}
        />

        <ConfirmDialog {...deleteDialog.dialogProps} />
      </div>
    </div>
  );
}
