"use client";

import Link from "next/link";
import { useTranslations } from "@/lib/i18n/client";
import { useAuthStore } from "@/stores/auth";
import { Ticket } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo } from "./TicketIcons";

interface TicketCardProps {
  ticket: Ticket;
  onClick?: () => void;
  showRepository?: boolean;
}

export function TicketCard({ ticket, onClick, showRepository = true }: TicketCardProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
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
    return due < now && ticket.status !== "done" && ticket.status !== "cancelled";
  };

  return (
    <div
      className="border rounded-lg p-3 bg-card hover:shadow-md hover:border-primary/30 transition-all duration-150 cursor-pointer"
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2 min-w-0">
          <TypeIcon type={ticket.type} size="sm" />
          <Link
            href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
            className="text-xs text-muted-foreground hover:text-primary font-mono"
            onClick={(e) => e.stopPropagation()}
          >
            {ticket.identifier}
          </Link>
        </div>
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${statusInfo.bgColor} ${statusInfo.color}`}
        >
          <StatusIcon status={ticket.status} size="xs" />
          {t(`tickets.status.${ticket.status}`)}
        </span>
      </div>

      {/* Title */}
      <h3 className="font-medium text-sm line-clamp-2 mb-2">{ticket.title}</h3>

      {/* Labels */}
      {ticket.labels && ticket.labels.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-2">
          {ticket.labels.map((label) => (
            <span
              key={label.id}
              className="px-2 py-0.5 rounded text-xs"
              style={{
                backgroundColor: `${label.color}20`,
                color: label.color,
              }}
            >
              {label.name}
            </span>
          ))}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-2">
        <div className="flex items-center gap-2">
          {/* Priority */}
          <PriorityIcon priority={ticket.priority} size="sm" />

          {/* Due Date */}
          {ticket.due_date && (
            <span
              className={`text-xs ${
                isOverdue()
                  ? "text-red-600 dark:text-red-400"
                  : isDueSoon()
                  ? "text-orange-600 dark:text-orange-400"
                  : "text-muted-foreground"
              }`}
            >
              {new Date(ticket.due_date).toLocaleDateString()}
            </span>
          )}
        </div>

        {/* Assignees */}
        <div className="flex -space-x-1">
          {ticket.assignees?.slice(0, 3).map((assignee) => (
            <div
              key={assignee.id}
              className="w-6 h-6 rounded-full border-2 border-background overflow-hidden"
              title={assignee.name || assignee.username}
            >
              {assignee.avatar_url ? (
                <img
                  src={assignee.avatar_url}
                  alt={assignee.username}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="w-full h-full bg-muted flex items-center justify-center text-xs">
                  {(assignee.name || assignee.username)[0].toUpperCase()}
                </div>
              )}
            </div>
          ))}
          {ticket.assignees && ticket.assignees.length > 3 && (
            <div className="w-6 h-6 rounded-full border-2 border-background bg-muted flex items-center justify-center text-xs">
              +{ticket.assignees.length - 3}
            </div>
          )}
        </div>
      </div>

      {/* Repository */}
      {showRepository && ticket.repository && (
        <div className="mt-2 text-xs text-muted-foreground">
          {ticket.repository.name}
        </div>
      )}
    </div>
  );
}

export default TicketCard;
