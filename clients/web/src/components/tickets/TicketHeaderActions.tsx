"use client";

import { useTranslations } from "next-intl";
import { MoreHorizontal, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface TicketHeaderActionsProps {
  onEdit: () => void;
  onMarkDone: () => void;
  onDelete: () => void;
}

export function TicketHeaderActions({ onEdit, onMarkDone, onDelete }: TicketHeaderActionsProps) {
  const t = useTranslations();

  return (
    <div className="flex shrink-0 items-center gap-1.5">
      <Button variant="outline" size="sm" className="h-7 px-3 text-xs" onClick={onEdit}>
        {t("common.edit")}
      </Button>
      <Button variant="outline" size="sm" className="h-7 px-3 text-xs" onClick={onMarkDone}>
        {t("tickets.detail.markDone")}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon" className="h-7 w-7" aria-label={t("common.more")}>
            <MoreHorizontal className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            className="text-destructive focus:text-destructive"
            onClick={onDelete}
          >
            <Trash2 className="w-4 h-4 mr-2" />
            {t("tickets.detail.deleteTicket")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
