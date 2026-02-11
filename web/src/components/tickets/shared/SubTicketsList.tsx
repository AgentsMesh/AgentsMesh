"use client";

import { useTranslations } from "next-intl";
import { Ticket } from "@/stores/ticket";
import { StatusIcon, TypeIcon, getStatusDisplayInfo } from "../TicketIcons";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface SubTicketsListProps {
  subTickets: Ticket[];
  onTicketClick: (identifier: string) => void;
  /** Compact style for panel view */
  compact?: boolean;
  className?: string;
}

/**
 * Shared sub-tickets list component used by both TicketDetail and TicketDetailPane
 */
export function SubTicketsList({
  subTickets,
  onTicketClick,
  compact = false,
  className,
}: SubTicketsListProps) {
  const t = useTranslations();

  if (subTickets.length === 0) return null;

  if (compact) {
    return (
      <div className={cn("space-y-2", className)}>
        <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
          {t("tickets.detail.subTickets")} ({subTickets.length})
        </label>
        <div className="space-y-1">
          {subTickets.map((subTicket) => (
            <button
              key={subTicket.id}
              className="w-full px-2.5 py-1.5 flex items-center gap-2 hover:bg-muted/50 rounded-md transition-colors text-left group"
              onClick={() => onTicketClick(subTicket.identifier)}
            >
              <StatusIcon status={subTicket.status} size="sm" />
              <span className="font-mono text-xs text-muted-foreground">
                {subTicket.identifier}
              </span>
              <span className="flex-1 truncate text-sm">{subTicket.title}</span>
              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
            </button>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("mb-6", className)}>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <span className="text-muted-foreground">◦</span>
        {t("tickets.detail.subTickets")} ({subTickets.length})
      </h3>
      <div className="border border-border rounded-lg divide-y divide-border">
        {subTickets.map((subTicket) => {
          const subStatusInfo = getStatusDisplayInfo(subTicket.status);
          return (
            <div
              key={subTicket.id}
              className="px-4 py-3 hover:bg-muted/50 cursor-pointer"
              onClick={() => onTicketClick(subTicket.identifier)}
            >
              <div className="flex items-center gap-2">
                <TypeIcon type={subTicket.type} size="sm" />
                <span className="font-mono text-xs text-muted-foreground">
                  {subTicket.identifier}
                </span>
                <span className="flex-1 truncate">{subTicket.title}</span>
                <span className={cn(
                  "flex items-center gap-1 px-2 py-0.5 rounded text-xs",
                  subStatusInfo.bgColor,
                  subStatusInfo.color
                )}>
                  <StatusIcon status={subTicket.status} size="xs" />
                  {subStatusInfo.label}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default SubTicketsList;
