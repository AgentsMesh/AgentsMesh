import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useLoopStore, useLoops } from "@/stores/loop";
import { CenteredSpinner } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { Button } from "@/components/ui/button";
import { AlertCircle, RefreshCw, Repeat, Plus } from "lucide-react";
import { useTranslations } from "next-intl";
import { LoopCreateDialog } from "@/components/loops/LoopCreateDialog";

export function LoopsPage() {
  const t = useTranslations();
  const navigate = useNavigate();
  const params = useParams<{ org: string }>();
  const orgSlug = params.org;
  const loops = useLoops();
  const loading = useLoopStore((s) => s.loading);
  const error = useLoopStore((s) => s.error);
  const fetchLoops = useLoopStore((s) => s.fetchLoops);
  const clearError = useLoopStore((s) => s.clearError);
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    fetchLoops();
  }, [fetchLoops]);

  useEffect(() => {
    if (loading || loops.length === 0 || !orgSlug) return;
    const first = loops.find((l) => l.status === "enabled") ?? loops[0];
    if (first) navigate(`/${orgSlug}/loops/${first.slug}`, { replace: true });
  }, [loops, loading, orgSlug, navigate]);

  if (error && loops.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 py-20 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-md bg-destructive/10">
          <AlertCircle className="h-6 w-6 text-destructive" />
        </div>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button variant="outline" size="sm" className="gap-1.5"
          onClick={() => { clearError(); fetchLoops(); }}>
          <RefreshCw className="h-3.5 w-3.5" />
          {t("loops.retry")}
        </Button>
      </div>
    );
  }

  if (loading && loops.length === 0) return <CenteredSpinner className="h-full" />;

  if (loops.length === 0) {
    return (
      <>
        <EmptyState
          size="full"
          icon={<Repeat className="h-12 w-12" />}
          title={t("loops.emptyTitle")}
          description={t("loops.emptyDescription")}
          actions={
            <Button onClick={() => setCreateOpen(true)} className="gap-1.5">
              <Plus className="h-4 w-4" />
              {t("loops.createFirstLoop")}
            </Button>
          }
        />
        <LoopCreateDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onCreated={() => {
            setCreateOpen(false);
            fetchLoops();
          }}
        />
      </>
    );
  }

  return <CenteredSpinner className="h-full" />;
}
