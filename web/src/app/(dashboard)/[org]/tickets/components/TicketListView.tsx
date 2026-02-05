"use client";

import { useTicketPrefetch } from "@/hooks/useTicketPrefetch";
import { Ticket, TicketStatus } from "@/stores/ticket";
import { StatusIcon, PriorityIcon, TypeIcon, getStatusDisplayInfo } from "@/components/tickets";
import { cn } from "@/lib/utils";

interface TicketListViewProps {
  tickets: Ticket[];
  selectedIdentifier: string | null;
  onTicketClick: (ticket: Ticket) => void;
  t: (key: string) => string;
}

/**
 * List view table component for tickets
 */
export function TicketListView({ tickets, selectedIdentifier, onTicketClick, t }: TicketListViewProps) {
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
