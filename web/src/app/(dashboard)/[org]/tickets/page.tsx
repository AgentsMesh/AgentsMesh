"use client";

import { useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";
import { useAuthStore } from "@/stores/auth";
import { KanbanBoard, TicketCreateDialog } from "@/components/tickets";
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n/client";

// Status colors for list view
const statusColors: Record<string, string> = {
  backlog: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
  todo: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  in_progress: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  in_review: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
  done: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  cancelled: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

const priorityColors: Record<string, string> = {
  none: "text-gray-400 dark:text-gray-500",
  low: "text-blue-500 dark:text-blue-400",
  medium: "text-yellow-500 dark:text-yellow-400",
  high: "text-orange-500 dark:text-orange-400",
  urgent: "text-red-500 dark:text-red-400",
};

export default function TicketsPage() {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const {
    tickets,
    loading,
    viewMode,
    fetchTickets,
    updateTicketStatus,
  } = useTicketStore();

  // Load tickets on mount
  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const handleStatusChange = useCallback(async (identifier: string, newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(identifier, newStatus);
    } catch (error) {
      console.error("Failed to update ticket status:", error);
    }
  }, [updateTicketStatus]);

  const handleTicketClick = useCallback((ticket: Ticket) => {
    router.push(`/${currentOrg?.slug}/tickets/${ticket.identifier}`);
  }, [router, currentOrg]);

  if (loading && tickets.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Content - filtered by sidebar */}
      {viewMode === "list" ? (
        <div className="flex-1 overflow-auto p-4">
          <ListView tickets={tickets} t={t} orgSlug={currentOrg?.slug} />
        </div>
      ) : (
        <div className="flex-1 min-h-0 p-4">
          <KanbanBoard
            tickets={tickets}
            onStatusChange={handleStatusChange}
            onTicketClick={handleTicketClick}
          />
        </div>
      )}
    </div>
  );
}

function ListView({ tickets, t, orgSlug }: { tickets: Ticket[]; t: (key: string) => string; orgSlug?: string }) {
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <table className="w-full">
        <thead className="bg-muted">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.id")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.titleColumn")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.status")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.priority")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.type")}</th>
            <th className="px-4 py-3 text-left text-sm font-medium">{t("tickets.listView.created")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {tickets.map((ticket) => (
            <tr key={ticket.id} className="hover:bg-muted/50 cursor-pointer">
              <td className="px-4 py-3">
                <code className="text-sm text-primary">{ticket.identifier}</code>
              </td>
              <td className="px-4 py-3">
                <Link
                  href={`/${orgSlug}/tickets/${ticket.identifier}`}
                  className="text-foreground hover:text-primary"
                >
                  {ticket.title}
                </Link>
              </td>
              <td className="px-4 py-3">
                <span
                  className={`px-2 py-1 text-xs rounded-full ${
                    statusColors[ticket.status] || "bg-gray-100 dark:bg-gray-800"
                  }`}
                >
                  {t(`tickets.status.${ticket.status}`)}
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={priorityColors[ticket.priority] || ""}>
                  {t(`tickets.priority.${ticket.priority}`)}
                </span>
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {t(`tickets.type.${ticket.type}`)}
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {ticket.created_at ? new Date(ticket.created_at).toLocaleDateString() : "-"}
              </td>
            </tr>
          ))}
          {tickets.length === 0 && (
            <tr>
              <td
                colSpan={6}
                className="px-4 py-8 text-center text-muted-foreground"
              >
                {t("tickets.listView.noTickets")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

