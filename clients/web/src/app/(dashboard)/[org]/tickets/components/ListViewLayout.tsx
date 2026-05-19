"use client";

import { Ticket } from "@/stores/ticket";
import { VirtualizedTicketList } from "@/components/tickets/VirtualizedTicketList";
import { TicketListView } from "./TicketListView";

const VIRTUALIZATION_THRESHOLD = 50;

interface ListViewLayoutProps {
  tickets: Ticket[];
  selectedSlug: string | null;
  onTicketClick: (ticket: Ticket) => void;
  t: (key: string) => string;
}

export function ListViewLayout({
  tickets,
  selectedSlug,
  onTicketClick,
  t,
}: ListViewLayoutProps) {
  const useVirtualization = tickets.length > VIRTUALIZATION_THRESHOLD;

  const ListComponent = useVirtualization ? (
    <VirtualizedTicketList
      tickets={tickets}
      selectedSlug={selectedSlug}
      onTicketClick={onTicketClick}
      t={t}
    />
  ) : (
    <TicketListView
      tickets={tickets}
      selectedSlug={selectedSlug}
      onTicketClick={onTicketClick}
      t={t}
    />
  );

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 overflow-hidden p-4">
        {ListComponent}
      </div>
    </div>
  );
}
