"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";

interface Label {
  id: number;
  name: string;
  color: string;
}

interface User {
  id: number;
  username: string;
  name?: string;
  avatarUrl?: string;
}

interface Ticket {
  id: number;
  number: number;
  identifier: string;
  type: "task" | "bug" | "feature" | "epic";
  title: string;
  description?: string;
  status: "backlog" | "todo" | "in_progress" | "in_review" | "done" | "cancelled";
  priority: "none" | "low" | "medium" | "high" | "urgent";
  dueDate?: string;
  createdAt: string;
  updatedAt: string;
  assignees?: User[];
  labels?: Label[];
  repository?: {
    id: number;
    name: string;
  };
}

interface TicketListProps {
  tickets: Ticket[];
  loading?: boolean;
  onTicketClick?: (ticket: Ticket) => void;
}

const typeConfig: Record<string, { icon: string; color: string; label: string }> = {
  task: { icon: "✓", color: "text-blue-500 dark:text-blue-400", label: "Task" },
  bug: { icon: "🐛", color: "text-red-500 dark:text-red-400", label: "Bug" },
  feature: { icon: "✨", color: "text-green-500 dark:text-green-400", label: "Feature" },
  epic: { icon: "⚡", color: "text-purple-500 dark:text-purple-400", label: "Epic" },
};

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bg: "bg-gray-100 dark:bg-gray-800" },
  todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bg: "bg-blue-100 dark:bg-blue-900/30" },
  in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bg: "bg-purple-100 dark:bg-purple-900/30" },
  done: { label: "Done", color: "text-green-600 dark:text-green-400", bg: "bg-green-100 dark:bg-green-900/30" },
  cancelled: { label: "Cancelled", color: "text-red-600 dark:text-red-400", bg: "bg-red-100 dark:bg-red-900/30" },
};

const priorityConfig: Record<string, { icon: string; color: string; label: string }> = {
  none: { icon: "—", color: "text-gray-400 dark:text-gray-500", label: "None" },
  low: { icon: "↓", color: "text-green-500 dark:text-green-400", label: "Low" },
  medium: { icon: "→", color: "text-yellow-500 dark:text-yellow-400", label: "Medium" },
  high: { icon: "↑", color: "text-orange-500 dark:text-orange-400", label: "High" },
  urgent: { icon: "⚡", color: "text-red-500 dark:text-red-400", label: "Urgent" },
};

export function TicketList({ tickets, loading, onTicketClick }: TicketListProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  if (loading) {
    return (
      <div className="space-y-2">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="h-16 bg-muted animate-pulse rounded-lg" />
        ))}
      </div>
    );
  }

  if (tickets.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <svg
          className="w-12 h-12 mx-auto mb-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        </svg>
        <p className="text-sm">{t("tickets.list.noTickets")}</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b text-left text-sm text-muted-foreground">
            <th className="pb-3 font-medium">{t("tickets.list.ticket")}</th>
            <th className="pb-3 font-medium">{t("tickets.filters.status")}</th>
            <th className="pb-3 font-medium">{t("tickets.filters.priority")}</th>
            <th className="pb-3 font-medium">{t("tickets.list.assignees")}</th>
            <th className="pb-3 font-medium">{t("tickets.list.dueDate")}</th>
            <th className="pb-3 font-medium">{t("tickets.list.updated")}</th>
          </tr>
        </thead>
        <tbody className="text-sm">
          {tickets.map((ticket) => {
            const typeStyle = typeConfig[ticket.type] || typeConfig.task;
            const statusStyle = statusConfig[ticket.status] || statusConfig.backlog;
            const priorityStyle = priorityConfig[ticket.priority] || priorityConfig.none;

            return (
              <tr
                key={ticket.id}
                className="border-b hover:bg-muted/50 cursor-pointer transition-colors"
                onClick={() => onTicketClick?.(ticket)}
              >
                {/* Ticket Info */}
                <td className="py-3">
                  <div className="flex items-center gap-3">
                    <span className={typeStyle.color} title={t(`tickets.type.${ticket.type}`)}>
                      {typeStyle.icon}
                    </span>
                    <div>
                      <div className="flex items-center gap-2">
                        <Link
                          href={`/${currentOrg?.slug}/tickets/${ticket.identifier}`}
                          className="text-xs text-muted-foreground hover:text-primary font-mono"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {ticket.identifier}
                        </Link>
                        {ticket.labels?.map((label) => (
                          <span
                            key={label.id}
                            className="px-1.5 py-0.5 rounded text-xs"
                            style={{
                              backgroundColor: `${label.color}20`,
                              color: label.color,
                            }}
                          >
                            {label.name}
                          </span>
                        ))}
                      </div>
                      <p className="font-medium mt-0.5 line-clamp-1">{ticket.title}</p>
                    </div>
                  </div>
                </td>

                {/* Status */}
                <td className="py-3">
                  <span
                    className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${statusStyle.bg} ${statusStyle.color}`}
                  >
                    {t(`tickets.status.${ticket.status}`)}
                  </span>
                </td>

                {/* Priority */}
                <td className="py-3">
                  <span
                    className={`inline-flex items-center gap-1 ${priorityStyle.color}`}
                    title={t(`tickets.priority.${ticket.priority}`)}
                  >
                    {priorityStyle.icon}
                    <span className="text-xs">{t(`tickets.priority.${ticket.priority}`)}</span>
                  </span>
                </td>

                {/* Assignees */}
                <td className="py-3">
                  {ticket.assignees && ticket.assignees.length > 0 ? (
                    <div className="flex -space-x-1">
                      {ticket.assignees.slice(0, 3).map((assignee) => (
                        <div
                          key={assignee.id}
                          className="w-6 h-6 rounded-full border-2 border-background overflow-hidden"
                          title={assignee.name || assignee.username}
                        >
                          {assignee.avatarUrl ? (
                            /* eslint-disable-next-line @next/next/no-img-element */
                            <img
                              src={assignee.avatarUrl}
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
                      {ticket.assignees.length > 3 && (
                        <div className="w-6 h-6 rounded-full border-2 border-background bg-muted flex items-center justify-center text-xs">
                          +{ticket.assignees.length - 3}
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </td>

                {/* Due Date */}
                <td className="py-3">
                  {ticket.dueDate ? (
                    <span
                      className={
                        new Date(ticket.dueDate) < new Date() &&
                        ticket.status !== "done" &&
                        ticket.status !== "cancelled"
                          ? "text-red-600 dark:text-red-400"
                          : "text-muted-foreground"
                      }
                    >
                      {formatDate(ticket.dueDate)}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </td>

                {/* Updated */}
                <td className="py-3 text-muted-foreground">
                  {formatDate(ticket.updatedAt)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default TicketList;
