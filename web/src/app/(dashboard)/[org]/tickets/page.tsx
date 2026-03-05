"use client";

import { useEffect, useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { useTicketStore, useFilteredTickets, Ticket, TicketStatus } from "@/stores/ticket";
import { useAuthStore } from "@/stores/auth";
import { TicketKeyboardHandler } from "@/components/tickets";
import { CenteredSpinner } from "@/components/ui/spinner";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { useTranslations } from "next-intl";
import { ListViewLayout, BoardViewLayout } from "./components";

export default function TicketsPage() {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();

  // Use individual selectors to prevent re-renders from unrelated store changes
  const viewMode = useTicketStore(state => state.viewMode);
  const fetchTickets = useTicketStore(state => state.fetchTickets);
  const updateTicketStatus = useTicketStore(state => state.updateTicketStatus);

  const tickets = useFilteredTickets();

  // Keyboard-selected ticket for J/K navigation highlighting
  const [keyboardSelectedSlug, setKeyboardSelectedSlug] = useState<string | null>(null);

  // Local loading state — store's shared `loading` is no longer set by async actions
  const [loading, setLoading] = useState(true);

  // State for auto-triggered create pod modal (when dragging ticket to in_progress)
  const [createPodTicket, setCreatePodTicket] = useState<Ticket | null>(null);

  // Load tickets on mount
  useEffect(() => {
    fetchTickets().finally(() => setLoading(false));
  }, [fetchTickets]);

  const handleStatusChange = useCallback(async (slug: string, newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(slug, newStatus);
    } catch (error) {
      console.error("Failed to update ticket status:", error);
    }
  }, [updateTicketStatus]);

  const handleTicketClick = useCallback((ticket: Ticket) => {
    // Navigate directly to ticket detail page (Linear-style)
    router.push(`/${currentOrg?.slug}/tickets/${ticket.slug}`);
  }, [router, currentOrg]);

  const handleCreatePodRequest = useCallback((ticket: Ticket) => {
    setCreatePodTicket(ticket);
  }, []);

  const handleCreatePodClose = useCallback(() => {
    setCreatePodTicket(null);
  }, []);

  // J/K only highlights, does not navigate
  const handleSelectTicket = useCallback((slug: string | null) => {
    setKeyboardSelectedSlug(slug);
  }, []);

  if (loading && tickets.length === 0) {
    return <CenteredSpinner className="h-full" />;
  }

  // Keyboard handler props
  const keyboardHandlerProps = {
    tickets,
    selectedSlug: keyboardSelectedSlug,
    onSelectTicket: handleSelectTicket,
    onOpenDetail: handleTicketClick,          // Enter → navigate
    onCloseDetail: () => setKeyboardSelectedSlug(null), // Escape → clear selection
    enabled: true,
  };

  // Render content based on view mode
  if (viewMode === "list") {
    return (
      <>
        <TicketKeyboardHandler {...keyboardHandlerProps} />
        <ListViewLayout
          tickets={tickets}
          selectedSlug={keyboardSelectedSlug}
          onTicketClick={handleTicketClick}
          t={t}
        />
      </>
    );
  }

  // Board view
  return (
    <>
      <TicketKeyboardHandler {...keyboardHandlerProps} />
      <BoardViewLayout
        tickets={tickets}
        onStatusChange={handleStatusChange}
        onTicketClick={handleTicketClick}
        onCreatePodRequest={handleCreatePodRequest}
      />
      <CreatePodModal
        open={!!createPodTicket}
        onClose={handleCreatePodClose}
        onCreated={handleCreatePodClose}
        ticketContext={
          createPodTicket
            ? {
                id: createPodTicket.id,
                slug: createPodTicket.slug,
                title: createPodTicket.title,
                repositoryId: createPodTicket.repository_id,
              }
            : undefined
        }
      />
    </>
  );
}
