"use client";

import { useState, useEffect, useCallback, lazy, Suspense } from "react";
import Link from "next/link";
import { useTranslations } from "@/lib/i18n/client";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { TicketData, TicketStatus, TicketPriority, TicketType } from "@/lib/api/ticket";
import { ticketApi } from "@/lib/api/client";
import {
  ExternalLink,
  Calendar,
  User,
  Tag,
  GitBranch,
  Clock,
  AlertCircle,
} from "lucide-react";

// Lazy load BlockViewer to avoid SSR issues
const BlockViewer = lazy(() =>
  import("@/components/ui/block-editor").then((mod) => ({ default: mod.BlockViewer }))
);

export interface TicketDetailPanelProps {
  ticketIdentifier: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onTicketUpdated?: (ticket: TicketData) => void;
}

const statusConfig: Record<TicketStatus, { label: string; color: string; bg: string }> = {
  backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bg: "bg-gray-100 dark:bg-gray-800" },
  todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bg: "bg-blue-100 dark:bg-blue-900/30" },
  in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bg: "bg-purple-100 dark:bg-purple-900/30" },
  done: { label: "Done", color: "text-green-600 dark:text-green-400", bg: "bg-green-100 dark:bg-green-900/30" },
  cancelled: { label: "Cancelled", color: "text-red-600 dark:text-red-400", bg: "bg-red-100 dark:bg-red-900/30" },
};

const priorityConfig: Record<TicketPriority, { label: string; color: string; icon: string }> = {
  none: { label: "None", color: "text-gray-400 dark:text-gray-500", icon: "—" },
  low: { label: "Low", color: "text-green-500 dark:text-green-400", icon: "↓" },
  medium: { label: "Medium", color: "text-yellow-500 dark:text-yellow-400", icon: "→" },
  high: { label: "High", color: "text-orange-500 dark:text-orange-400", icon: "↑" },
  urgent: { label: "Urgent", color: "text-red-500 dark:text-red-400", icon: "⚡" },
};

const typeConfig: Record<string, { label: string; color: string; icon: string }> = {
  task: { label: "Task", color: "text-blue-500 dark:text-blue-400", icon: "✓" },
  bug: { label: "Bug", color: "text-red-500 dark:text-red-400", icon: "🐛" },
  feature: { label: "Feature", color: "text-green-500 dark:text-green-400", icon: "✨" },
  improvement: { label: "Improvement", color: "text-cyan-500 dark:text-cyan-400", icon: "📈" },
  epic: { label: "Epic", color: "text-purple-500 dark:text-purple-400", icon: "⚡" },
};

const statusOptions: { value: TicketStatus; label: string }[] = [
  { value: "backlog", label: "Backlog" },
  { value: "todo", label: "To Do" },
  { value: "in_progress", label: "In Progress" },
  { value: "in_review", label: "In Review" },
  { value: "done", label: "Done" },
  { value: "cancelled", label: "Cancelled" },
];

