"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { Ticket } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, getStatusDisplayInfo } from "./TicketIcons";

interface TicketCardProps {
  ticket: Ticket;
  onClick?: () => void;
  showRepository?: boolean;
  showStatus?: boolean;
}

export function TicketCard({ ticket, onClick, showRepository = true, showStatus = true }: TicketCardProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const statusInfo = getStatusDisplayInfo(ticket.status);

  const isDueSoon = () => {
    if (!ticket.due_date) return false;
    const due = new Date(ticket.due_date);
    const now = new Date();
    const diff = due.getTime() - now.getTime();
    const days = diff / (1000 * 60 * 60 * 24);
    return days >= 0 && days <= 3;
  };

  const isOverdue = () => {
    if (!ticket.due_date) return false;
    const due = new Date(ticket.due_date);
    const now = new Date();
    return due < now && ticket.status !== "done";
  };

  return (
    <div
      className="cursor-pointer rounded-md border border-border bg-card p-3.5 transition-colors hover:border-border-strong"
      onClick={onClick}
    >
      {/* Header: slug (mono) + status badge */}
      <div className="mb-2 flex items-center justify-between gap-2">
        <Link
          href={`/${currentOrg?.slug}/tickets/${ticket.slug}`}
          className="font-mono text-[11px] tracking-[0.02em] text-muted-foreground/80 hover:text-primary"
          onClick={(e) => e.stopPropagation()}
        >
          {ticket.slug}
        </Link>
        {showStatus && (
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${statusInfo.bgColor} ${statusInfo.color}`}
          >
            <StatusIcon status={ticket.status} size="xs" />
            {t(`tickets.status.${ticket.status}`)}
          </span>
        )}
      </div>

      {/* Title — clamped to 2 lines */}
      <h3 className="mb-2 line-clamp-2 text-[13px] font-semibold leading-[18px] text-foreground">
        {ticket.title}
      </h3>

      {/* Labels */}
      {ticket.labels && ticket.labels.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {ticket.labels.map((label) => (
            <span
              key={label.id}
              className="rounded-sm px-1.5 py-0.5 text-[10px] font-medium"
              style={{
                backgroundColor: `${label.color}1F`,
                color: label.color,
              }}
            >
              {label.name}
            </span>
          ))}
        </div>
      )}

      {/* Divider */}
      <div className="my-2 h-px w-full bg-border" />

      {/* Footer: priority + due | assignees */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <PriorityIcon priority={ticket.priority} size="sm" />
          {ticket.due_date && (
            <span
              className={`font-mono text-[11px] tabular-nums ${
                isOverdue()
                  ? "font-medium text-destructive"
                  : isDueSoon()
                  ? "text-warning"
                  : "text-muted-foreground/70"
              }`}
            >
              {new Date(ticket.due_date).toLocaleDateString()}
            </span>
          )}
        </div>

        <div className="flex -space-x-1.5">
          {ticket.assignees?.slice(0, 3).map((assignee) => (
            <div
              key={assignee.user_id}
              className="h-5 w-5 overflow-hidden rounded-full border border-card ring-1 ring-border"
              title={assignee.user?.name || assignee.user?.username}
            >
              {assignee.user?.avatar_url ? (
                /* eslint-disable-next-line @next/next/no-img-element */
                <img
                  src={assignee.user.avatar_url}
                  alt={assignee.user?.username}
                  className="h-full w-full object-cover"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center bg-accent text-[9px] font-semibold text-accent-foreground">
                  {(assignee.user?.name || assignee.user?.username || "?")[0].toUpperCase()}
                </div>
              )}
            </div>
          ))}
          {ticket.assignees && ticket.assignees.length > 3 && (
            <div className="flex h-5 w-5 items-center justify-center rounded-full border border-card bg-muted text-[9px] font-medium text-muted-foreground">
              +{ticket.assignees.length - 3}
            </div>
          )}
        </div>
      </div>

      {/* Repository */}
      {showRepository && ticket.repository && (
        <div className="mt-2 truncate font-mono text-[10px] text-muted-foreground/60">
          {ticket.repository.name}
        </div>
      )}
    </div>
  );
}

export default TicketCard;
