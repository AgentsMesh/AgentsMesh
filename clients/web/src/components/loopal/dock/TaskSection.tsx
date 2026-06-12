"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Circle, CircleDot, CheckCircle2 } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { DockList } from "./DockList";

const ICON: Record<string, { Icon: typeof Circle; cls: string }> = {
  pending: { Icon: Circle, cls: "text-muted-foreground" },
  in_progress: { Icon: CircleDot, cls: "text-blue-500" },
  completed: { Icon: CheckCircle2, cls: "text-green-500" },
};

export function TaskSection({ podKey }: { podKey: string }) {
  const t = useTranslations("loopal");
  const { tasks } = useLoopalSession(podKey);
  const [open, setOpen] = useState<string | null>(null);

  return (
    <DockList>
      {/* Hide completed — mirrors loopal's tasks_panel (renders non-Completed). */}
      {tasks.filter((task) => task.status !== "completed").map((task) => {
        const { Icon, cls } = ICON[task.status] ?? ICON.pending;
        const blocked = task.blocked_by.length;
        return (
          <div key={task.id} className="rounded-md border border-border px-2 py-1.5">
            <button
              type="button"
              onClick={() => {
                if (blocked) setOpen(open === task.id ? null : task.id);
              }}
              className="flex w-full items-center gap-2 text-left"
            >
              <Icon className={`h-3.5 w-3.5 shrink-0 ${cls}`} />
              <span className="min-w-0 flex-1 truncate text-xs">{task.subject}</span>
              {blocked > 0 && (
                <span className="shrink-0 rounded bg-yellow-500/15 px-1.5 py-0.5 text-[10px] text-yellow-700">
                  {t("dock.tasks.blockedBy", { count: blocked })}
                </span>
              )}
            </button>
            {open === task.id && blocked > 0 && (
              <div className="mt-1 border-t border-border pt-1 font-mono text-[11px] text-muted-foreground">
                {task.blocked_by.join(", ")}
              </div>
            )}
          </div>
        );
      })}
    </DockList>
  );
}
