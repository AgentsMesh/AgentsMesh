"use client";

import { useTranslations } from "next-intl";
import { RotateCw, Unplug } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { loopalControl } from "../loopalControl";
import { DockList, type Confirm } from "./DockList";
import { StatusPill } from "./StatusPill";

const TONE: Record<string, string> = {
  connected: "bg-green-500/15 text-green-600",
  error: "bg-red-500/15 text-red-600",
};

export function McpSection({ podKey, confirm }: { podKey: string; confirm: Confirm }) {
  const t = useTranslations("loopal");
  const { mcp } = useLoopalSession(podKey);

  async function disconnect(server: string) {
    const ok = await confirm({
      title: t("dock.mcp.disconnectConfirmTitle"),
      description: server,
      variant: "destructive",
      confirmText: t("dock.mcp.disconnect"),
    });
    if (ok) loopalControl(podKey, "loopal.mcpDisconnect", { server });
  }

  return (
    <DockList>
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => loopalControl(podKey, "loopal.mcpStatus")}
          className="flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] text-muted-foreground hover:text-foreground"
        >
          <RotateCw className="h-3 w-3" /> {t("dock.mcp.refresh")}
        </button>
      </div>
      {mcp.map((s) => (
        <div
          key={s.name}
          className="flex items-center justify-between gap-2 rounded-md border border-border px-2 py-1.5"
        >
          <span className="min-w-0 flex-1 truncate font-mono text-xs">{s.name}</span>
          <div className="flex shrink-0 items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground">{t("dock.mcp.toolCount", { count: s.tool_count })}</span>
            <StatusPill label={s.status} tone={TONE[s.status] ?? "bg-muted text-muted-foreground"} />
            <button
              type="button"
              onClick={() => loopalControl(podKey, "loopal.mcpReconnect", { server: s.name })}
              className="rounded p-1 text-muted-foreground hover:text-foreground"
              title={t("dock.mcp.reconnect")}
            >
              <RotateCw className="h-3 w-3" />
            </button>
            <button
              type="button"
              onClick={() => disconnect(s.name)}
              className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
              title={t("dock.mcp.disconnect")}
            >
              <Unplug className="h-3 w-3" />
            </button>
          </div>
        </div>
      ))}
    </DockList>
  );
}
