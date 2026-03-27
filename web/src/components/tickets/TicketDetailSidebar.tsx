"use client";

import { Ticket, TicketStatus } from "@/stores/ticket";
import { Button } from "@/components/ui/button";
import { StatusSelect } from "./StatusSelect";
import { RepositorySelect } from "@/components/common/RepositorySelect";
import { SidebarPodSection } from "./SidebarPodSection";
import { Trash2, Clock } from "lucide-react";
import { cn } from "@/lib/utils";

interface TicketDetailSidebarProps {
  ticket: Ticket;
  onDelete: () => void;
  onStatusChange: (status: TicketStatus) => void;
  onRepositoryChange: (repositoryId: number | null) => void;
  ticketSlug: string;
  t: (key: string, params?: Record<string, string | number>) => string;
  commentsSlot?: React.ReactNode;
}

function formatRelativeDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffDay > 30) return date.toLocaleDateString();
  if (diffDay > 0) return `${diffDay}d ago`;
  if (diffHr > 0) return `${diffHr}h ago`;
  if (diffMin > 0) return `${diffMin}m ago`;
  return "just now";
}

export function TicketDetailSidebar({
  ticket, onDelete, onStatusChange, onRepositoryChange, ticketSlug, t, commentsSlot,
}: TicketDetailSidebarProps) {
  const handleStatusChange = async (status: TicketStatus) => {
    onStatusChange(status);
  };

  return (
    <div className="lg:w-72 shrink-0 space-y-3">
      <SidebarPodSection ticket={ticket} ticketSlug={ticketSlug} />

      <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
        {/* Status */}
        <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">{t("tickets.filters.status")}</span>
            <StatusSelect value={ticket.status} onChange={handleStatusChange} showLabel size="sm" />
          </div>
        </div>

        <div className="mx-4 border-t border-border/40" />

        {/* Repository */}
        <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
          <span className="text-xs font-medium text-muted-foreground block mb-2">{t("tickets.detail.repository")}</span>
          <RepositorySelect value={ticket.repository_id ?? null} onChange={onRepositoryChange}
            placeholder={t("tickets.detail.noRepository")} className="text-sm" />
        </div>

        <div className="mx-4 border-t border-border/40" />

        {/* Due Date */}
        {ticket.due_date && (
          <>
            <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">{t("tickets.detail.dueDate")}</span>
                <span className={cn("text-sm tabular-nums",
                  new Date(ticket.due_date) < new Date() && ticket.status !== "done"
                    ? "text-destructive font-medium" : "text-foreground")}>
                  {new Date(ticket.due_date).toLocaleDateString()}
                </span>
              </div>
            </div>
            <div className="mx-4 border-t border-border/40" />
          </>
        )}

        {/* Assignees */}
        <AssigneesSection assignees={ticket.assignees} t={t} />

        <div className="mx-4 border-t border-border/40" />

        {/* Timestamps */}
        <div className="px-4 py-3">
          <div className="flex flex-col gap-1 text-xs text-muted-foreground/70">
            <div className="flex items-center gap-1.5">
              <Clock className="w-3 h-3 shrink-0" />
              <span title={new Date(ticket.created_at).toLocaleString()}>
                {t("tickets.detail.created")} {formatRelativeDate(ticket.created_at)}
              </span>
            </div>
            <div className="flex items-center gap-1.5 ml-[18px]">
              <span title={new Date(ticket.updated_at).toLocaleString()}>
                {t("tickets.detail.updated")} {formatRelativeDate(ticket.updated_at)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {commentsSlot}

      <Button className="w-full" variant="outline" size="sm" onClick={onDelete}>
        <Trash2 className="h-3.5 w-3.5 mr-1.5 text-destructive" />
        <span className="text-destructive">{t("common.delete")}</span>
      </Button>
    </div>
  );
}

function AssigneesSection({ assignees, t }: {
  assignees?: Ticket["assignees"];
  t: (key: string) => string;
}) {
  return (
    <div className="px-4 py-3">
      <span className="text-xs font-medium text-muted-foreground block mb-2.5">{t("tickets.detail.assignees")}</span>
      {assignees && assignees.length > 0 ? (
        <div className="flex flex-col gap-2">
          {assignees.map((a) => (
            <div key={a.user_id} className="flex items-center gap-2 group">
              {a.user?.avatar_url ? (
                /* eslint-disable-next-line @next/next/no-img-element */
                <img src={a.user.avatar_url} alt="" className="w-6 h-6 rounded-full ring-1 ring-border/50" />
              ) : (
                <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-semibold text-primary ring-1 ring-primary/20">
                  {(a.user?.name || a.user?.username || "?")[0].toUpperCase()}
                </div>
              )}
              <span className="text-sm text-foreground/90">{a.user?.name || a.user?.username}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground/50 italic">{t("tickets.detail.noAssignees")}</p>
      )}
    </div>
  );
}

export default TicketDetailSidebar;
