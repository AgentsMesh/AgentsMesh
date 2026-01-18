"use client";

import { useRef } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Ticket } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo } from "./TicketIcons";
import { useTicketPrefetch } from "@/hooks/useTicketPrefetch";
import { cn } from "@/lib/utils";

interface VirtualizedTicketListProps {
  tickets: Ticket[];
  selectedIdentifier: string | null;
  onTicketClick: (ticket: Ticket) => void;
  t: (key: string) => string;
  estimatedRowHeight?: number;
}

/**
 * Virtualized ticket list for handling large datasets efficiently.
 * Uses @tanstack/react-virtual for windowing.
 */
export function VirtualizedTicketList({
  tickets,
  selectedIdentifier,
  onTicketClick,
  t,
  estimatedRowHeight = 48,
}: VirtualizedTicketListProps) {
  const parentRef = useRef<HTMLDivElement>(null);
  const { prefetchOnHover, cancelPrefetch } = useTicketPrefetch();

  const virtualizer = useVirtualizer({
    count: tickets.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimatedRowHeight,
    overscan: 10, // Render 10 extra items above and below the visible area
  });

  const virtualItems = virtualizer.getVirtualItems();

  if (tickets.length === 0) {
    return (
      <div className="border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-8 text-center text-muted-foreground">
          {t("tickets.listView.noTickets")}
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      {/* Header */}
      <div className="bg-muted/50 border-b border-border">
        <div className="grid grid-cols-[1fr_2fr_120px_100px_100px_100px] gap-2 px-4 py-2.5">
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.id")}
          </div>
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.titleColumn")}
          </div>
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.status")}
          </div>
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.priority")}
          </div>
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.type")}
          </div>
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("tickets.listView.created")}
          </div>
        </div>
      </div>

      {/* Virtualized Body */}
      <div
        ref={parentRef}
        className="overflow-auto"
        style={{ height: "calc(100% - 41px)", maxHeight: "calc(100vh - 200px)" }}
      >
        <div
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {virtualItems.map((virtualRow) => {
            const ticket = tickets[virtualRow.index];
            const isSelected = ticket.identifier === selectedIdentifier;
            const statusInfo = getStatusDisplayInfo(ticket.status);

            return (
              <div
                key={virtualRow.key}
                data-index={virtualRow.index}
                ref={virtualizer.measureElement}
                className={cn(
                  "absolute top-0 left-0 w-full cursor-pointer transition-all duration-150 border-b border-border",
                  isSelected
                    ? "bg-primary/10 hover:bg-primary/15"
                    : "hover:bg-muted/50"
                )}
                style={{
                  transform: `translateY(${virtualRow.start}px)`,
                }}
                onClick={() => onTicketClick(ticket)}
                onMouseEnter={() => prefetchOnHover(ticket.identifier)}
                onMouseLeave={cancelPrefetch}
              >
                <div className="grid grid-cols-[1fr_2fr_120px_100px_100px_100px] gap-2 px-4 py-2.5 items-center">
                  {/* ID with Type Icon */}
                  <div className="flex items-center gap-2 min-w-0">
                    <TypeIcon type={ticket.type} size="sm" />
                    <code
                      className={cn(
                        "text-sm font-mono truncate",
                        isSelected ? "text-primary font-medium" : "text-primary"
                      )}
                    >
                      {ticket.identifier}
                    </code>
                  </div>

                  {/* Title */}
                  <div className="text-sm text-foreground truncate">
                    {ticket.title}
                  </div>

                  {/* Status */}
                  <div>
                    <span
                      className={cn(
                        "inline-flex items-center gap-1.5 px-2 py-0.5 text-xs rounded-full font-medium",
                        statusInfo.bgColor,
                        statusInfo.color
                      )}
                    >
                      <StatusIcon status={ticket.status} size="xs" />
                      <span className="truncate">{t(`tickets.status.${ticket.status}`)}</span>
                    </span>
                  </div>

                  {/* Priority */}
                  <div className="flex items-center gap-1.5">
                    <PriorityIcon priority={ticket.priority} size="sm" />
                    <span className="text-sm text-muted-foreground truncate">
                      {t(`tickets.priority.${ticket.priority}`)}
                    </span>
                  </div>

                  {/* Type */}
                  <div className="flex items-center gap-1.5">
                    <TypeIcon type={ticket.type} size="xs" />
                    <span className="text-sm text-muted-foreground truncate">
                      {t(`tickets.type.${ticket.type}`)}
                    </span>
                  </div>

                  {/* Created Date */}
                  <div className="text-sm text-muted-foreground">
                    {ticket.created_at
                      ? new Date(ticket.created_at).toLocaleDateString()
                      : "-"}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default VirtualizedTicketList;
