"use client";

import { useEffect, useState, useCallback, lazy, Suspense } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, Ticket, TicketStatus, TicketPriority } from "@/stores/ticket";
import { ticketApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  X,
  ExternalLink,
  Clock,
  Loader2,
  AlertCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import TicketPodPanel from "./TicketPodPanel";
import { StatusSelect } from "./StatusSelect";
import { PrioritySelect } from "./PrioritySelect";
import { InlineEditableText } from "./InlineEditableText";
import { useTicketExtraData } from "./hooks";
import { SubTicketsList, RelationsList, CommitsList, LabelsList } from "./shared";
import { RepositorySelect } from "@/components/common/RepositorySelect";

// Lazy load BlockViewer to avoid SSR issues
const BlockViewer = lazy(() =>
  import("@/components/ui/block-editor").then((mod) => ({ default: mod.BlockViewer }))
);

export interface TicketDetailPaneProps {
  slug: string;
  onClose: () => void;
  className?: string;
}

export function TicketDetailPane({ slug, onClose, className }: TicketDetailPaneProps) {
  const t = useTranslations();
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const updateTicket = useTicketStore((s) => s.updateTicket);
  const updateTicketStatus = useTicketStore((s) => s.updateTicketStatus);
  const tickets = useTicketStore((s) => s.tickets);

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Use shared hook for extra data
  const { subTickets, relations, commits } = useTicketExtraData(slug, !!ticket);

  // Load ticket data
  useEffect(() => {
    if (!slug) return;

    // First try to find ticket from store (for instant display)
    const cachedTicket = tickets.find(t => t.slug === slug);
    if (cachedTicket) {
      setTicket(cachedTicket);
    } else {
      // Reset stale ticket from previous slug
      setTicket(null);
    }

    // If ticket was removed from store (e.g., deleted), skip the API call
    // to avoid a 404 request for a ticket that no longer exists
    if (tickets.length > 0 && !cachedTicket) {
      return;
    }

    const loadTicket = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await ticketApi.get(slug);
        setTicket(data);
      } catch (err: unknown) {
        console.error("Failed to load ticket:", err);
        setError(err instanceof Error ? err.message : "Failed to load ticket");
      } finally {
        setLoading(false);
      }
    };

    loadTicket();
  }, [slug, tickets]);

  // Handle status change with optimistic update
  const handleStatusChange = useCallback(
    async (newStatus: TicketStatus) => {
      if (!ticket) return;

      const oldStatus = ticket.status;
      setTicket({ ...ticket, status: newStatus });

      try {
        await updateTicketStatus(slug, newStatus);
      } catch (err: unknown) {
        console.error("Failed to update status:", err);
        setTicket({ ...ticket, status: oldStatus });
        throw err;
      }
    },
    [ticket, slug, updateTicketStatus]
  );

  // Handle priority change with optimistic update
  const handlePriorityChange = useCallback(
    async (newPriority: TicketPriority) => {
      if (!ticket) return;

      const oldPriority = ticket.priority;
      setTicket({ ...ticket, priority: newPriority });

      try {
        await updateTicket(slug, { priority: newPriority });
      } catch (err: unknown) {
        console.error("Failed to update priority:", err);
        setTicket({ ...ticket, priority: oldPriority });
        throw err;
      }
    },
    [ticket, slug, updateTicket]
  );

  // Handle title change with optimistic update
  const handleTitleChange = useCallback(
    async (newTitle: string) => {
      if (!ticket || !newTitle.trim()) return;

      const oldTitle = ticket.title;
      setTicket({ ...ticket, title: newTitle });

      try {
        await updateTicket(slug, { title: newTitle });
      } catch (err: unknown) {
        console.error("Failed to update title:", err);
        setTicket({ ...ticket, title: oldTitle });
        throw err;
      }
    },
    [ticket, slug, updateTicket]
  );

  // Handle repository change with optimistic update
  const handleRepositoryChange = useCallback(
    async (newRepositoryId: number | null) => {
      if (!ticket) return;

      const oldRepositoryId = ticket.repository_id;
      const oldRepository = ticket.repository;
      setTicket({ ...ticket, repository_id: newRepositoryId ?? undefined, repository: undefined });

      try {
        const updated = await updateTicket(slug, { repositoryId: newRepositoryId });
        setTicket(updated);
      } catch (err: unknown) {
        console.error("Failed to update repository:", err);
        setTicket({ ...ticket, repository_id: oldRepositoryId, repository: oldRepository });
        throw err;
      }
    },
    [ticket, slug, updateTicket]
  );

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  // Navigate to sub-ticket or related ticket
  const handleTicketClick = (ticketSlug: string) => {
    router.push(`/${currentOrg?.slug}/tickets?ticket=${ticketSlug}`);
  };

  if (loading && !ticket) {
    return (
      <div className={cn("flex items-center justify-center h-full", className)}>
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error && !ticket) {
    return (
      <div className={cn("flex flex-col items-center justify-center h-full text-destructive", className)}>
        <AlertCircle className="h-8 w-8 mb-2" />
        <p className="text-sm">{error}</p>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className={cn("flex items-center justify-center h-full text-muted-foreground", className)}>
        <p className="text-sm">{t("tickets.detail.notFound")}</p>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col h-full bg-background", className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/40 shrink-0 bg-muted/20">
        <code className="text-xs text-muted-foreground/80 font-mono tracking-wide bg-muted/60 px-2 py-0.5 rounded">
          {ticket.slug}
        </code>
        <div className="flex items-center gap-0.5">
          <Link
            href={`/${currentOrg?.slug}/tickets/${ticket.slug}`}
            className="p-1.5 rounded-md hover:bg-muted/80 text-muted-foreground/60 hover:text-foreground transition-colors"
            title={t("tickets.detail.viewFullDetails")}
          >
            <ExternalLink className="h-3.5 w-3.5" />
          </Link>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-muted/80 text-muted-foreground/60 hover:text-foreground transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-4">
          {/* Title */}
          <InlineEditableText
            value={ticket.title}
            onSave={handleTitleChange}
            placeholder={t("tickets.createDialog.titlePlaceholder")}
            className="text-lg font-bold leading-tight tracking-tight"
            inputClassName="text-lg font-bold tracking-tight"
          />

          {/* Status & Priority */}
          <div className="flex items-center gap-2 flex-wrap">
            <StatusSelect
              value={ticket.status}
              onChange={handleStatusChange}
              size="sm"
            />
            <PrioritySelect
              value={ticket.priority}
              onChange={handlePriorityChange}
              size="sm"
            />
          </div>

          {/* Content */}
          {ticket.content && (
            <div className="space-y-1.5">
              <label className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">
                {t("tickets.detail.content")}
              </label>
              <div className="rounded-lg overflow-hidden bg-muted/20 ring-1 ring-border/30">
                <Suspense fallback={<div className="h-[100px] animate-pulse bg-muted/30 rounded-lg" />}>
                  <BlockViewer key={slug} content={ticket.content} />
                </Suspense>
              </div>
            </div>
          )}

          {/* Labels */}
          <LabelsList labels={ticket.labels || []} compact />

          {/* Assignees */}
          {ticket.assignees && ticket.assignees.length > 0 && (
            <div className="flex items-center gap-2.5">
              <span className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">
                {t("tickets.detail.assignees")}
              </span>
              <div className="flex items-center -space-x-1.5">
                {ticket.assignees.map((assignee) => (
                  <div
                    key={assignee.user_id}
                    className="w-6 h-6 rounded-full bg-primary/15 flex items-center justify-center text-[10px] font-semibold text-primary border-2 border-background ring-1 ring-primary/10"
                    title={assignee.user?.name || assignee.user?.username}
                  >
                    {(assignee.user?.name || assignee.user?.username || "?")[0].toUpperCase()}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Repository */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-medium text-muted-foreground/60 uppercase tracking-wider">
              {t("tickets.detail.repository")}
            </label>
            <RepositorySelect
              value={ticket.repository_id ?? null}
              onChange={handleRepositoryChange}
              placeholder={t("tickets.detail.noRepository")}
              className="text-sm"
            />
          </div>

          {/* Metadata */}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground/60">
            <span className="flex items-center gap-1.5">
              <Clock className="h-3 w-3" />
              {formatDate(ticket.created_at)}
            </span>
            {ticket.due_date && (
              <span className={cn(
                "flex items-center gap-1 font-medium",
                new Date(ticket.due_date) < new Date() ? "text-destructive" : "text-muted-foreground/70"
              )}>
                Due {formatDate(ticket.due_date)}
              </span>
            )}
          </div>

          {/* Divider before linked items */}
          {(subTickets.length > 0 || relations.length > 0 || commits.length > 0) && (
            <div className="border-t border-border/30 pt-1" />
          )}

          <SubTicketsList
            subTickets={subTickets}
            onTicketClick={handleTicketClick}
            compact
          />

          <RelationsList
            relations={relations}
            onTicketClick={handleTicketClick}
            compact
          />

          <CommitsList
            commits={commits}
            viewAllLink={`/${currentOrg?.slug}/tickets/${ticket.slug}`}
            compact
          />

          {/* AgentPods */}
          <TicketPodPanel
            ticketSlug={slug}
            ticketTitle={ticket.title}
            ticketId={ticket.id}
            repositoryId={ticket.repository_id}
          />
        </div>
      </div>

      {/* Footer */}
      <div className="shrink-0 px-4 py-2 border-t border-border/30">
        <Link href={`/${currentOrg?.slug}/tickets/${ticket.slug}`}>
          <Button variant="ghost" size="sm" className="w-full text-muted-foreground/70 hover:text-foreground text-xs">
            <ExternalLink className="h-3 w-3 mr-1.5" />
            {t("tickets.detail.viewFullDetails")}
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default TicketDetailPane;
