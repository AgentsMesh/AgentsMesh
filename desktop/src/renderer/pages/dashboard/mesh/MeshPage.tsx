import { useEffect, useMemo } from "react";
import { CenteredSpinner } from "@/components/ui/spinner";
import { MeshTopology } from "@/components/mesh";
import { useMeshStore, useTopology, type MeshNode } from "@/stores/mesh";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { RefreshCw, Minus, Plus, Maximize2 } from "lucide-react";

export function MeshPage() {
  const t = useTranslations();
  const topology = useTopology();
  const loading = useMeshStore((s) => s.loading);
  const error = useMeshStore((s) => s.error);
  const fetchTopology = useMeshStore((s) => s.fetchTopology);
  const clearError = useMeshStore((s) => s.clearError);

  useEffect(() => {
    fetchTopology();
  }, [fetchTopology]);

  const activePodCount = useMemo(
    () =>
      topology?.nodes.filter((n: MeshNode) => n.status === "running" || n.status === "initializing").length || 0,
    [topology?.nodes],
  );

  const runnerCounts = useMemo(() => {
    const runners = topology?.runners || [];
    const online = runners.filter((r) => r.status === "online").length;
    return { online, total: runners.length };
  }, [topology?.runners]);

  return (
    <div className="flex h-full w-full min-w-0 flex-col overflow-hidden">
      <header className="flex items-center justify-between border-b border-border px-6 py-3.5">
        <h1 className="text-[18px] font-semibold text-foreground">{t("mesh.page.title")}</h1>

        <div className="flex items-center gap-2">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium text-foreground">
            <span className="h-2 w-2 rounded-full bg-success" />
            {t("mesh.page.activePods", { count: activePodCount })}
          </span>

          <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium text-foreground">
            <span aria-hidden="true">🖥</span>
            {t("mesh.page.runnersOnline", { online: runnerCounts.online, total: runnerCounts.total })}
          </span>

          <div className="mx-1 inline-flex items-center rounded-md border border-border bg-muted p-0.5">
            <ZoomButton aria={t("mesh.page.zoomOut")}>
              <Minus className="h-3.5 w-3.5" />
            </ZoomButton>
            <ZoomButton aria={t("mesh.page.fit")} active>
              <Maximize2 className="h-3.5 w-3.5" />
            </ZoomButton>
            <ZoomButton aria={t("mesh.page.zoomIn")}>
              <Plus className="h-3.5 w-3.5" />
            </ZoomButton>
          </div>

          <button
            type="button"
            onClick={() => fetchTopology()}
            className="inline-flex h-7 items-center gap-1.5 rounded-md border border-border bg-background px-2.5 text-xs font-medium text-foreground hover:bg-muted"
          >
            <RefreshCw className={cn("h-3.5 w-3.5", loading && "animate-spin")} />
            {t("mesh.page.refresh")}
          </button>
        </div>
      </header>

      {error && (
        <div className="mx-6 mt-4 flex items-center justify-between rounded-md border border-destructive/30 bg-destructive/10 px-4 py-2 text-sm text-destructive">
          <span>{error}</span>
          <button type="button" onClick={clearError} className="text-xs font-medium hover:underline">
            {t("mesh.page.dismiss")}
          </button>
        </div>
      )}

      <div className="relative flex-1 bg-[#FAFBFC]">
        {loading && !topology && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/50">
            <CenteredSpinner />
          </div>
        )}
        <MeshTopology />

        {loading && topology && (
          <div className="absolute right-4 top-4 inline-flex items-center gap-1.5 rounded-full border border-border bg-background/80 px-3 py-1 text-xs text-muted-foreground">
            <span className="h-2 w-2 animate-pulse rounded-full bg-primary" />
            {t("mesh.page.updating")}
          </div>
        )}
      </div>
    </div>
  );
}

function ZoomButton({
  children, aria, active,
}: {
  children: React.ReactNode;
  aria: string;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      aria-label={aria}
      className={cn(
        "flex h-6 w-7 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:text-foreground",
        active && "bg-background text-foreground shadow-sm",
      )}
    >
      {children}
    </button>
  );
}
