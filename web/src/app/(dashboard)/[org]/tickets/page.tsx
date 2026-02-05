"use client";

import { useEffect, useCallback, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";
import { useAuthStore } from "@/stores/auth";
import { TicketKeyboardHandler } from "@/components/tickets";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useTranslations } from "@/lib/i18n/client";
import { ListViewLayout, BoardViewLayout } from "./components";

// Breakpoint for responsive layout
const DESKTOP_BREAKPOINT = 1024;

export default function TicketsPage() {
  const t = useTranslations();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { currentOrg } = useAuthStore();
  const {
    tickets,
    loading,
    viewMode,
    selectedTicketIdentifier,
    fetchTickets,
    updateTicketStatus,
    setSelectedTicketIdentifier,
  } = useTicketStore();

  // Track screen size for responsive layout
  const [isDesktop, setIsDesktop] = useState(true);

  // Get selected ticket from URL or store
  const selectedTicketFromUrl = searchParams.get("ticket");

  // Sync URL with store
  useEffect(() => {
    if (selectedTicketFromUrl !== selectedTicketIdentifier) {
      setSelectedTicketIdentifier(selectedTicketFromUrl);
    }
  }, [selectedTicketFromUrl, selectedTicketIdentifier, setSelectedTicketIdentifier]);

  // Handle window resize
  useEffect(() => {
    const checkDesktop = () => {
      setIsDesktop(window.innerWidth >= DESKTOP_BREAKPOINT);
    };

    checkDesktop();
    window.addEventListener("resize", checkDesktop);
    return () => window.removeEventListener("resize", checkDesktop);
  }, []);

  // Load tickets on mount
  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const handleStatusChange = useCallback(async (identifier: string, newStatus: TicketStatus) => {
    try {
      await updateTicketStatus(identifier, newStatus);
    } catch (error) {
      console.error("Failed to update ticket status:", error);
    }
  }, [updateTicketStatus]);

  const handleTicketClick = useCallback((ticket: Ticket) => {
    if (!isDesktop) {
      // On mobile, navigate to full page
      router.push(`/${currentOrg?.slug}/tickets/${ticket.identifier}`);
    } else {
      // On desktop, update URL with query param to show panel
      const newUrl = `/${currentOrg?.slug}/tickets?ticket=${ticket.identifier}`;
      router.push(newUrl, { scroll: false });
    }
  }, [router, currentOrg, isDesktop]);

  const handleClosePanel = useCallback(() => {
    setSelectedTicketIdentifier(null);
    router.push(`/${currentOrg?.slug}/tickets`, { scroll: false });
  }, [router, currentOrg, setSelectedTicketIdentifier]);

  const handleSelectTicket = useCallback((id: string | null) => {
    if (id) {
      router.push(`/${currentOrg?.slug}/tickets?ticket=${id}`, { scroll: false });
    } else {
      router.push(`/${currentOrg?.slug}/tickets`, { scroll: false });
    }
  }, [router, currentOrg]);

  // Check if we have a selected ticket
  const hasSelectedTicket = !!selectedTicketIdentifier;

  if (loading && tickets.length === 0) {
    return <CenteredSpinner className="h-full" />;
  }

  // Common keyboard handler props
  const keyboardHandlerProps = {
    tickets,
    selectedIdentifier: selectedTicketIdentifier,
    onSelectTicket: handleSelectTicket,
    onOpenDetail: handleTicketClick,
    onCloseDetail: handleClosePanel,
    enabled: isDesktop,
  };

  // Render content based on view mode and screen size
  if (viewMode === "list") {
    return (
      <>
        <TicketKeyboardHandler {...keyboardHandlerProps} />
        <ListViewLayout
          tickets={tickets}
          selectedTicketIdentifier={selectedTicketIdentifier}
          hasSelectedTicket={hasSelectedTicket && isDesktop}
          onTicketClick={handleTicketClick}
          onClosePanel={handleClosePanel}
          t={t}
        />
      </>
    );
  }

  // Board view with bottom slide-up panel
  return (
    <>
      <TicketKeyboardHandler {...keyboardHandlerProps} />
      <BoardViewLayout
        tickets={tickets}
        selectedTicketIdentifier={selectedTicketIdentifier}
        hasSelectedTicket={hasSelectedTicket && isDesktop}
        onStatusChange={handleStatusChange}
        onTicketClick={handleTicketClick}
        onClosePanel={handleClosePanel}
      />
    </>
  );
}
