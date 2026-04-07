"use client";

import { useMemo } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { StatusIcon, PriorityIcon, getStatusDisplayInfo, getPriorityDisplayInfo } from "@/components/tickets";
import type { Ticket, TicketStatus, TicketPriority } from "@/stores/ticket";
import { statusOptions, priorityOptions } from "./types";
import { FilterSection } from "./FilterSection";

interface TicketFilterSectionProps {
  statusExpanded: boolean;
  priorityExpanded: boolean;
  onStatusExpandedChange: (expanded: boolean) => void;
  onPriorityExpandedChange: (expanded: boolean) => void;
  selectedStatuses: TicketStatus[];
  selectedPriorities: TicketPriority[];
  onToggleStatus: (status: TicketStatus) => void;
  onTogglePriority: (priority: TicketPriority) => void;
  allTickets?: Ticket[];
  externalStatusCounts?: Record<string, number>;
  externalPriorityCounts?: Record<string, number>;
  t: (key: string) => string;
}

/**
 * TicketFilterSection - Renders status and priority filter collapsibles
 */
export function TicketFilterSection({
  statusExpanded,
  priorityExpanded,
  onStatusExpandedChange,
  onPriorityExpandedChange,
  selectedStatuses,
  selectedPriorities,
  onToggleStatus,
  onTogglePriority,
  allTickets,
  externalStatusCounts,
  externalPriorityCounts,
  t,
}: TicketFilterSectionProps) {
  const statusCounts = useMemo(() => {
    if (externalStatusCounts) return externalStatusCounts;
    if (!allTickets) return {};
    const counts: Record<string, number> = {};
    for (const ticket of allTickets) {
      counts[ticket.status] = (counts[ticket.status] || 0) + 1;
    }
    return counts;
  }, [externalStatusCounts, allTickets]);

  const priorityCounts = useMemo(() => {
    if (externalPriorityCounts) return externalPriorityCounts;
    if (!allTickets) return {};
    const counts: Record<string, number> = {};
    for (const ticket of allTickets) {
      counts[ticket.priority] = (counts[ticket.priority] || 0) + 1;
    }
    return counts;
  }, [externalPriorityCounts, allTickets]);

  return (
    <div className="border-t border-border">
      {/* Status Filter */}
      <FilterSection
        title={t("tickets.filters.status")}
        expanded={statusExpanded}
        onExpandedChange={onStatusExpandedChange}
        selectedCount={selectedStatuses.length}
      >
        {statusOptions.map((status) => {
          const info = getStatusDisplayInfo(status, t);
          const count = statusCounts[status];
          return (
            <label
              key={status}
              className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
            >
              <Checkbox
                checked={selectedStatuses.includes(status)}
                onCheckedChange={() => onToggleStatus(status)}
                className="h-3.5 w-3.5"
              />
              <StatusIcon status={status} size="xs" />
              <span className="flex-1">{info.label}</span>
              {count !== undefined && (
                <span className="text-muted-foreground/60 font-mono">{count}</span>
              )}
            </label>
          );
        })}
      </FilterSection>

      {/* Priority Filter */}
      <FilterSection
        title={t("tickets.filters.priority")}
        expanded={priorityExpanded}
        onExpandedChange={onPriorityExpandedChange}
        selectedCount={selectedPriorities.length}
        showBorder
      >
        {priorityOptions.map((priority) => {
          const info = getPriorityDisplayInfo(priority, t);
          const count = priorityCounts[priority];
          return (
            <label
              key={priority}
              className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
            >
              <Checkbox
                checked={selectedPriorities.includes(priority)}
                onCheckedChange={() => onTogglePriority(priority)}
                className="h-3.5 w-3.5"
              />
              <PriorityIcon priority={priority} size="xs" />
              <span className="flex-1">{info.label}</span>
              {count !== undefined && (
                <span className="text-muted-foreground/60 font-mono">{count}</span>
              )}
            </label>
          );
        })}
      </FilterSection>
    </div>
  );
}

export default TicketFilterSection;
