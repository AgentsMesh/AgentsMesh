"use client";

import { useEffect, useCallback } from "react";
import { Ticket } from "@/stores/ticket";

interface TicketKeyboardHandlerProps {
  tickets: Ticket[];
  selectedSlug: string | null;
  onSelectTicket: (slug: string | null) => void;
  onOpenDetail: (ticket: Ticket) => void;
  onCloseDetail: () => void;
  onCreateNew?: () => void;
  enabled?: boolean;
}

export function TicketKeyboardHandler({
  tickets,
  selectedSlug,
  onSelectTicket,
  onOpenDetail,
  onCloseDetail,
  onCreateNew,
  enabled = true,
}: TicketKeyboardHandlerProps) {
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!enabled) return;

    const target = e.target as HTMLElement;
    if (
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.isContentEditable
    ) {
      if (e.key !== "Escape") {
        return;
      }
    }

    const currentIndex = selectedSlug
      ? tickets.findIndex((t) => t.slug === selectedSlug)
      : -1;

    switch (e.key) {
      case "j":
      case "ArrowDown": {
        e.preventDefault();
        if (tickets.length === 0) return;

        if (currentIndex === -1) {
          onSelectTicket(tickets[0].slug);
        } else if (currentIndex < tickets.length - 1) {
          onSelectTicket(tickets[currentIndex + 1].slug);
        }
        break;
      }

      case "k":
      case "ArrowUp": {
        e.preventDefault();
        if (tickets.length === 0) return;

        if (currentIndex === -1) {
          onSelectTicket(tickets[tickets.length - 1].slug);
        } else if (currentIndex > 0) {
          onSelectTicket(tickets[currentIndex - 1].slug);
        }
        break;
      }

      case "Enter": {
        if (selectedSlug && currentIndex !== -1) {
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
        if (!e.metaKey && !e.ctrlKey && !e.altKey && onCreateNew) {
          e.preventDefault();
          onCreateNew();
        }
        break;
      }

      case "Home": {
        e.preventDefault();
        if (tickets.length > 0) {
          onSelectTicket(tickets[0].slug);
        }
        break;
      }

      case "End": {
        e.preventDefault();
        if (tickets.length > 0) {
          onSelectTicket(tickets[tickets.length - 1].slug);
        }
        break;
      }
    }
  }, [enabled, tickets, selectedSlug, onSelectTicket, onOpenDetail, onCloseDetail, onCreateNew]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return null;
}

export default TicketKeyboardHandler;
