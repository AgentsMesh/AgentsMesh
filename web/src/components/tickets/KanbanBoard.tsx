"use client";

import { useState, useMemo } from "react";
import {
  DndContext,
  DragOverlay,
  rectIntersection,
  pointerWithin,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  useDroppable,
  DragStartEvent,
  DragEndEvent,
  DragOverEvent,
  CollisionDetection,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { TicketCard } from "./TicketCard";
import { Ticket, TicketStatus } from "@/stores/ticket";
import { useTranslations } from "next-intl";
import { useTicketPrefetch } from "@/hooks/useTicketPrefetch";
import { cn } from "@/lib/utils";
import { GripVertical } from "lucide-react";

type Status = TicketStatus;

interface KanbanBoardProps {
  tickets: Ticket[];
  onStatusChange?: (slug: string, newStatus: Status) => void;
  onTicketClick?: (ticket: Ticket) => void;
  onCreatePodRequest?: (ticket: Ticket) => void;
  excludeStatuses?: Status[];
}

const statusConfig: { status: Status; labelKey: string; topColor: string; dotColor: string }[] = [
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

function SortableTicket({ ticket, onTicketClick, onMouseEnter, onMouseLeave }: SortableTicketProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: ticket.slug });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={cn(
        "transition-all duration-200 cursor-grab active:cursor-grabbing",
        isDragging
          ? "opacity-40 scale-[0.97] z-50"
          : "hover:scale-[1.01] hover:shadow-sm"
      )}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    >
      <TicketCard
        ticket={ticket}
        onClick={() => onTicketClick?.(ticket)}
        showRepository={false}
        showStatus={false}
      />
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

function DroppableColumn({
  status,
  labelKey,
  topColor,
  dotColor,
  tickets,
  isOver,
  onTicketClick,
  prefetchOnHover,
  cancelPrefetch,
  t,
}: DroppableColumnProps) {
  const ticketIds = useMemo(() => tickets.map((t) => t.slug), [tickets]);

  const { setNodeRef, isOver: isDroppableOver } = useDroppable({
    id: status,
  });

  const highlighted = isOver || isDroppableOver;

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex-shrink-0 w-72 flex flex-col rounded-lg bg-muted/30 transition-all duration-200 overflow-hidden",
        highlighted && "ring-2 ring-primary/50 bg-primary/5"
      )}
    >
      {/* Color bar */}
      <div className={cn("h-1 w-full", topColor)} />

      {/* Column Header */}
      <div className="flex items-center px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className={cn("w-2 h-2 rounded-full", dotColor)} />
          <h3 className="font-medium text-sm">{t(labelKey)}</h3>
        </div>
      </div>

      {/* Column Content */}
      <div className="flex-1 overflow-y-auto px-2 pb-2 space-y-1.5 min-h-[100px]">
        <SortableContext items={ticketIds} strategy={verticalListSortingStrategy}>
          {tickets.map((ticket) => (
            <SortableTicket
              key={ticket.slug}
              ticket={ticket}
              onTicketClick={onTicketClick}
              onMouseEnter={() => prefetchOnHover(ticket.slug)}
              onMouseLeave={cancelPrefetch}
            />
          ))}
        </SortableContext>

        {/* Empty State */}
        {tickets.length === 0 && (
          <div className={cn(
            "flex flex-col items-center justify-center py-10 text-muted-foreground/50 transition-colors rounded-lg border-2 border-dashed border-transparent",
            highlighted && "border-primary/30 text-primary/50"
          )}>
            <GripVertical className="h-5 w-5 mb-2" />
            <span className="text-xs font-medium">
              {highlighted
                ? (t("tickets.kanban.dropHere") || "Drop here")
                : (t("tickets.kanban.noTickets"))
              }
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

export function KanbanBoard({
  tickets,
  onStatusChange,
  onTicketClick,
  onCreatePodRequest,
  excludeStatuses = [],
}: KanbanBoardProps) {
  const t = useTranslations();
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [overColumn, setOverColumn] = useState<Status | null>(null);
  const { prefetchOnHover, cancelPrefetch } = useTicketPrefetch();

  const columns = statusConfig.filter((s) => !excludeStatuses.includes(s.status));
  const columnIds = useMemo(() => new Set<string>(columns.map(c => c.status)), [columns]);

  const sensors = useSensors(
    useSensor(MouseSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 250,
        tolerance: 5,
      },
    }),
    useSensor(KeyboardSensor)
  );

  const collisionDetection: CollisionDetection = (args) => {
    const pointerCollisions = pointerWithin(args);

    const columnCollision = pointerCollisions.find(
      collision => columnIds.has(collision.id as string)
    );

    if (columnCollision) {
      return [columnCollision];
    }

    const rectCollisions = rectIntersection(args);

    const ticketCollision = rectCollisions.find(
      collision => !columnIds.has(collision.id as string)
    );

    if (ticketCollision) {
      const ticketId = ticketCollision.id as string;
      const ticket = findTicketBySlug(ticketId);
      if (ticket && columnIds.has(ticket.status)) {
        return [{ id: ticket.status }];
      }
    }

    return pointerCollisions.length > 0 ? pointerCollisions : rectCollisions;
  };

  const getTicketsByStatus = (status: Status) =>
    tickets.filter((t) => t.status === status);

  const findTicketBySlug = (slug: string): Ticket | undefined =>
    tickets.find((t) => t.slug === slug);

  const findContainerByTicketId = (ticketId: string): Status | undefined => {
    const ticket = findTicketBySlug(ticketId);
    return ticket?.status;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;
    const ticket = findTicketBySlug(active.id as string);
    if (ticket) {
      setActiveTicket(ticket);
    }
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { over } = event;

    if (!over) {
      setOverColumn(null);
      return;
    }

    const overId = over.id as string;

    if (columns.some(c => c.status === overId)) {
      setOverColumn(overId as Status);
      return;
    }

    const overTicketContainer = findContainerByTicketId(overId);
    if (overTicketContainer) {
      setOverColumn(overTicketContainer);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    setActiveTicket(null);
    setOverColumn(null);

    if (!over) return;

    const activeId = active.id as string;
    const overId = over.id as string;

    const activeTicket = findTicketBySlug(activeId);
    if (!activeTicket) return;

    let targetStatus: Status | undefined;

    if (columns.some(c => c.status === overId)) {
      targetStatus = overId as Status;
    } else {
      targetStatus = findContainerByTicketId(overId);
    }

    if (targetStatus && activeTicket.status !== targetStatus) {
      onStatusChange?.(activeId, targetStatus);

      const podTriggerSources: Status[] = ["backlog", "todo"];
      if (
        targetStatus === "in_progress" &&
        podTriggerSources.includes(activeTicket.status)
      ) {
        onCreatePodRequest?.(activeTicket);
      }
    }
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={collisionDetection}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex gap-3 overflow-x-auto pb-4 h-full">
        {columns.map(({ status, labelKey, topColor, dotColor }) => (
          <DroppableColumn
            key={status}
            status={status}
            labelKey={labelKey}
            topColor={topColor}
            dotColor={dotColor}
            tickets={getTicketsByStatus(status)}
            isOver={overColumn === status}
            onTicketClick={onTicketClick}
            prefetchOnHover={prefetchOnHover}
            cancelPrefetch={cancelPrefetch}
            t={t}
          />
        ))}
      </div>

      {/* Drag Overlay */}
      <DragOverlay>
        {activeTicket ? (
          <div className="opacity-95 scale-[1.02] rotate-1 shadow-lg ring-2 ring-primary/30 rounded-lg">
            <TicketCard
              ticket={activeTicket}
              showRepository={false}
              showStatus={false}
            />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}

export default KanbanBoard;
