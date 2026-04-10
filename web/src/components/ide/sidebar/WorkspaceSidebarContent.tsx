"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Terminal, Loader2, Plus, RefreshCw, Search, ChevronDown } from "lucide-react";
import { useTranslations } from "next-intl";
import { PodListItem } from "./PodListItem";
import { RenameDialog } from "@/components/shared/RenameDialog";
import { RunnerSection } from "./RunnerSection";
import { WorkspaceFilters } from "./WorkspaceFilters";
import { useWorkspaceSidebar } from "./useWorkspaceSidebar";

interface WorkspaceSidebarContentProps {
  className?: string;
  onCreatePod?: () => void;
  onTerminatePod?: () => void;
}

export function WorkspaceSidebarContent({ className, onCreatePod, onTerminatePod }: WorkspaceSidebarContentProps) {
  const t = useTranslations();
  const s = useWorkspaceSidebar(t, onTerminatePod);

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input placeholder={t("workspace.searchPlaceholder")} value={s.searchQuery}
            onChange={(e) => s.setSearchQuery(e.target.value)} className="pl-8 h-8 text-sm" />
        </div>
      </div>

      <div className="flex items-center gap-1 px-2 pb-2">
        <Button size="sm" variant="outline" className="flex-1 h-8 text-xs" onClick={onCreatePod}>
          <Plus className="w-3 h-3 mr-1" />{t("workspace.newPod")}
        </Button>
        <Button size="sm" variant="ghost" className="h-8 w-8 p-0" onClick={s.handleRefresh} disabled={s.refreshing}>
          <RefreshCw className={cn("w-4 h-4", s.refreshing && "animate-spin")} />
        </Button>
      </div>

      <WorkspaceFilters filter={s.filter} onFilterChange={s.handleFilterChange} t={t} />

      <div className="flex-1 overflow-y-auto">
        {s.loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : s.sortedPods.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Terminal className="w-8 h-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              {s.searchQuery ? t("workspace.emptyState.noMatch")
                : s.filter === "mine" ? t("workspace.emptyState.title")
                : t("workspace.emptyState.noFiltered", { filter: t(`workspace.filters.${s.filter}`) })}
            </p>
            {!s.searchQuery && s.filter === "mine" && (
              <Button size="sm" variant="outline" className="mt-3" onClick={onCreatePod}>
                {t("workspace.emptyState.createFirst")}
              </Button>
            )}
          </div>
        ) : (
          <div className="py-1">
            {s.sortedPods.map((pod) => (
              <PodListItem key={pod.pod_key} pod={pod} isOpen={s.isPodOpen(pod.pod_key)}
                onClick={() => s.handleOpenTerminal(pod)} onTerminate={() => s.handleTerminateClick(pod.pod_key)}
                onRename={() => s.setRenamePod(pod)}
                onTogglePerpetual={(perpetual) => s.handleTogglePerpetual(pod.pod_key, perpetual)} />
            ))}
            {s.podHasMore && (
              <div className="px-3 py-2">
                <Button size="sm" variant="ghost" className="w-full h-8 text-xs text-muted-foreground"
                  onClick={s.loadMorePods} disabled={s.loadingMore}>
                  {s.loadingMore ? <Loader2 className="w-3 h-3 mr-1 animate-spin" /> : <ChevronDown className="w-3 h-3 mr-1" />}
                  {t("workspace.loadMore")}
                </Button>
              </div>
            )}
          </div>
        )}
      </div>

      <RunnerSection runners={s.runners} loading={s.runnersLoading} expanded={s.runnersExpanded}
        onToggle={s.setRunnersExpanded} currentOrgSlug={s.currentOrg?.slug} t={t} />

      <RenameDialog open={s.renamePod !== null} onOpenChange={(open) => { if (!open) s.setRenamePod(null); }}
        currentName={s.renamePod?.alias || ""} onConfirm={s.handleRenameConfirm} />

      <ConfirmDialog {...s.dialogProps} />
    </div>
  );
}

export default WorkspaceSidebarContent;
