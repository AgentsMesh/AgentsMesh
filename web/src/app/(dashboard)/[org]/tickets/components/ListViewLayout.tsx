"use client";

import { Group, Panel } from "react-resizable-panels";
import { motion, AnimatePresence } from "framer-motion";
import { Ticket } from "@/stores/ticket";
import { TicketDetailPane } from "@/components/tickets";
import { VirtualizedTicketList } from "@/components/tickets/VirtualizedTicketList";
import { ResizeHandle } from "./ResizeHandle";
import { TicketListView } from "./TicketListView";

// Use virtualization when ticket count exceeds this threshold
const VIRTUALIZATION_THRESHOLD = 50;

interface ListViewLayoutProps {
  tickets: Ticket[];
  selectedTicketSlug: string | null;
  hasSelectedTicket: boolean;
  onTicketClick: (ticket: Ticket) => void;
  onClosePanel: () => void;
  t: (key: string) => string;
}

/**
 * List view with right-side resizable panel
 */
export function ListViewLayout({
  tickets,
  selectedTicketSlug,
  hasSelectedTicket,
  onTicketClick,
  onClosePanel,
  t,
}: ListViewLayoutProps) {
  // Use virtualization for large datasets
  const useVirtualization = tickets.length > VIRTUALIZATION_THRESHOLD;

  const ListComponent = useVirtualization ? (
    <VirtualizedTicketList
      tickets={tickets}
      selectedSlug={hasSelectedTicket ? selectedTicketSlug : null}
      onTicketClick={onTicketClick}
      t={t}
    />
  ) : (
    <TicketListView
      tickets={tickets}
      selectedSlug={hasSelectedTicket ? selectedTicketSlug : null}
      onTicketClick={onTicketClick}
      t={t}
    />
  );

  if (!hasSelectedTicket) {
    // No selected ticket - full width list
    return (
      <div className="h-full flex flex-col">
        <div className="flex-1 overflow-hidden p-4">
          {ListComponent}
        </div>
      </div>
    );
  }

  // With selected ticket - resizable panels
  return (
    <Group orientation="horizontal" className="h-full">
      <Panel defaultSize={60} minSize={30}>
        <div className="h-full overflow-hidden p-4">
          {ListComponent}
        </div>
      </Panel>
      <ResizeHandle direction="horizontal" />
      <Panel defaultSize={40} minSize={25}>
        <AnimatePresence mode="wait">
          {selectedTicketSlug && (
            <motion.div
              key={selectedTicketSlug}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.15, ease: "easeOut" }}
              className="h-full border-l"
            >
              <TicketDetailPane
                slug={selectedTicketSlug}
                onClose={onClosePanel}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </Panel>
    </Group>
  );
}
