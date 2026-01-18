"use client";

import { useEffect, useCallback } from "react";
import { Ticket } from "@/stores/ticket";

interface TicketKeyboardHandlerProps {
  tickets: Ticket[];
  selectedIdentifier: string | null;
  onSelectTicket: (identifier: string | null) => void;
  onOpenDetail: (ticket: Ticket) => void;
  onCloseDetail: () => void;
  onCreateNew?: () => void;
  enabled?: boolean;
}

/**
 * Keyboard navigation handler for ticket list.
 *
 * Shortcuts:
 * - J / ArrowDown: Select next ticket
 * - K / ArrowUp: Select previous ticket
 * - Enter: Open detail panel
 * - Escape: Close detail panel
 * - C: Create new ticket
 * - /: Focus search (to be implemented)
 */
export function TicketKeyboardHandler({
  tickets,
  selectedIdentifier,
  onSelectTicket,
  onOpenDetail,
  onCloseDetail,
  onCreateNew,
  enabled = true,
}: TicketKeyboardHandlerProps) {
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!enabled) return;

    // Skip if user is typing in an input field
    const target = e.target as HTMLElement;
    if (
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.isContentEditable
    ) {
      // Allow Escape to work in inputs
      if (e.key !== "Escape") {
        return;
      }
    }

    // Find current index
    const currentIndex = selectedIdentifier
      ? tickets.findIndex((t) => t.identifier === selectedIdentifier)
      : -1;

    switch (e.key) {
      case "j":
      case "ArrowDown": {
        e.preventDefault();
        if (tickets.length === 0) return;

        if (currentIndex === -1) {
          // Select first ticket
          onSelectTicket(tickets[0].identifier);
        } else if (currentIndex < tickets.length - 1) {
          // Select next ticket
          onSelectTicket(tickets[currentIndex + 1].identifier);
        }
        break;
      }

      case "k":
      case "ArrowUp": {
        e.preventDefault();
        if (tickets.length === 0) return;

        if (currentIndex === -1) {
          // Select last ticket
          onSelectTicket(tickets[tickets.length - 1].identifier);
        } else if (currentIndex > 0) {
          // Select previous ticket
          onSelectTicket(tickets[currentIndex - 1].identifier);
        }
        break;
      }

      case "Enter": {
        if (selectedIdentifier && currentIndex !== -1) {
          e.preventDefault();
          onOpenDetail(tickets[currentIndex]);
        }
        break;
      }

      case "Escape": {
        e.preventDefault();
        onCloseDetail();
        break;
      }

      case "c": {
        // Only trigger if not in an input and not with modifiers
        if (!e.metaKey && !e.ctrlKey && !e.altKey && onCreateNew) {
          e.preventDefault();
          onCreateNew();
        }
        break;
      }

      case "Home": {
        e.preventDefault();
        if (tickets.length > 0) {
          onSelectTicket(tickets[0].identifier);
        }
        break;
      }

      case "End": {
        e.preventDefault();
        if (tickets.length > 0) {
          onSelectTicket(tickets[tickets.length - 1].identifier);
        }
        break;
      }
    }
  }, [enabled, tickets, selectedIdentifier, onSelectTicket, onOpenDetail, onCloseDetail, onCreateNew]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // This component doesn't render anything
  return null;
}

export default TicketKeyboardHandler;
