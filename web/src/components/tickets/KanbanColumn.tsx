"use client";

import { useMemo } from "react";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { TicketCard } from "./TicketCard";
import { Ticket, TicketStatus } from "@/stores/ticket";
import { cn } from "@/lib/utils";
import { GripVertical } from "lucide-react";

type Status = TicketStatus;

export const statusConfig: { status: Status; labelKey: string; topColor: string; dotColor: string }[] = [
  { status: "backlog", labelKey: "tickets.status.backlog", topColor: "bg-gray-400 dark:bg-gray-500", dotColor: "bg-gray-400" },
  { status: "todo", labelKey: "tickets.status.todo", topColor: "bg-blue-400 dark:bg-blue-500", dotColor: "bg-blue-400" },
  { status: "in_progress", labelKey: "tickets.status.in_progress", topColor: "bg-yellow-400 dark:bg-yellow-500", dotColor: "bg-yellow-400" },
  { status: "in_review", labelKey: "tickets.status.in_review", topColor: "bg-purple-400 dark:bg-purple-500", dotColor: "bg-purple-400" },
  { status: "done", labelKey: "tickets.status.done", topColor: "bg-green-400 dark:bg-green-500", dotColor: "bg-green-400" },
];

interface SortableTicketProps {
  ticket: Ticket;
  onTicketClick?: (ticket: Ticket) => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}

export function SortableTicket({ ticket, onTicketClick, onMouseEnter, onMouseLeave }: SortableTicketProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: ticket.slug });
  const style = { transform: CSS.Transform.toString(transform), transition };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners}
      className={cn("transition-all duration-200 cursor-grab active:cursor-grabbing",
        isDragging ? "opacity-40 scale-[0.97] z-50" : "hover:scale-[1.01] hover:shadow-sm")}
      onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
      <TicketCard ticket={ticket} onClick={() => onTicketClick?.(ticket)} showRepository={false} showStatus={false} />
    </div>
  );
}

interface DroppableColumnProps {
  status: Status;
  labelKey: string;
  topColor: string;
  dotColor: string;
  tickets: Ticket[];
  isOver: boolean;
  onTicketClick?: (ticket: Ticket) => void;
  prefetchOnHover: (slug: string) => void;
  cancelPrefetch: () => void;
  t: (key: string) => string;
}

export function DroppableColumn({
  status, labelKey, topColor, dotColor, tickets, isOver,
  onTicketClick, prefetchOnHover, cancelPrefetch, t,
}: DroppableColumnProps) {
  const ticketIds = useMemo(() => tickets.map((t) => t.slug), [tickets]);
  const { setNodeRef, isOver: isDroppableOver } = useDroppable({ id: status });
  const highlighted = isOver || isDroppableOver;

  return (
    <div ref={setNodeRef}
      className={cn("flex-shrink-0 w-72 flex flex-col rounded-lg bg-muted/30 transition-all duration-200 overflow-hidden",
        highlighted && "ring-2 ring-primary/50 bg-primary/5")}>
      <div className={cn("h-1 w-full", topColor)} />
      <div className="flex items-center px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className={cn("w-2 h-2 rounded-full", dotColor)} />
          <h3 className="font-medium text-sm">{t(labelKey)}</h3>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto px-2 pb-2 space-y-1.5 min-h-[100px]">
        <SortableContext items={ticketIds} strategy={verticalListSortingStrategy}>
          {tickets.map((ticket) => (
            <SortableTicket key={ticket.slug} ticket={ticket} onTicketClick={onTicketClick}
              onMouseEnter={() => prefetchOnHover(ticket.slug)} onMouseLeave={cancelPrefetch} />
          ))}
        </SortableContext>
        {tickets.length === 0 && (
          <div className={cn("flex flex-col items-center justify-center py-10 text-muted-foreground/50 transition-colors rounded-lg border-2 border-dashed border-transparent",
            highlighted && "border-primary/30 text-primary/50")}>
            <GripVertical className="h-5 w-5 mb-2" />
            <span className="text-xs font-medium">
              {highlighted ? (t("tickets.kanban.dropHere") || "Drop here") : t("tickets.kanban.noTickets")}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
