"use client";

import { useEffect, useCallback, useRef, lazy, Suspense, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTicketStore, useCurrentTicket, TicketStatus } from "@/stores/ticket";
import { useTicketExtraData } from "./hooks";
import { LabelsList, CommentsList, SubTicketsList, RelationsList, CommitsList } from "./shared";
import { TicketDetailSidebar } from "./TicketDetailSidebar";
import { InlineEditableText } from "./InlineEditableText";
import { SpawnPodButton } from "./SpawnPodButton";
import { StatusSelect } from "./StatusSelect";

const BlockEditor = lazy(() => import("@/components/ui/block-editor"));

interface TicketDetailProps {
  slug: string;
}

export function TicketDetail({ slug }: TicketDetailProps) {
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useCurrentOrg();

  // Use individual selectors to prevent re-renders from unrelated store changes
  // (e.g., fetchTickets() triggered by WebSocket events sets shared `loading`)
  const currentTicket = useCurrentTicket();
  const fetchTicket = useTicketStore(state => state.fetchTicket);
  const updateTicket = useTicketStore(state => state.updateTicket);
  const updateTicketStatus = useTicketStore(state => state.updateTicketStatus);
  const deleteTicket = useTicketStore(state => state.deleteTicket);
  const setCurrentTicket = useTicketStore(state => state.setCurrentTicket);
  const error = useTicketStore(state => state.error);

  // Local loading state to avoid re-renders from shared store `loading`
  // Derived from whether we've loaded the current slug (avoids setState in effect)
  const [loadedSlug, setLoadedSlug] = useState<string | null>(null);
  const initialLoading = loadedSlug !== slug;

  const { dialogProps, confirm } = useConfirmDialog();
  const { subTickets, relations, commits, comments, addComment, updateComment, deleteComment } = useTicketExtraData(slug, !!currentTicket);

  const contentSaveTimerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    return () => {
      if (contentSaveTimerRef.current) {
        clearTimeout(contentSaveTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    // Clear stale ticket from previous slug so the skeleton shows
    // instead of briefly rendering old ticket data
    setCurrentTicket(null);
    fetchTicket(slug).finally(() => setLoadedSlug(slug));
  }, [slug, fetchTicket, setCurrentTicket]);

  const handleTitleSave = useCallback(async (newTitle: string) => {
    if (!newTitle.trim()) return;
    try {
      await updateTicket(slug, { title: newTitle });
    } catch (err) {
      console.error("Failed to update title:", err);
      throw err;
    }
  }, [slug, updateTicket]);

  const handleContentChange = useCallback((newContent: string) => {
    if (contentSaveTimerRef.current) {
      clearTimeout(contentSaveTimerRef.current);
    }
    contentSaveTimerRef.current = setTimeout(async () => {
      try {
        await updateTicket(slug, { content: newContent });
      } catch (err) {
        console.error("Failed to update content:", err);
      }
    }, 800);
  }, [slug, updateTicket]);

  const handleStatusChange = async (newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(slug, newStatus);
    } catch (err) {
      console.error("Failed to update status:", err);
    }
  };

  const handleRepositoryChange = async (repositoryId: number | null) => {
    try {
      await updateTicket(slug, { repositoryId });
    } catch (err) {
      console.error("Failed to update repository:", err);
    }
  };

  const handleDelete = useCallback(async () => {
    const confirmed = await confirm({
      title: t("tickets.detail.deleteTicket"),
      description: t("tickets.detail.deleteConfirmation", { slug }),
      variant: "destructive",
      confirmText: t("common.delete"),
      cancelText: t("common.cancel"),
    });
    if (confirmed) {
      try {
        await deleteTicket(slug);
        router.push(`/${currentOrg?.slug}/tickets`);
      } catch (err) {
        console.error("Failed to delete ticket:", err);
      }
    }
  }, [confirm, deleteTicket, slug, router, currentOrg, t]);

  if (initialLoading && !currentTicket) {
    return <TicketDetailSkeleton />;
  }

  if (error) {
    return (
      <div className="text-center py-16">
        <div className="text-destructive mb-4 text-sm">{error}</div>
        <Button variant="outline" size="sm" onClick={() => fetchTicket(slug)}>
          {t("tickets.detail.retry")}
        </Button>
      </div>
    );
  }

  if (!currentTicket) {
    return (
      <div className="text-center py-16 text-muted-foreground text-sm">
        {t("tickets.detail.notFound")}
      </div>
    );
  }

  return (
    <div className="flex flex-col lg:flex-row gap-6 lg:gap-8">
      {/* Main Content */}
      <div className="flex-1 min-w-0 space-y-6">
        {/* Header — matches the design's prominent header with meta row + big CTA */}
        <div className="border-b border-border pb-6">
          <div className="flex items-start justify-between gap-6">
            {/* Left: slug + status, title, meta row, labels */}
            <div className="flex-1 min-w-0 space-y-3">
              <div className="flex items-center gap-2.5">
                <span className="font-mono text-[13px] text-muted-foreground">{slug}</span>
                <StatusSelect value={currentTicket.status} onChange={handleStatusChange} showLabel size="sm" />
              </div>
              <InlineEditableText
                value={currentTicket.title}
                onSave={handleTitleSave}
                placeholder={t("tickets.createDialog.titlePlaceholder")}
                className="text-[22px] font-semibold leading-7 text-foreground"
                inputClassName="text-[22px] font-semibold"
              />

              {/* Meta row: Repo / Priority / Due / Opened */}
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs">
                {currentTicket.repository && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-muted-foreground">{t("tickets.detail.repository")}</span>
                    <span className="font-mono font-medium text-foreground">
                      {currentTicket.repository.name}
                    </span>
                  </div>
                )}
                {currentTicket.priority && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-muted-foreground">{t("tickets.detail.priority")}</span>
                    <span className="font-medium text-foreground">
                      {t(`tickets.priority.${currentTicket.priority}`)}
                    </span>
                  </div>
                )}
                {currentTicket.due_date && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-muted-foreground">{t("tickets.detail.dueDate")}</span>
                    <span className="font-medium text-foreground">
                      {new Date(currentTicket.due_date).toLocaleDateString()}
                    </span>
                  </div>
                )}
                {currentTicket.created_at && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-muted-foreground">{t("tickets.detail.opened")}</span>
                    <span className="text-foreground">
                      {new Date(currentTicket.created_at).toLocaleDateString()}
                    </span>
                  </div>
                )}
              </div>

              {currentTicket.labels && currentTicket.labels.length > 0 && (
                <LabelsList labels={currentTicket.labels} compact />
              )}
            </div>

            {/* Right: big Spawn Pod CTA + helper text + secondary actions */}
            <div className="flex shrink-0 flex-col items-end gap-2 w-[260px]">
              <SpawnPodButton
                ticket={currentTicket}
                ticketSlug={slug}
                size="lg"
                className="h-11 w-full gap-2 text-sm font-semibold shadow-sm"
              />
              <p className="text-right text-[11px] text-muted-foreground">
                {currentTicket.repository?.name ?? "—"} · {t("tickets.detail.lastUsedAgent")}
              </p>
              <div className="flex items-center gap-1.5">
                <Button variant="outline" size="sm" className="h-7 px-3 text-xs">
                  {t("common.edit")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 px-3 text-xs"
                  onClick={() => handleStatusChange("done" as TicketStatus)}
                >
                  {t("tickets.detail.markDone")}
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-7 w-7"
                  onClick={handleDelete}
                  aria-label={t("common.more")}
                >
                  ⋯
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="rounded-md border border-border overflow-hidden bg-card shadow-xs min-h-[200px] max-h-[65vh] overflow-y-auto">
          <Suspense fallback={<div className="h-[200px] animate-pulse bg-muted/30 rounded-md" />}>
            <BlockEditor
              key={slug}
              initialContent={currentTicket.content || ""}
              onChange={handleContentChange}
              editable={true}
            />
          </Suspense>
        </div>

        {/* Linked items moved into the right rail per design. Leave hook in place
            so the data is still fetched — render only the comments thread here. */}

        {/* Comments (large screens only — on small screens shown above delete in sidebar) */}
        <div className="hidden lg:block">
          <CommentsList
            comments={comments}
            onAddComment={addComment}
            onUpdateComment={updateComment}
            onDeleteComment={deleteComment}
          />
        </div>

      </div>

      {/* Sidebar — 4-section rail per design */}
      <TicketDetailSidebar
        ticket={currentTicket}
        ticketSlug={slug}
        subTickets={subTickets}
        relations={relations}
        commits={commits}
        t={t}
        commentsSlot={
          <div className="lg:hidden">
            <CommentsList
              comments={comments}
              onAddComment={addComment}
              onUpdateComment={updateComment}
              onDeleteComment={deleteComment}
            />
          </div>
        }
      />

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

function TicketDetailSkeleton() {
  return (
    <div className="animate-pulse" data-testid="ticket-detail-skeleton">
      <div className="flex flex-col lg:flex-row gap-6 lg:gap-8">
        <div className="flex-1 space-y-6">
          <div className="space-y-4">
            <div className="flex items-center gap-2.5">
              <div className="h-5 w-20 bg-muted/60 rounded" />
              <div className="h-5 w-24 bg-muted/60 rounded-full" />
            </div>
            <div className="h-8 bg-muted/60 rounded-lg w-3/4" />
          </div>
          <div className="h-10 bg-muted/40 rounded-lg w-full" />
          <div className="h-64 bg-muted/40 rounded-xl" />
        </div>
        <div className="lg:w-72 shrink-0 space-y-3">
          <div className="h-[52px] bg-muted/50 rounded-xl" />
          <div className="rounded-xl border border-border/40 overflow-hidden">
            <div className="h-12 bg-muted/30" />
            <div className="h-12 bg-muted/20" />
            <div className="h-12 bg-muted/30" />
            <div className="h-16 bg-muted/20" />
            <div className="h-10 bg-muted/30" />
          </div>
          <div className="h-9 bg-muted/30 rounded-lg" />
        </div>
      </div>
    </div>
  );
}

export default TicketDetail;
