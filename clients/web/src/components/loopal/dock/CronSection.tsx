"use client";

import { useTranslations } from "next-intl";
import { Trash2 } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { loopalControl } from "../loopalControl";
import { DockList, type Confirm } from "./DockList";
import { StatusPill } from "./StatusPill";

type T = (k: string, v?: Record<string, string | number | Date>) => string;

function relTime(ms: number | null, t: T): string {
  if (!ms) return "";
  const d = ms - Date.now();
  if (d <= 0) return t("dock.cron.due");
  const m = Math.round(d / 60000);
  if (m < 60) return t("dock.cron.inMinutes", { m });
  const h = Math.round(m / 60);
  if (h < 24) return t("dock.cron.inHours", { h });
  return t("dock.cron.inDays", { d: Math.round(h / 24) });
}

export function CronSection({ podKey, confirm }: { podKey: string; confirm: Confirm }) {
  const t = useTranslations("loopal");
  const { crons } = useLoopalSession(podKey);

  async function del(id: string, expr: string) {
    const ok = await confirm({
      title: t("dock.cron.deleteConfirmTitle"),
      description: expr,
      variant: "destructive",
      confirmText: t("dock.cron.delete"),
    });
    if (ok) loopalControl(podKey, "loopal.cronDelete", { id });
  }

  return (
    <DockList>
      {crons.map((c) => (
        <div key={c.id} className="rounded-md border border-border px-2 py-1.5">
          <div className="flex items-center justify-between gap-2">
            <span className="min-w-0 flex-1 truncate font-mono text-xs">{c.cron_expr || "—"}</span>
            <div className="flex shrink-0 items-center gap-1.5">
              <StatusPill label={c.recurring ? t("dock.cron.recurring") : t("dock.cron.once")} tone="bg-muted text-muted-foreground" />
              {c.durable && <StatusPill label={t("dock.cron.durable")} tone="bg-muted text-muted-foreground" />}
              {c.next_fire_unix_ms ? (
                <span className="text-[10px] text-muted-foreground">{relTime(c.next_fire_unix_ms, t)}</span>
              ) : null}
              <button
                type="button"
                onClick={() => del(c.id, c.cron_expr)}
                className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                title={t("dock.cron.delete")}
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          </div>
          {c.prompt && <div className="mt-1 truncate text-[11px] text-muted-foreground">{c.prompt}</div>}
        </div>
      ))}
    </DockList>
  );
}
