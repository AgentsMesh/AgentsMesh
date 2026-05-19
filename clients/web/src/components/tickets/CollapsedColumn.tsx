"use client";

import { useDroppable } from "@dnd-kit/core";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import type { TicketStatus } from "@/stores/ticket";

interface CollapsedColumnProps {
  status: TicketStatus;
  labelKey: string;
  topColor: string;
  dotColor: string;
  totalCount: number;
  isOver: boolean;
  onExpand: () => void;
  t: (key: string) => string;
}

export function CollapsedColumn({
  status, labelKey, topColor, dotColor, totalCount, isOver, onExpand, t,
}: CollapsedColumnProps) {
  const { setNodeRef, isOver: isDroppableOver } = useDroppable({ id: status });
  const highlighted = isOver || isDroppableOver;

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex-shrink-0 w-16 flex flex-col items-center rounded-lg bg-muted/30",
        "transition-all duration-200 overflow-hidden cursor-pointer group",
        highlighted && "ring-2 ring-primary/50 bg-primary/5",
      )}
      onClick={onExpand}
    >
      <div className={cn("h-1 w-full", topColor)} />
      <div className="flex flex-col items-center gap-2 py-4 px-1">
        <div className={cn("w-2.5 h-2.5 rounded-full", dotColor)} />
        <span className="text-xs font-medium text-muted-foreground [writing-mode:vertical-lr]">
          {t(labelKey)}
        </span>
        <span className="text-sm font-mono font-semibold text-muted-foreground">
          {totalCount}
        </span>
        <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/50 group-hover:text-foreground transition-colors" />
      </div>
    </div>
  );
}
