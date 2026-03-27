import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  ArrowLeft,
  Play,
  Pencil,
  Loader2,
  MoreHorizontal,
  Power,
  Trash2,
} from "lucide-react";
import type { LoopData } from "@/stores/loop";

interface LoopHeaderProps {
  loop: LoopData;
  orgSlug: string;
  triggering: boolean;
  t: (key: string) => string;
  onBack: () => void;
  onTrigger: () => void;
  onEdit: () => void;
  onEnable: () => void;
  onDisable: () => void;
  onDelete: () => void;
}

export function LoopHeader({
  loop,
  orgSlug,
  triggering,
  t,
  onBack,
  onTrigger,
  onEdit,
  onEnable,
  onDisable,
  onDelete,
}: LoopHeaderProps) {
  const isEnabled = loop.status === "enabled";

  return (
    <div className="mb-8">
      <button
        className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-4"
        onClick={onBack}
      >
        <ArrowLeft className="w-3.5 h-3.5" />
        {t("loops.back")}
      </button>

      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="min-w-0">
          <div className="flex items-center gap-3 mb-1.5">
            <h1 className="text-xl font-bold truncate">{loop.name}</h1>
            <StatusBadge isEnabled={isEnabled} t={t} />
          </div>
          {loop.description && (
            <p className="text-sm text-muted-foreground">{loop.description}</p>
          )}
        </div>

        <div className="flex gap-2 flex-shrink-0 w-full sm:w-auto">
          {isEnabled && (
            <Button size="sm" onClick={onTrigger}
              disabled={triggering || loop.active_run_count >= loop.max_concurrent_runs}
              className="gap-1.5">
              {triggering ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5" />
              )}
              {t("loops.trigger")}
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={onEdit} className="gap-1.5">
            <Pencil className="w-3.5 h-3.5" />
            {t("common.edit")}
          </Button>
          <ActionsDropdown
            isEnabled={isEnabled}
            t={t}
            onEnable={onEnable}
            onDisable={onDisable}
            onDelete={onDelete}
          />
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ isEnabled, t }: { isEnabled: boolean; t: (key: string) => string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium flex-shrink-0",
        isEnabled
          ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
          : "bg-gray-500/10 text-gray-600 dark:text-gray-400"
      )}
    >
      <span className={cn("w-1.5 h-1.5 rounded-full", isEnabled ? "bg-emerald-500" : "bg-gray-400")} />
      {isEnabled ? t("loops.statusEnabled") : t("loops.statusDisabled")}
    </span>
  );
}

function ActionsDropdown({
  isEnabled,
  t,
  onEnable,
  onDisable,
  onDelete,
}: {
  isEnabled: boolean;
  t: (key: string) => string;
  onEnable: () => void;
  onDisable: () => void;
  onDelete: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="sm" variant="ghost" className="h-8 w-8 p-0">
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {isEnabled ? (
          <DropdownMenuItem onClick={onDisable}>
            <Power className="w-4 h-4 mr-2" />
            {t("loops.disable")}
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem onClick={onEnable}>
            <Power className="w-4 h-4 mr-2" />
            {t("loops.enable")}
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={onDelete}>
          <Trash2 className="w-4 h-4 mr-2" />
          {t("common.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
