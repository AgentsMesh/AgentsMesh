"use client";

import { useEffect, useCallback, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Group, Panel, Separator } from "react-resizable-panels";
import { motion, AnimatePresence } from "framer-motion";
import { useTicketStore, Ticket, TicketStatus } from "@/stores/ticket";
import { useAuthStore } from "@/stores/auth";
import { KanbanBoard, TicketDetailPane, TicketKeyboardHandler, StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo } from "@/components/tickets";
import { VirtualizedTicketList } from "@/components/tickets/VirtualizedTicketList";
import { useTicketPrefetch } from "@/hooks/useTicketPrefetch";
import { Loader2, GripVertical, GripHorizontal } from "lucide-react";
import { useTranslations } from "@/lib/i18n/client";
import { cn } from "@/lib/utils";

// Breakpoint for responsive layout
const DESKTOP_BREAKPOINT = 1024;

// Use virtualization when ticket count exceeds this threshold
const VIRTUALIZATION_THRESHOLD = 50;

/**
 * VS Code style resize handle - hidden by default, highlights on hover
 */
function ResizeHandle({ direction }: { direction: "horizontal" | "vertical" }) {
  const isHorizontal = direction === "horizontal";

  return (
    <Separator
      className={cn(
        "group relative flex items-center justify-center bg-transparent transition-colors",
        isHorizontal
          ? "w-1 cursor-col-resize hover:bg-primary"
          : "h-1 cursor-row-resize hover:bg-primary"
      )}
    >
      {/* Expand hit area */}
      <div
        className={cn(
          "absolute z-10",
          isHorizontal ? "w-3 h-full -left-1" : "h-3 w-full -top-1"
        )}
      />
      {/* Grip indicator */}
      <div className={cn(
        "opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground"
      )}>
        {isHorizontal ? (
          <GripVertical className="h-4 w-4" />
        ) : (
          <GripHorizontal className="h-4 w-4" />
        )}
      </div>
    </Separator>
  );
}

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

  // Check if we have a selected ticket
  const hasSelectedTicket = !!selectedTicketIdentifier;

  if (loading && tickets.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Render content based on view mode and screen size
  if (viewMode === "list") {
    return (
      <>
        <TicketKeyboardHandler
          tickets={tickets}
          selectedIdentifier={selectedTicketIdentifier}
          onSelectTicket={(id) => {
            if (id) {
              router.push(`/${currentOrg?.slug}/tickets?ticket=${id}`, { scroll: false });
            } else {
              router.push(`/${currentOrg?.slug}/tickets`, { scroll: false });
            }
          }}
          onOpenDetail={handleTicketClick}
          onCloseDetail={handleClosePanel}
          enabled={isDesktop}
        />
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
      <TicketKeyboardHandler
        tickets={tickets}
        selectedIdentifier={selectedTicketIdentifier}
        onSelectTicket={(id) => {
          if (id) {
            router.push(`/${currentOrg?.slug}/tickets?ticket=${id}`, { scroll: false });
          } else {
            router.push(`/${currentOrg?.slug}/tickets`, { scroll: false });
          }
        }}
        onOpenDetail={handleTicketClick}
        onCloseDetail={handleClosePanel}
        enabled={isDesktop}
      />
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

/**
 * List view with right-side resizable panel
 */
interface ListViewLayoutProps {
  tickets: Ticket[];
  selectedTicketIdentifier: string | null;
  hasSelectedTicket: boolean;
  onTicketClick: (ticket: Ticket) => void;
  onClosePanel: () => void;
  t: (key: string) => string;
}

function ListViewLayout({
  tickets,
  selectedTicketIdentifier,
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
      selectedIdentifier={hasSelectedTicket ? selectedTicketIdentifier : null}
      onTicketClick={onTicketClick}
      t={t}
    />
  ) : (
    <ListView
      tickets={tickets}
      selectedIdentifier={hasSelectedTicket ? selectedTicketIdentifier : null}
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
          {selectedTicketIdentifier && (
            <motion.div
              key={selectedTicketIdentifier}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.15, ease: "easeOut" }}
              className="h-full border-l"
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

/**
 * Board view with bottom slide-up panel
 */
interface BoardViewLayoutProps {
  tickets: Ticket[];
  selectedTicketIdentifier: string | null;
  hasSelectedTicket: boolean;
  onStatusChange: (identifier: string, newStatus: TicketStatus) => Promise<void>;
  onTicketClick: (ticket: Ticket) => void;
  onClosePanel: () => void;
}

function BoardViewLayout({
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

/**
 * List view table component
 */
interface ListViewProps {
  tickets: Ticket[];
  selectedIdentifier: string | null;
  onTicketClick: (ticket: Ticket) => void;
  t: (key: string) => string;
}

function ListView({ tickets, selectedIdentifier, onTicketClick, t }: ListViewProps) {
  const { prefetchOnHover, cancelPrefetch } = useTicketPrefetch();

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <table className="w-full">
        <thead className="bg-muted/50">
          <tr>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.id")}</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.titleColumn")}</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.status")}</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.priority")}</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.type")}</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{t("tickets.listView.created")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {tickets.map((ticket) => {
            const isSelected = ticket.identifier === selectedIdentifier;
            const statusInfo = getStatusDisplayInfo(ticket.status);
            return (
              <tr
                key={ticket.id}
                className={cn(
                  "cursor-pointer transition-all duration-150",
                  isSelected
                    ? "bg-primary/10 hover:bg-primary/15"
                    : "hover:bg-muted/50"
                )}
                onClick={() => onTicketClick(ticket)}
                onMouseEnter={() => prefetchOnHover(ticket.identifier)}
                onMouseLeave={cancelPrefetch}
              >
                <td className="px-4 py-2.5">
                  <div className="flex items-center gap-2">
                    <TypeIcon type={ticket.type} size="sm" />
                    <code className={cn(
                      "text-sm font-mono",
                      isSelected ? "text-primary font-medium" : "text-primary"
                    )}>
                      {ticket.identifier}
                    </code>
                  </div>
                </td>
                <td className="px-4 py-2.5">
                  <span className="text-sm text-foreground line-clamp-1">
                    {ticket.title}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <span
                    className={cn(
                      "inline-flex items-center gap-1.5 px-2 py-0.5 text-xs rounded-full font-medium",
                      statusInfo.bgColor,
                      statusInfo.color
                    )}
                  >
                    <StatusIcon status={ticket.status} size="xs" />
                    {t(`tickets.status.${ticket.status}`)}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <div className="flex items-center gap-1.5">
                    <PriorityIcon priority={ticket.priority} size="sm" />
                    <span className="text-sm text-muted-foreground">
                      {t(`tickets.priority.${ticket.priority}`)}
                    </span>
                  </div>
                </td>
                <td className="px-4 py-2.5">
                  <div className="flex items-center gap-1.5">
                    <TypeIcon type={ticket.type} size="xs" />
                    <span className="text-sm text-muted-foreground">
                      {t(`tickets.type.${ticket.type}`)}
                    </span>
                  </div>
                </td>
                <td className="px-4 py-2.5 text-sm text-muted-foreground">
                  {ticket.created_at ? new Date(ticket.created_at).toLocaleDateString() : "-"}
                </td>
              </tr>
            );
          })}
          {tickets.length === 0 && (
            <tr>
              <td
                colSpan={6}
                className="px-4 py-8 text-center text-muted-foreground"
              >
                {t("tickets.listView.noTickets")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
