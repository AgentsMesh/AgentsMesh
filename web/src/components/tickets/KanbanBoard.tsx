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

type Status = TicketStatus;

interface KanbanBoardProps {
  tickets: Ticket[];
  onStatusChange?: (identifier: string, newStatus: Status) => void;
  onTicketClick?: (ticket: Ticket) => void;
  onCreatePodRequest?: (ticket: Ticket) => void;
  excludeStatuses?: Status[];
}

const statusConfig: { status: Status; labelKey: string; color: string }[] = [
  { status: "backlog", labelKey: "tickets.status.backlog", color: "border-gray-300 dark:border-gray-600" },
  { status: "todo", labelKey: "tickets.status.todo", color: "border-blue-300 dark:border-blue-600" },
  { status: "in_progress", labelKey: "tickets.status.in_progress", color: "border-yellow-300 dark:border-yellow-600" },
  { status: "in_review", labelKey: "tickets.status.in_review", color: "border-purple-300 dark:border-purple-600" },
  { status: "done", labelKey: "tickets.status.done", color: "border-green-300 dark:border-green-600" },
];

/**
 * Sortable ticket item wrapper
 */
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
  } = useSortable({ id: ticket.identifier });

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
          ? "opacity-50 scale-95 rotate-1 z-50"
          : "hover:scale-[1.02] hover:shadow-md"
      )}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    >
      <TicketCard
        ticket={ticket}
        onClick={() => onTicketClick?.(ticket)}
        showRepository={false}
      />
    </div>
  );
}

/**
 * Droppable column wrapper
 */
interface DroppableColumnProps {
  status: Status;
  labelKey: string;
  color: string;
  tickets: Ticket[];
  isOver: boolean;
  onTicketClick?: (ticket: Ticket) => void;
  prefetchOnHover: (identifier: string) => void;
  cancelPrefetch: () => void;
  t: (key: string) => string;
}

function DroppableColumn({
  status,
  labelKey,
  color,
  tickets,
  isOver,
  onTicketClick,
  prefetchOnHover,
  cancelPrefetch,
  t,
}: DroppableColumnProps) {
  const ticketIds = useMemo(() => tickets.map((t) => t.identifier), [tickets]);

  // Register column as droppable area
  const { setNodeRef, isOver: isDroppableOver } = useDroppable({
    id: status,
  });

  const highlighted = isOver || isDroppableOver;

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex-shrink-0 w-72 flex flex-col rounded-lg bg-muted/30 transition-all duration-200",
        highlighted && "ring-2 ring-primary bg-primary/5 scale-[1.02]"
      )}
    >
      {/* Column Header */}
      <div className={`flex items-center justify-between p-3 border-b-2 ${color}`}>
        <h3 className="font-medium text-sm">{t(labelKey)}</h3>
        <span className="text-xs text-muted-foreground bg-background px-2 py-0.5 rounded-full">
          {tickets.length}
        </span>
      </div>

      {/* Column Content */}
      <div className="flex-1 overflow-y-auto p-2 space-y-2 min-h-[100px]">
        <SortableContext items={ticketIds} strategy={verticalListSortingStrategy}>
          {tickets.map((ticket) => (
            <SortableTicket
              key={ticket.identifier}
              ticket={ticket}
              onTicketClick={onTicketClick}
              onMouseEnter={() => prefetchOnHover(ticket.identifier)}
              onMouseLeave={cancelPrefetch}
            />
          ))}
        </SortableContext>

        {/* Empty State */}
        {tickets.length === 0 && (
          <div className="text-center py-8 text-muted-foreground text-sm">
            {t("tickets.kanban.noTickets")}
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
  excludeStatuses = ["cancelled"],
}: KanbanBoardProps) {
  const t = useTranslations();
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [overColumn, setOverColumn] = useState<Status | null>(null);
  const { prefetchOnHover, cancelPrefetch } = useTicketPrefetch();

  const columns = statusConfig.filter((s) => !excludeStatuses.includes(s.status));
  const columnIds = useMemo(() => new Set<string>(columns.map(c => c.status)), [columns]);

  // Configure sensors for drag and drop
  // MouseSensor for desktop (activate after 8px movement)
  // TouchSensor for mobile (activate after 250ms long-press, allowing normal scroll)
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

  /**
   * Custom collision detection that prioritizes columns over tickets.
   * This ensures dropping on a column's empty area works correctly.
   */
  const collisionDetection: CollisionDetection = (args) => {
    // First, check for collisions with columns using pointerWithin
    const pointerCollisions = pointerWithin(args);

    // Find if we're over a column
    const columnCollision = pointerCollisions.find(
      collision => columnIds.has(collision.id as string)
    );

    if (columnCollision) {
      return [columnCollision];
    }

    // Fall back to rect intersection for tickets within same column
    const rectCollisions = rectIntersection(args);

    // If over a ticket, find its parent column
    const ticketCollision = rectCollisions.find(
      collision => !columnIds.has(collision.id as string)
    );

    if (ticketCollision) {
      // Return both the ticket and find which column it belongs to
      const ticketId = ticketCollision.id as string;
      const ticket = findTicketByIdentifier(ticketId);
      if (ticket && columnIds.has(ticket.status)) {
        // Return the column as the collision target
        return [{ id: ticket.status }];
      }
    }

    return pointerCollisions.length > 0 ? pointerCollisions : rectCollisions;
  };

  const getTicketsByStatus = (status: Status) =>
    tickets.filter((t) => t.status === status);

  const findTicketByIdentifier = (identifier: string): Ticket | undefined =>
    tickets.find((t) => t.identifier === identifier);

  const findContainerByTicketId = (ticketId: string): Status | undefined => {
    const ticket = findTicketByIdentifier(ticketId);
    return ticket?.status;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;
    const ticket = findTicketByIdentifier(active.id as string);
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

    // Check if over a column (status) or a ticket
    const overId = over.id as string;

    // If it's a column ID (status)
    if (columns.some(c => c.status === overId)) {
      setOverColumn(overId as Status);
      return;
    }

    // If it's a ticket, find its container
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

    const activeTicket = findTicketByIdentifier(activeId);
    if (!activeTicket) return;

    // Determine target status
    let targetStatus: Status | undefined;

    // Check if dropped on a column
    if (columns.some(c => c.status === overId)) {
      targetStatus = overId as Status;
    } else {
      // Dropped on a ticket, use that ticket's status
      targetStatus = findContainerByTicketId(overId);
    }

    // Update status if changed
    if (targetStatus && activeTicket.status !== targetStatus) {
      onStatusChange?.(activeId, targetStatus);

      // Trigger create pod modal when moving from backlog/todo to in_progress
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
      <div className="flex gap-4 overflow-x-auto pb-4 h-full">
        {columns.map(({ status, labelKey, color }) => (
          <DroppableColumn
            key={status}
            status={status}
            labelKey={labelKey}
            color={color}
            tickets={getTicketsByStatus(status)}
            isOver={overColumn === status}
            onTicketClick={onTicketClick}
            prefetchOnHover={prefetchOnHover}
            cancelPrefetch={cancelPrefetch}
            t={t}
          />
        ))}
      </div>

      {/* Drag Overlay - Shows the dragged item */}
      <DragOverlay>
        {activeTicket ? (
          <div className="opacity-90 scale-105 rotate-2 shadow-xl">
            <TicketCard
              ticket={activeTicket}
              showRepository={false}
            />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}

export default KanbanBoard;
