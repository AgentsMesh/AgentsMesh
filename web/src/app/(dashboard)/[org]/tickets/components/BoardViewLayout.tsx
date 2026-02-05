"use client";

import { Group, Panel } from "react-resizable-panels";
import { motion, AnimatePresence } from "framer-motion";
import { Ticket, TicketStatus } from "@/stores/ticket";
import { KanbanBoard, TicketDetailPane } from "@/components/tickets";
import { ResizeHandle } from "./ResizeHandle";

interface BoardViewLayoutProps {
  tickets: Ticket[];
  selectedTicketIdentifier: string | null;
  hasSelectedTicket: boolean;
  onStatusChange: (identifier: string, newStatus: TicketStatus) => Promise<void>;
  onTicketClick: (ticket: Ticket) => void;
  onClosePanel: () => void;
}

/**
 * Board view with bottom slide-up panel
 */
export function BoardViewLayout({
  tickets,
  selectedTicketIdentifier,
  hasSelectedTicket,
  onStatusChange,
  onTicketClick,
  onClosePanel,
}: BoardViewLayoutProps) {
  if (!hasSelectedTicket) {
    // No selected ticket - full height board
    return (
      <div className="h-full flex flex-col">
        <div className="flex-1 min-h-0 p-4">
          <KanbanBoard
            tickets={tickets}
            onStatusChange={onStatusChange}
            onTicketClick={onTicketClick}
          />
        </div>
      </div>
    );
  }

  // With selected ticket - vertical resizable panels
  return (
    <Group orientation="vertical" className="h-full">
      <Panel defaultSize={60} minSize={30}>
        <div className="h-full p-4">
          <KanbanBoard
            tickets={tickets}
            onStatusChange={onStatusChange}
            onTicketClick={onTicketClick}
          />
        </div>
      </Panel>
      <ResizeHandle direction="vertical" />
      <Panel defaultSize={40} minSize={20}>
        <AnimatePresence mode="wait">
          {selectedTicketIdentifier && (
            <motion.div
              key={selectedTicketIdentifier}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 20 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="h-full border-t"
            >
              <TicketDetailPane
                identifier={selectedTicketIdentifier}
                onClose={onClosePanel}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </Panel>
    </Group>
  );
}
