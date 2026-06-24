"use client";

import { useTicketStore, TicketViewMode } from "@/stores/ticket";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { LayoutGrid, List, Download, Plus } from "lucide-react";

interface TicketsPageHeaderProps {
  onCreateClick?: () => void;
  onExportClick?: () => void;
  rightExtra?: React.ReactNode;
}

export function TicketsPageHeader({ onCreateClick, onExportClick, rightExtra }: TicketsPageHeaderProps) {
  const t = useTranslations();
  const viewMode = useTicketStore((s) => s.viewMode);
  const setViewMode = useTicketStore((s) => s.setViewMode);

  const tab = (mode: TicketViewMode, Icon: typeof LayoutGrid, label: string) => (
    <button
      type="button"
      onClick={() => setViewMode(mode)}
      className={cn(
        "flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors",
        viewMode === mode
          ? "bg-background text-foreground shadow-xs"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      <Icon className="h-3.5 w-3.5" />
      {label}
    </button>
  );

  return (
    <header className="app-drag flex items-center justify-between gap-4 border-b border-border px-6 py-3">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold tracking-tight text-foreground">{t("tickets.title")}</h1>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-0.5 rounded-md border border-border bg-muted p-0.5">
          {tab("board", LayoutGrid, t("tickets.views.board"))}
          {tab("list", List, t("tickets.views.list"))}
        </div>
        {onExportClick && (
          <Button variant="outline" size="sm" onClick={onExportClick} className="h-7 gap-1.5">
            <Download className="h-3.5 w-3.5" />
            {t("tickets.actions.export")}
          </Button>
        )}
        {onCreateClick && (
          <Button size="sm" onClick={onCreateClick} className="h-7 gap-1.5">
            <Plus className="h-3.5 w-3.5" />
            {t("tickets.actions.newTicket")}
          </Button>
        )}
        {rightExtra}
      </div>
    </header>
  );
}
