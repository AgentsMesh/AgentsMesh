"use client";

import Link from "next/link";
import { useTranslations } from "@/lib/i18n/client";
import { useAuthStore } from "@/stores/auth";
import { Ticket } from "@/stores/ticket";

interface TicketCardProps {
  ticket: Ticket;
  onClick?: () => void;
  showRepository?: boolean;
}

const typeConfig: Record<string, { icon: string; color: string }> = {
  task: { icon: "✓", color: "text-blue-500 dark:text-blue-400" },
  bug: { icon: "🐛", color: "text-red-500 dark:text-red-400" },
  feature: { icon: "✨", color: "text-green-500 dark:text-green-400" },
  improvement: { icon: "📈", color: "text-cyan-500 dark:text-cyan-400" },
  epic: { icon: "⚡", color: "text-purple-500 dark:text-purple-400" },
  subtask: { icon: "◦", color: "text-gray-500 dark:text-gray-400" },
  story: { icon: "📖", color: "text-teal-500 dark:text-teal-400" },
};

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bg: "bg-gray-100 dark:bg-gray-800" },
  todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bg: "bg-blue-100 dark:bg-blue-900/30" },
  in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bg: "bg-purple-100 dark:bg-purple-900/30" },
  done: { label: "Done", color: "text-green-600 dark:text-green-400", bg: "bg-green-100 dark:bg-green-900/30" },
  cancelled: { label: "Cancelled", color: "text-red-600 dark:text-red-400", bg: "bg-red-100 dark:bg-red-900/30" },
};

const priorityConfig: Record<string, { icon: string; color: string }> = {
  none: { icon: "—", color: "text-gray-400 dark:text-gray-500" },
  low: { icon: "↓", color: "text-green-500 dark:text-green-400" },
  medium: { icon: "→", color: "text-yellow-500 dark:text-yellow-400" },
  high: { icon: "↑", color: "text-orange-500 dark:text-orange-400" },
  urgent: { icon: "⚡", color: "text-red-500 dark:text-red-400" },
};

export function TicketCard({ ticket, onClick, showRepository = true }: TicketCardProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const typeStyle = typeConfig[ticket.type] || typeConfig.task;
  const statusStyle = statusConfig[ticket.status] || statusConfig.backlog;
  const priorityStyle = priorityConfig[ticket.priority] || priorityConfig.none;

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
      className="border rounded-lg p-3 bg-card hover:shadow-md transition-shadow cursor-pointer"
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className={typeStyle.color} title={t(`tickets.type.${ticket.type}`)}>
            {typeStyle.icon}
          </span>
          <Link
            href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
            className="text-xs text-muted-foreground hover:text-primary font-mono"
            onClick={(e) => e.stopPropagation()}
          >
            {ticket.identifier}
          </Link>
        </div>
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusStyle.bg} ${statusStyle.color}`}
        >
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
          <span className={`text-sm ${priorityStyle.color}`} title={t(`tickets.priority.${ticket.priority}`)}>
            {priorityStyle.icon}
          </span>

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
