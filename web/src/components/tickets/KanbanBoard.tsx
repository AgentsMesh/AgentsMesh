"use client";

import { useState, DragEvent } from "react";
import { TicketCard } from "./TicketCard";
import { Ticket, TicketStatus } from "@/stores/ticket";
import { useTranslations } from "@/lib/i18n/client";

type Status = TicketStatus;

interface KanbanBoardProps {
  tickets: Ticket[];
  onStatusChange?: (identifier: string, newStatus: Status) => void;
  onTicketClick?: (ticket: Ticket) => void;
  excludeStatuses?: Status[];
}

const statusConfig: { status: Status; labelKey: string; color: string }[] = [
  { status: "backlog", labelKey: "tickets.status.backlog", color: "border-gray-300 dark:border-gray-600" },
  { status: "todo", labelKey: "tickets.status.todo", color: "border-blue-300 dark:border-blue-600" },
  { status: "in_progress", labelKey: "tickets.status.in_progress", color: "border-yellow-300 dark:border-yellow-600" },
  { status: "in_review", labelKey: "tickets.status.in_review", color: "border-purple-300 dark:border-purple-600" },
  { status: "done", labelKey: "tickets.status.done", color: "border-green-300 dark:border-green-600" },
];

export function KanbanBoard({
  tickets,
  onStatusChange,
  onTicketClick,
  excludeStatuses = ["cancelled"],
}: KanbanBoardProps) {
  const t = useTranslations();
  const [draggedTicket, setDraggedTicket] = useState<Ticket | null>(null);
  const [dragOverStatus, setDragOverStatus] = useState<Status | null>(null);

  const columns = statusConfig.filter((s) => !excludeStatuses.includes(s.status));

  const getTicketsByStatus = (status: Status) =>
    tickets.filter((t) => t.status === status);

  const handleDragStart = (e: DragEvent, ticket: Ticket) => {
    setDraggedTicket(ticket);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", ticket.identifier);
  };

  const handleDragEnd = () => {
    setDraggedTicket(null);
    setDragOverStatus(null);
  };

  const handleDragOver = (e: DragEvent, status: Status) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOverStatus(status);
  };

  const handleDragLeave = () => {
    setDragOverStatus(null);
  };

  const handleDrop = (e: DragEvent, newStatus: Status) => {
    e.preventDefault();
    setDragOverStatus(null);

    if (draggedTicket && draggedTicket.status !== newStatus) {
      onStatusChange?.(draggedTicket.identifier, newStatus);
    }
    setDraggedTicket(null);
  };

  return (
    <div className="flex gap-4 overflow-x-auto pb-4 h-full">
      {columns.map(({ status, labelKey, color }) => {
        const columnTickets = getTicketsByStatus(status);
        const isDropTarget = dragOverStatus === status;

        return (
          <div
            key={status}
            className={`flex-shrink-0 w-72 flex flex-col rounded-lg bg-muted/30 ${
              isDropTarget ? "ring-2 ring-primary" : ""
            }`}
            onDragOver={(e) => handleDragOver(e, status)}
            onDragLeave={handleDragLeave}
            onDrop={(e) => handleDrop(e, status)}
          >
            {/* Column Header */}
            <div
              className={`flex items-center justify-between p-3 border-b-2 ${color}`}
            >
              <h3 className="font-medium text-sm">{t(labelKey)}</h3>
              <span className="text-xs text-muted-foreground bg-background px-2 py-0.5 rounded-full">
                {columnTickets.length}
              </span>
            </div>

            {/* Column Content */}
            <div className="flex-1 overflow-y-auto p-2 space-y-2">
              {columnTickets.map((ticket) => (
                <div
                  key={ticket.id}
                  draggable
                  onDragStart={(e) => handleDragStart(e, ticket)}
                  onDragEnd={handleDragEnd}
                  className={`transition-opacity ${
                    draggedTicket?.id === ticket.id ? "opacity-50" : ""
                  }`}
                >
                  <TicketCard
                    ticket={ticket}
                    onClick={() => onTicketClick?.(ticket)}
                    showRepository={false}
                  />
                </div>
              ))}

              {/* Empty State */}
              {columnTickets.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  {t("tickets.kanban.noTickets")}
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default KanbanBoard;
