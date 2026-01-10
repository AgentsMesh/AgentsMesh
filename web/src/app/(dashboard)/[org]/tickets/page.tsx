"use client";

import { useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";
import { KanbanBoard, TicketCreateDialog } from "@/components/tickets";
import { Loader2 } from "lucide-react";

// Status colors for list view
const statusColors: Record<string, string> = {
  backlog: "bg-gray-100 text-gray-700",
  todo: "bg-blue-100 text-blue-700",
  in_progress: "bg-yellow-100 text-yellow-700",
  in_review: "bg-purple-100 text-purple-700",
  done: "bg-green-100 text-green-700",
  cancelled: "bg-red-100 text-red-700",
};

const priorityColors: Record<string, string> = {
  none: "text-gray-400",
  low: "text-blue-500",
  medium: "text-yellow-500",
  high: "text-orange-500",
  urgent: "text-red-500",
};

export default function TicketsPage() {
  const router = useRouter();
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
    router.push(`tickets/${ticket.identifier}`);
  }, [router]);

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
          <ListView tickets={tickets} />
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

function ListView({ tickets }: { tickets: Ticket[] }) {
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <table className="w-full">
        <thead className="bg-muted">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-medium">ID</th>
            <th className="px-4 py-3 text-left text-sm font-medium">Title</th>
            <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
            <th className="px-4 py-3 text-left text-sm font-medium">Priority</th>
            <th className="px-4 py-3 text-left text-sm font-medium">Type</th>
            <th className="px-4 py-3 text-left text-sm font-medium">Created</th>
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
                  href={`tickets/${ticket.identifier}`}
                  className="text-foreground hover:text-primary"
                >
                  {ticket.title}
                </Link>
              </td>
              <td className="px-4 py-3">
                <span
                  className={`px-2 py-1 text-xs rounded-full ${
                    statusColors[ticket.status] || "bg-gray-100"
                  }`}
                >
                  {ticket.status.replace("_", " ")}
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={priorityColors[ticket.priority] || ""}>
                  {ticket.priority}
                </span>
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {ticket.type}
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
                No tickets found. Use sidebar filters or create a new ticket.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