const priorityOptions: { value: TicketPriority; label: string }[] = [
  { value: "urgent", label: "Urgent" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
  { value: "none", label: "None" },
];

export function TicketDetailPanel({
  ticketIdentifier,
  open,
  onOpenChange,
  onTicketUpdated,
}: TicketDetailPanelProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const [ticket, setTicket] = useState<TicketData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updating, setUpdating] = useState(false);

  // Load ticket when identifier changes
  useEffect(() => {
    if (!ticketIdentifier || !open) {
      setTicket(null);
      return;
    }

    const loadTicket = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await ticketApi.get(ticketIdentifier);
        setTicket(data);
      } catch (err: unknown) {
        console.error("Failed to load ticket:", err);
        setError(err instanceof Error ? err.message : "Failed to load ticket");
      } finally {
        setLoading(false);
      }
    };

    loadTicket();
  }, [ticketIdentifier, open]);

  const handleStatusChange = useCallback(
    async (newStatus: TicketStatus) => {
      if (!ticket) return;

      setUpdating(true);
      try {
        await ticketApi.updateStatus(ticket.identifier, newStatus);
        const updatedTicket = { ...ticket, status: newStatus };
        setTicket(updatedTicket);
        onTicketUpdated?.(updatedTicket);
      } catch (err: unknown) {
        console.error("Failed to update status:", err);
      } finally {
        setUpdating(false);
      }
    },
    [ticket, onTicketUpdated]
  );

  const handlePriorityChange = useCallback(
    async (newPriority: TicketPriority) => {
      if (!ticket) return;

      setUpdating(true);
      try {
        await ticketApi.update(ticket.identifier, { priority: newPriority });
        const updatedTicket = { ...ticket, priority: newPriority };
        setTicket(updatedTicket);
        onTicketUpdated?.(updatedTicket);
      } catch (err: unknown) {
        console.error("Failed to update priority:", err);
      } finally {
        setUpdating(false);
      }
    },
    [ticket, onTicketUpdated]
  );

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg overflow-y-auto">
        {loading && (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        )}

        {error && (
          <div className="flex flex-col items-center justify-center h-32 text-destructive">
            <AlertCircle className="h-8 w-8 mb-2" />
            <p className="text-sm">{error}</p>
          </div>
        )}

        {ticket && !loading && (
          <>
            <SheetHeader className="pr-8">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className={typeConfig[ticket.type]?.color}>
                  {typeConfig[ticket.type]?.icon}
                </span>
                <code className="text-primary">{ticket.identifier}</code>
                <Link
                  href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
                  className="ml-auto hover:text-primary"
                  title="Open full page"
                >
                  <ExternalLink className="h-4 w-4" />
                </Link>
              </div>
              <SheetTitle className="text-xl leading-tight">
                {ticket.title}
              </SheetTitle>
            </SheetHeader>

            <div className="mt-6 space-y-6">
              {/* Status & Priority */}
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground">
                    {t("tickets.filters.status")}
                  </label>
                  <Select
                    value={ticket.status}
                    onValueChange={(val) => handleStatusChange(val as TicketStatus)}
                    disabled={updating}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {t(`tickets.status.${opt.value}`)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground">
                    {t("tickets.filters.priority")}
                  </label>
                  <Select
                    value={ticket.priority}
                    onValueChange={(val) => handlePriorityChange(val as TicketPriority)}
                    disabled={updating}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {priorityOptions.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {t(`tickets.priority.${opt.value}`)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Description / Summary */}
              {ticket.description && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground">
                    {t("tickets.detail.summary")}
                  </label>
                  <p className="text-sm whitespace-pre-wrap">{ticket.description}</p>
                </div>
              )}

              {/* Content - Rich Text */}
              {ticket.content && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground">
                    {t("tickets.detail.content")}
                  </label>
                  <div className="border border-border rounded-md overflow-hidden bg-card">
                    <Suspense fallback={<div className="h-[100px] animate-pulse bg-muted" />}>
                      <BlockViewer content={ticket.content} />
                    </Suspense>
                  </div>
                </div>
              )}

              {/* Assignees */}
              {ticket.assignees && ticket.assignees.length > 0 && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <User className="h-4 w-4" />
                    {t("tickets.detail.assignees")}
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {ticket.assignees.map((assignee) => (
                      <div
                        key={assignee.id}
                        className="flex items-center gap-2 px-2 py-1 rounded-md bg-muted text-sm"
                      >
                        <div className="w-5 h-5 rounded-full bg-primary/20 flex items-center justify-center text-xs">
                          {(assignee.name || assignee.username)[0].toUpperCase()}
                        </div>
                        {assignee.name || assignee.username}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Labels */}
              {ticket.labels && ticket.labels.length > 0 && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <Tag className="h-4 w-4" />
                    {t("tickets.detail.labels")}
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {ticket.labels.map((label) => (
                      <Badge
                        key={label.id}
                        style={{
                          backgroundColor: `${label.color}20`,
                          color: label.color,
                          borderColor: label.color,
                        }}
                        className="border"
                      >
                        {label.name}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Repository */}
              {ticket.repository && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <GitBranch className="h-4 w-4" />
                    {t("tickets.detail.repository")}
                  </label>
                  <p className="text-sm">{ticket.repository.name}</p>
                </div>
              )}

              {/* Dates */}
              <div className="space-y-2">
                <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                  <Clock className="h-4 w-4" />
                  {t("tickets.detail.timeline")}
                </label>
                <div className="text-sm space-y-1">
                  <p className="text-muted-foreground">
                    {t("tickets.detail.created")}: {formatDate(ticket.created_at)}
                  </p>
                  <p className="text-muted-foreground">
                    {t("tickets.detail.updated")}: {formatDate(ticket.updated_at)}
                  </p>
                  {ticket.due_date && (
                    <p className={new Date(ticket.due_date) < new Date() ? "text-destructive" : ""}>
                      {t("tickets.detail.dueDate")}: {formatDate(ticket.due_date)}
                    </p>
                  )}
                  {ticket.started_at && (
                    <p className="text-muted-foreground">
                      {t("tickets.detail.started")}: {formatDate(ticket.started_at)}
                    </p>
                  )}
                  {ticket.completed_at && (
                    <p className="text-muted-foreground">
                      {t("tickets.detail.completed")}: {formatDate(ticket.completed_at)}
                    </p>
                  )}
                </div>
              </div>

              {/* Actions */}
              <div className="pt-4 border-t">
                <Link href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}>
                  <Button variant="outline" className="w-full">
                    <ExternalLink className="h-4 w-4 mr-2" />
                    {t("tickets.detail.viewFullDetails")}
                  </Button>
                </Link>
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

export default TicketDetailPanel;
