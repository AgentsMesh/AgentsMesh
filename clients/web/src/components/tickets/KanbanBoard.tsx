"use client";

import { useState, useMemo, useCallback } from "react";
import {
  DndContext, DragOverlay, rectIntersection, pointerWithin,
  KeyboardSensor, MouseSensor, TouchSensor, useSensor, useSensors,
  DragStartEvent, DragEndEvent, DragOverEvent, CollisionDetection,
} from "@dnd-kit/core";
import { TicketCard } from "./TicketCard";
import { Ticket, TicketStatus, ColumnPagination } from "@/stores/ticket";
import type { BoardColumn } from "@/lib/api/ticketTypes";
import { useTranslations } from "next-intl";
import { useTicketPrefetch } from "@/hooks/useTicketPrefetch";
import { useColumnInfiniteScroll } from "@/hooks/useColumnInfiniteScroll";
import { statusConfig, DroppableColumn } from "./KanbanColumn";
import { CollapsedColumn } from "./CollapsedColumn";

type Status = TicketStatus;

interface KanbanBoardProps {
  tickets: Ticket[];
  boardColumns?: BoardColumn[];
  columnPagination: Record<string, ColumnPagination>;
  doneCollapsed: boolean;
  onLoadMoreColumn: (status: string) => Promise<void>;
  onSetDoneCollapsed: (collapsed: boolean) => void;
  onStatusChange?: (slug: string, newStatus: Status) => void;
  onTicketClick?: (ticket: Ticket) => void;
  onCreatePodRequest?: (ticket: Ticket) => void;
  excludeStatuses?: Status[];
}

export function KanbanBoard({
  tickets, boardColumns, columnPagination, doneCollapsed,
  onLoadMoreColumn, onSetDoneCollapsed,
  onStatusChange, onTicketClick, onCreatePodRequest, excludeStatuses = [],
}: KanbanBoardProps) {
  const t = useTranslations();
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [overColumn, setOverColumn] = useState<Status | null>(null);
  const { prefetchOnHover, cancelPrefetch } = useTicketPrefetch();

  const columns = statusConfig.filter((s) => !excludeStatuses.includes(s.status));
  const columnIds = useMemo(() => new Set<string>(columns.map(c => c.status)), [columns]);

  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 5 } }),
    useSensor(KeyboardSensor)
  );

  const findTicketBySlug = (slug: string) => tickets.find((t) => t.slug === slug);
  const findContainerByTicketId = (id: string) => findTicketBySlug(id)?.status;
  const getTicketsByStatus = (status: Status) => tickets.filter((t) => t.status === status);

  const getColumnCount = (status: Status): number | undefined => {
    if (!boardColumns) return undefined;
    return boardColumns.find((c) => c.status === status)?.count;
  };

  const collisionDetection: CollisionDetection = (args) => {
    const pointerCollisions = pointerWithin(args);
    const columnCollision = pointerCollisions.find(c => columnIds.has(c.id as string));
    if (columnCollision) return [columnCollision];
    const rectCollisions = rectIntersection(args);
    const ticketCollision = rectCollisions.find(c => !columnIds.has(c.id as string));
    if (ticketCollision) {
      const ticket = findTicketBySlug(ticketCollision.id as string);
      if (ticket && columnIds.has(ticket.status)) return [{ id: ticket.status }];
    }
    return pointerCollisions.length > 0 ? pointerCollisions : rectCollisions;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const ticket = findTicketBySlug(event.active.id as string);
    if (ticket) setActiveTicket(ticket);
  };

  const handleDragOver = (event: DragOverEvent) => {
    if (!event.over) { setOverColumn(null); return; }
    const overId = event.over.id as string;
    if (columns.some(c => c.status === overId)) { setOverColumn(overId as Status); return; }
    const container = findContainerByTicketId(overId);
    if (container) setOverColumn(container);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveTicket(null);
    setOverColumn(null);
    if (!over) return;
    const activeId = active.id as string;
    const overId = over.id as string;
    const ticket = findTicketBySlug(activeId);
    if (!ticket) return;
    const targetStatus = columns.some(c => c.status === overId)
      ? (overId as Status) : findContainerByTicketId(overId);
    if (targetStatus && ticket.status !== targetStatus) {
      onStatusChange?.(activeId, targetStatus);
      if (targetStatus === "in_progress" && ["backlog", "todo"].includes(ticket.status)) {
        onCreatePodRequest?.(ticket);
      }
    }
  };

  return (
    <DndContext sensors={sensors} collisionDetection={collisionDetection}
      onDragStart={handleDragStart} onDragOver={handleDragOver} onDragEnd={handleDragEnd}>
      <div className="flex gap-3 overflow-x-auto pb-4 h-full">
        {columns.map(({ status, labelKey, topColor, dotColor }) => {
          const isDone = status === "done";
          if (isDone && doneCollapsed) {
            return (
              <CollapsedColumn key={status} status={status} labelKey={labelKey}
                topColor={topColor} dotColor={dotColor} totalCount={getColumnCount(status) ?? 0}
                isOver={overColumn === status} onExpand={() => onSetDoneCollapsed(false)} t={t} />
            );
          }
          const pag = columnPagination[status];
          return (
            <ExpandedColumn key={status} status={status} labelKey={labelKey}
              topColor={topColor} dotColor={dotColor}
              tickets={getTicketsByStatus(status)} totalCount={getColumnCount(status)}
              pagination={pag} loadMore={onLoadMoreColumn}
              isOver={overColumn === status} onTicketClick={onTicketClick}
              onCollapse={isDone ? () => onSetDoneCollapsed(true) : undefined}
              prefetchOnHover={prefetchOnHover} cancelPrefetch={cancelPrefetch} t={t} />
          );
        })}
      </div>
      <DragOverlay>
        {activeTicket ? (
          <div className="opacity-95 scale-[1.02] rotate-1 shadow-lg ring-2 ring-primary/30 rounded-lg">
            <TicketCard ticket={activeTicket} showRepository={false} showStatus={false} />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}

function ExpandedColumn({ status, pagination, loadMore, onCollapse, ...props }: {
  status: Status;
  pagination?: ColumnPagination;
  loadMore: (status: string) => Promise<void>;
  onCollapse?: () => void;
} & Omit<React.ComponentProps<typeof DroppableColumn>, "hasMore" | "loadingMore" | "sentinelRef" | "onCollapse">) {
  const handleLoadMore = useCallback(() => loadMore(status), [loadMore, status]);
  const sentinelRef = useColumnInfiniteScroll({
    hasMore: pagination?.hasMore ?? false,
    loading: pagination?.loading ?? false,
    onLoadMore: handleLoadMore,
  });

  return (
    <DroppableColumn {...props} status={status}
      hasMore={pagination?.hasMore} loadingMore={pagination?.loading}
      sentinelRef={sentinelRef} onCollapse={onCollapse} />
  );
}

export default KanbanBoard;
