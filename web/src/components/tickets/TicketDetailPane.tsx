"use client";

import { useEffect, useState, useCallback, lazy, Suspense } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, Ticket, TicketStatus, TicketPriority } from "@/stores/ticket";
import { TypeIcon, getTypeDisplayInfo } from "./TicketIcons";
import { ticketApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  X,
  ExternalLink,
  GitBranch,
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

// Lazy load BlockViewer to avoid SSR issues
const BlockViewer = lazy(() =>
  import("@/components/ui/block-editor").then((mod) => ({ default: mod.BlockViewer }))
);

export interface TicketDetailPaneProps {
  identifier: string;
  onClose: () => void;
  className?: string;
}

export function TicketDetailPane({ identifier, onClose, className }: TicketDetailPaneProps) {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const { updateTicket, updateTicketStatus, tickets } = useTicketStore();

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Use shared hook for extra data
  const { subTickets, relations, commits } = useTicketExtraData(identifier, !!ticket);

  // Load ticket data
  useEffect(() => {
    if (!identifier) return;

    // First try to find ticket from store (for instant display)
    const cachedTicket = tickets.find(t => t.identifier === identifier);
    if (cachedTicket) {
      setTicket(cachedTicket);
    }

    const loadTicket = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await ticketApi.get(identifier);
        setTicket(data);
      } catch (err: unknown) {
        console.error("Failed to load ticket:", err);
        setError(err instanceof Error ? err.message : "Failed to load ticket");
      } finally {
        setLoading(false);
      }
    };

    loadTicket();
  }, [identifier, tickets]);

  // Handle status change with optimistic update
  const handleStatusChange = useCallback(
    async (newStatus: TicketStatus) => {
      if (!ticket) return;

      const oldStatus = ticket.status;
      setTicket({ ...ticket, status: newStatus });

      try {
        await updateTicketStatus(identifier, newStatus);
      } catch (err: unknown) {
        console.error("Failed to update status:", err);
        setTicket({ ...ticket, status: oldStatus });
        throw err;
      }
    },
    [ticket, identifier, updateTicketStatus]
  );

  // Handle priority change with optimistic update
  const handlePriorityChange = useCallback(
    async (newPriority: TicketPriority) => {
      if (!ticket) return;

      const oldPriority = ticket.priority;
      setTicket({ ...ticket, priority: newPriority });

      try {
        await updateTicket(identifier, { priority: newPriority });
      } catch (err: unknown) {
        console.error("Failed to update priority:", err);
        setTicket({ ...ticket, priority: oldPriority });
        throw err;
      }
    },
    [ticket, identifier, updateTicket]
  );

  // Handle title change with optimistic update
  const handleTitleChange = useCallback(
    async (newTitle: string) => {
      if (!ticket || !newTitle.trim()) return;

      const oldTitle = ticket.title;
      setTicket({ ...ticket, title: newTitle });

      try {
        await updateTicket(identifier, { title: newTitle });
      } catch (err: unknown) {
        console.error("Failed to update title:", err);
        setTicket({ ...ticket, title: oldTitle });
        throw err;
      }
    },
    [ticket, identifier, updateTicket]
  );

  // Handle description change with optimistic update
  const handleDescriptionChange = useCallback(
    async (newDescription: string) => {
      if (!ticket) return;

      const oldDescription = ticket.description;
      setTicket({ ...ticket, description: newDescription });

      try {
        await updateTicket(identifier, { description: newDescription });
      } catch (err: unknown) {
        console.error("Failed to update description:", err);
        setTicket({ ...ticket, description: oldDescription });
        throw err;
      }
    },
    [ticket, identifier, updateTicket]
  );

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  // Navigate to sub-ticket or related ticket
  const handleTicketClick = (ticketIdentifier: string) => {
    router.push(`/${currentOrg?.slug}/tickets?ticket=${ticketIdentifier}`);
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

  const typeInfo = getTypeDisplayInfo(ticket.type);

  return (
    <div className={cn("flex flex-col h-full bg-background", className)}>
      {/* Header - Clean minimal style */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/50 shrink-0 bg-muted/30">
        <div className="flex items-center gap-2 min-w-0">
          <code className="text-sm text-muted-foreground font-mono">{ticket.identifier}</code>
        </div>
        <div className="flex items-center gap-0.5">
          <Link
            href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
            className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title={t("tickets.detail.viewFullDetails")}
          >
            <ExternalLink className="h-4 w-4" />
          </Link>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-5">
          {/* Title - Inline Editable */}
          <InlineEditableText
            value={ticket.title}
            onSave={handleTitleChange}
            placeholder={t("tickets.createDialog.titlePlaceholder")}
            className="text-lg font-semibold leading-tight"
            inputClassName="text-lg font-semibold"
          />

          {/* Status & Priority - Linear Style inline */}
          <div className="flex items-center gap-3 flex-wrap">
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
            <div className={cn("flex items-center gap-1.5 px-2 py-1 rounded-md text-sm", typeInfo.bgColor, typeInfo.color)}>
              <TypeIcon type={ticket.type} size="sm" />
              <span>{t(`tickets.type.${ticket.type}`)}</span>
            </div>
          </div>

          {/* Description / Summary - Inline Editable */}
          <div className="space-y-1">
            <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
              {t("tickets.detail.summary")}
            </label>
            <InlineEditableText
              value={ticket.description || ""}
              onSave={handleDescriptionChange}
              placeholder={t("tickets.createDialog.summaryPlaceholder")}
              className="text-sm text-muted-foreground"
              multiline
              debounceMs={1000}
            />
          </div>

          {/* Content - Rich Text (read-only in panel, edit on full page) */}
          {ticket.content && (
            <div className="space-y-1">
              <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
                {t("tickets.detail.content")}
              </label>
              <div className="rounded-lg overflow-hidden bg-muted/30">
                <Suspense fallback={<div className="h-[100px] animate-pulse bg-muted" />}>
                  <BlockViewer content={ticket.content} />
                </Suspense>
              </div>
            </div>
          )}

          {/* Labels */}
          <LabelsList labels={ticket.labels || []} compact />

          {/* Assignees - Compact inline */}
          {ticket.assignees && ticket.assignees.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
                {t("tickets.detail.assignees")}
              </span>
              <div className="flex items-center -space-x-1">
                {ticket.assignees.map((assignee) => (
                  <div
                    key={assignee.id}
                    className="w-6 h-6 rounded-full bg-primary/20 flex items-center justify-center text-xs border-2 border-background"
                    title={assignee.name || assignee.username}
                  >
                    {(assignee.name || assignee.username)[0].toUpperCase()}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Details - Compact metadata row */}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
            {ticket.repository && (
              <span className="flex items-center gap-1">
                <GitBranch className="h-3 w-3" />
                {(ticket.repository as { name: string }).name}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {formatDate(ticket.created_at)}
            </span>
            {ticket.due_date && (
              <span className={cn("flex items-center gap-1", new Date(ticket.due_date) < new Date() && "text-destructive")}>
                Due {formatDate(ticket.due_date)}
              </span>
            )}
          </div>

          {/* Sub-tickets (using shared component) */}
          <SubTicketsList
            subTickets={subTickets}
            onTicketClick={handleTicketClick}
            compact
          />

          {/* Relations (using shared component) */}
          <RelationsList
            relations={relations}
            onTicketClick={handleTicketClick}
            compact
          />

          {/* Commits (using shared component) */}
          <CommitsList
            commits={commits}
            viewAllLink={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
            compact
          />

          {/* AgentPods */}
          <TicketPodPanel
            ticketIdentifier={identifier}
            ticketTitle={ticket.title}
            ticketDescription={ticket.description}
            ticketId={ticket.id}
            repositoryId={ticket.repository_id}
          />
        </div>
      </div>

      {/* Footer Actions */}
      <div className="shrink-0 px-4 py-2.5 border-t border-border/50 bg-muted/20">
        <Link href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}>
          <Button variant="ghost" size="sm" className="w-full text-muted-foreground hover:text-foreground">
            <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
            {t("tickets.detail.viewFullDetails")}
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default TicketDetailPane;
