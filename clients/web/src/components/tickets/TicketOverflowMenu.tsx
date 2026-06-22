"use client";

import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface TicketOverflowMenuProps {
  onDelete: () => void;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function TicketOverflowMenu({ onDelete, t }: TicketOverflowMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon" className="h-7 w-7" aria-label={t("common.more")}>
          ⋯
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onDelete} className="cursor-pointer text-destructive focus:text-destructive">
          <Trash2 className="h-4 w-4" />
          {t("common.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default TicketOverflowMenu;
