import { useEffect, useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { useTicketStore, useFilteredTickets, Ticket, TicketStatus } from "@/stores/ticket";
import { useAuthStore } from "@/stores/auth";
import { TicketKeyboardHandler } from "@/components/tickets";
import { CenteredSpinner } from "@/components/ui/spinner";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { buildTicketContext } from "@/components/tickets/buildTicketContext";
import { useTranslations } from "next-intl";
import { ListViewLayout, BoardViewLayout } from "./components";

export function TicketsPage() {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();

  const viewMode = useTicketStore(state => state.viewMode);
  const fetchTickets = useTicketStore(state => state.fetchTickets);
  const fetchBoard = useTicketStore(state => state.fetchBoard);
  const updateTicketStatus = useTicketStore(state => state.updateTicketStatus);

  const tickets = useFilteredTickets();

  const [keyboardSelectedSlug, setKeyboardSelectedSlug] = useState<string | null>(null);

  const [loading, setLoading] = useState(true);

  const [createPodTicket, setCreatePodTicket] = useState<Ticket | null>(null);

  useEffect(() => {
    const load = viewMode === "board" ? fetchBoard() : fetchTickets();
    load.finally(() => setLoading(false));
  }, [fetchTickets, fetchBoard, viewMode]);

  const handleStatusChange = useCallback(async (slug: string, newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(slug, newStatus);
    } catch (error) {
      console.error("Failed to update ticket status:", error);
    }
  }, [updateTicketStatus]);

  const handleTicketClick = useCallback((ticket: Ticket) => {
    router.push(`/${currentOrg?.slug}/tickets/${ticket.slug}`);
  }, [router, currentOrg]);

  const handleCreatePodRequest = useCallback((ticket: Ticket) => {
    setCreatePodTicket(ticket);
  }, []);

  const handleCreatePodClose = useCallback(() => {
    setCreatePodTicket(null);
  }, []);

  const handleSelectTicket = useCallback((slug: string | null) => {
    setKeyboardSelectedSlug(slug);
  }, []);

  if (loading && tickets.length === 0) {
    return <CenteredSpinner className="h-full" />;
  }

  const keyboardHandlerProps = {
    tickets,
    selectedSlug: keyboardSelectedSlug,
    onSelectTicket: handleSelectTicket,
    onOpenDetail: handleTicketClick,
    onCloseDetail: () => setKeyboardSelectedSlug(null),
    enabled: true,
  };

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
            ? buildTicketContext(createPodTicket, createPodTicket.slug)
            : undefined
        }
      />
    </>
  );
}
