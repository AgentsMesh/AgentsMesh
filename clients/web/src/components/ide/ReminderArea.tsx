"use client";

import {
  Tooltip,
  TooltipContent,
  TooltipPortal,
  TooltipTrigger,
} from "@radix-ui/react-tooltip";
import { useTranslations } from "next-intl";
import { Info, RefreshCw, ArrowUpCircle, X, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { useVisibleReminders, useReminderStore, type ReminderTone } from "@/stores/reminders";

const TONE_CLASS: Record<ReminderTone, string> = {
  info: "text-blue-600 dark:text-blue-400 hover:bg-blue-500/10",
  warning: "text-amber-600 dark:text-amber-400 hover:bg-amber-500/10",
  success: "text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/10",
};

const TONE_ICON: Record<ReminderTone, LucideIcon> = {
  info: Info,
  warning: RefreshCw,
  success: ArrowUpCircle,
};

export function ReminderArea() {
  const reminders = useVisibleReminders();
  const dismiss = useReminderStore((s) => s.dismiss);
  const t = useTranslations();

  if (reminders.length === 0) return null;

  return (
    <nav className="flex flex-col items-stretch py-2 gap-0.5 px-2 border-t border-border">
      {reminders.map((r) => {
        const Icon = TONE_ICON[r.tone];
        return (
          <Tooltip key={r.id}>
            <TooltipTrigger asChild>
              <div
                role={r.onAction ? "button" : "status"}
                tabIndex={r.onAction ? 0 : undefined}
                onClick={r.onAction}
                onKeyDown={
                  r.onAction
                    ? (e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          r.onAction!();
                        }
                      }
                    : undefined
                }
                className={cn(
                  "group w-full h-9 px-2 flex items-center gap-2 rounded-md transition-colors",
                  TONE_CLASS[r.tone],
                  r.onAction && "cursor-pointer",
                )}
              >
                <Icon className="w-4 h-4 shrink-0" />
                <span className="flex-1 text-xs leading-tight font-medium truncate">{r.message}</span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    dismiss(r.id);
                  }}
                  aria-label={t("reminders.dismiss")}
                  className="shrink-0 rounded p-0.5 opacity-0 transition group-hover:opacity-100 hover:bg-black/10 dark:hover:bg-white/10"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            </TooltipTrigger>
            <TooltipPortal>
              <TooltipContent
                side="right"
                className="z-50 max-w-xs rounded border border-border bg-popover px-2 py-1 text-sm text-popover-foreground shadow-md"
              >
                {r.message}
              </TooltipContent>
            </TooltipPortal>
          </Tooltip>
        );
      })}
    </nav>
  );
}

export default ReminderArea;
