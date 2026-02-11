"use client";

import { useTranslations } from "next-intl";
import { TicketRelation } from "@/lib/api";
import { ChevronRight, Link as LinkIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface RelationsListProps {
  relations: TicketRelation[];
  onTicketClick: (identifier: string) => void;
  /** Compact style for panel view */
  compact?: boolean;
  className?: string;
}

/**
 * Shared relations list component used by both TicketDetail and TicketDetailPane
 */
export function RelationsList({
  relations,
  onTicketClick,
  compact = false,
  className,
}: RelationsListProps) {
  const t = useTranslations();

  if (relations.length === 0) return null;

  if (compact) {
    return (
      <div className={cn("space-y-2", className)}>
        <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider flex items-center gap-1">
          <LinkIcon className="h-3 w-3" />
          {t("tickets.detail.related")}
        </label>
        <div className="space-y-1">
          {relations.map((relation) => {
            const targetTicket = relation.target_ticket;
            if (!targetTicket) return null;
            return (
              <button
                key={relation.id}
                className="w-full px-2.5 py-1.5 flex items-center gap-2 hover:bg-muted/50 rounded-md transition-colors text-left group"
                onClick={() => onTicketClick(targetTicket.identifier)}
              >
                <span className="text-[10px] text-muted-foreground capitalize bg-muted/70 px-1.5 py-0.5 rounded">
                  {relation.relation_type}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {targetTicket.identifier}
                </span>
                <span className="flex-1 truncate text-sm">{targetTicket.title}</span>
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
              </button>
            );
          })}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("mb-6", className)}>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        {t("tickets.detail.related")} ({relations.length})
      </h3>
      <div className="border border-border rounded-lg divide-y divide-border">
        {relations.map((relation) => {
          const targetTicket = relation.target_ticket;
          if (!targetTicket) return null;
          return (
            <div
              key={relation.id}
              className="px-4 py-3 hover:bg-muted/50 cursor-pointer"
              onClick={() => onTicketClick(targetTicket.identifier)}
            >
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground capitalize">
                  {relation.relation_type}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {targetTicket.identifier}
                </span>
                <span className="flex-1 truncate">{targetTicket.title}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default RelationsList;
