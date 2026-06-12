"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Square } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { loopalControl } from "../loopalControl";
import { DockList, type Confirm } from "./DockList";
import { StatusPill } from "./StatusPill";

const TONE: Record<string, string> = {
  Running: "bg-blue-500/15 text-blue-600",
  Completed: "bg-green-500/15 text-green-600",
  Failed: "bg-red-500/15 text-red-600",
  Killed: "bg-yellow-500/15 text-yellow-700",
};

export function BgShellSection({ podKey, confirm }: { podKey: string; confirm: Confirm }) {
  const t = useTranslations("loopal");
  const { bg_tasks } = useLoopalSession(podKey);
  const [open, setOpen] = useState<string | null>(null);

  async function kill(id: string, description: string) {
    const ok = await confirm({
      title: t("dock.bgShell.killConfirmTitle"),
      description,
      variant: "destructive",
      confirmText: t("dock.bgShell.kill"),
    });
    if (ok) loopalControl(podKey, "loopal.bgTaskKill", { id });
  }

  return (
    <DockList>
      {/* Only Running shells render — mirrors loopal's bg_tasks_panel
          (render_bg_tasks filters Running); completed output lands in activity. */}
      {bg_tasks.filter((bt) => bt.status === "Running").map((bt) => {
        const exit = bt.exit_code != null && bt.exit_code !== 0 ? ` (${bt.exit_code})` : "";
        return (
          <div key={bt.id} className="rounded-md border border-border">
            <div className="flex items-center justify-between gap-2 px-2 py-1.5">
              <button
                type="button"
                onClick={() => setOpen(open === bt.id ? null : bt.id)}
                className="min-w-0 flex-1 truncate text-left font-mono text-xs hover:text-foreground"
                title={bt.description}
              >
                {bt.description}
              </button>
              <div className="flex shrink-0 items-center gap-1.5">
                <StatusPill label={bt.status + exit} tone={TONE[bt.status] ?? "bg-muted text-muted-foreground"} />
                {bt.status === "Running" && (
                  <button
                    type="button"
                    onClick={() => kill(bt.id, bt.description)}
                    className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                    title={t("dock.bgShell.kill")}
                  >
                    <Square className="h-3 w-3" />
                  </button>
                )}
              </div>
            </div>
            {open === bt.id && bt.output && (
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap border-t border-border bg-muted/30 px-2 py-1.5 text-[11px] text-muted-foreground">
                {bt.output}
              </pre>
            )}
          </div>
        );
      })}
    </DockList>
  );
}
