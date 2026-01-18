"use client";

import { useEffect, useState, useCallback, lazy, Suspense } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "@/lib/i18n/client";
import { useAuthStore } from "@/stores/auth";
import { useTicketStore, Ticket, TicketStatus, TicketPriority } from "@/stores/ticket";
import { StatusIcon, TypeIcon, getTypeDisplayInfo } from "./TicketIcons";
import { ticketApi, TicketRelation, TicketCommit } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  X,
  ExternalLink,
  GitBranch,
  Clock,
  Loader2,
  AlertCircle,
  ChevronRight,
  Link as LinkIcon,
  GitCommit,
} from "lucide-react";
import { cn } from "@/lib/utils";
import TicketPodPanel from "./TicketPodPanel";
import { StatusSelect } from "./StatusSelect";
import { PrioritySelect } from "./PrioritySelect";
import { InlineEditableText } from "./InlineEditableText";

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
  const [subTickets, setSubTickets] = useState<Ticket[]>([]);
  const [relations, setRelations] = useState<TicketRelation[]>([]);
  const [commits, setCommits] = useState<TicketCommit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  // Fetch extra data (sub-tickets, relations, commits)
  const fetchExtraData = useCallback(async () => {
    if (!ticket) return;

    try {
      const [subTicketsRes, relationsRes, commitsRes] = await Promise.all([
        ticketApi.getSubTickets(identifier).catch(() => ({ tickets: [] })),
        ticketApi.listRelations(identifier).catch(() => ({ relations: [] })),
        ticketApi.listCommits(identifier).catch(() => ({ commits: [] })),
      ]);

      setSubTickets(subTicketsRes.tickets || []);
      setRelations(relationsRes.relations || []);
      setCommits(commitsRes.commits || []);
    } catch (err) {
      console.error("Failed to fetch extra data:", err);
    }
  }, [ticket, identifier]);

  useEffect(() => {
    if (ticket) {
      fetchExtraData();
    }
  }, [ticket, fetchExtraData]);

  // Handle status change with optimistic update
  const handleStatusChange = useCallback(
    async (newStatus: TicketStatus) => {
      if (!ticket) return;

      const oldStatus = ticket.status;
      // Optimistic update
      setTicket({ ...ticket, status: newStatus });

      try {
        await updateTicketStatus(identifier, newStatus);
      } catch (err: unknown) {
        console.error("Failed to update status:", err);
        // Rollback on failure
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
      // Optimistic update
      setTicket({ ...ticket, priority: newPriority });

      try {
        await updateTicket(identifier, { priority: newPriority });
      } catch (err: unknown) {
        console.error("Failed to update priority:", err);
        // Rollback on failure
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
      // Optimistic update
      setTicket({ ...ticket, title: newTitle });

      try {
        await updateTicket(identifier, { title: newTitle });
      } catch (err: unknown) {
        console.error("Failed to update title:", err);
        // Rollback on failure
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
      // Optimistic update
      setTicket({ ...ticket, description: newDescription });

      try {
        await updateTicket(identifier, { description: newDescription });
      } catch (err: unknown) {
        console.error("Failed to update description:", err);
        // Rollback on failure
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
          {ticket.labels && ticket.labels.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {ticket.labels.map((label) => (
                <Badge
                  key={label.id}
                  style={{
                    backgroundColor: `${label.color}15`,
                    color: label.color,
                  }}
                  className="text-xs font-normal border-0"
                >
                  {label.name}
                </Badge>
              ))}
            </div>
          )}

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

          {/* Sub-tickets */}
          {subTickets.length > 0 && (
            <div className="space-y-2">
              <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
                {t("tickets.detail.subTickets")} ({subTickets.length})
              </label>
              <div className="space-y-1">
                {subTickets.map((subTicket) => (
                  <button
                    key={subTicket.id}
                    className="w-full px-2.5 py-1.5 flex items-center gap-2 hover:bg-muted/50 rounded-md transition-colors text-left group"
                    onClick={() => handleTicketClick(subTicket.identifier)}
                  >
                    <StatusIcon status={subTicket.status} size="sm" />
                    <span className="font-mono text-xs text-muted-foreground">
                      {subTicket.identifier}
                    </span>
                    <span className="flex-1 truncate text-sm">{subTicket.title}</span>
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Relations */}
          {relations.length > 0 && (
            <div className="space-y-2">
              <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider flex items-center gap-1">
                <LinkIcon className="h-3 w-3" />
                {t("tickets.detail.related")}
              </label>
              <div className="space-y-1">
                {relations.map((relation) => {
                  const targetTicket = relation.target_ticket;
                  if (!targetTicket) return null;
                  return (
                    <button
                      key={relation.id}
                      className="w-full px-2.5 py-1.5 flex items-center gap-2 hover:bg-muted/50 rounded-md transition-colors text-left group"
                      onClick={() => handleTicketClick(targetTicket.identifier)}
                    >
                      <span className="text-[10px] text-muted-foreground capitalize bg-muted/70 px-1.5 py-0.5 rounded">
                        {relation.relation_type}
                      </span>
                      <span className="font-mono text-xs text-muted-foreground">
                        {targetTicket.identifier}
                      </span>
                      <span className="flex-1 truncate text-sm">{targetTicket.title}</span>
                      <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Commits */}
          {commits.length > 0 && (
            <div className="space-y-2">
              <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider flex items-center gap-1">
                <GitCommit className="h-3 w-3" />
                {t("tickets.detail.commits")}
              </label>
              <div className="space-y-1">
                {commits.slice(0, 5).map((commit) => (
                  <div key={commit.id} className="px-2.5 py-1.5 rounded-md hover:bg-muted/30 transition-colors">
                    <div className="flex items-start gap-2">
                      <code className="font-mono text-[10px] text-muted-foreground shrink-0">
                        {commit.commit_sha.substring(0, 7)}
                      </code>
                      <div className="flex-1 min-w-0">
                        <p className="truncate text-sm">{commit.commit_message}</p>
                        <p className="text-[11px] text-muted-foreground/70 mt-0.5">
                          {commit.author_name} • {commit.committed_at ? formatDate(commit.committed_at) : "N/A"}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
                {commits.length > 5 && (
                  <Link
                    href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
                    className="block text-xs text-primary hover:underline px-2.5 py-1"
                  >
                    {t("common.viewAll")} ({commits.length})
                  </Link>
                )}
              </div>
            </div>
          )}

          {/* AgentPods */}
          <TicketPodPanel
            ticketIdentifier={identifier}
            ticketTitle={ticket.title}
            ticketDescription={ticket.description}
            ticketId={ticket.id}
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
