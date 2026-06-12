"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Terminal, Clock, ListChecks, Network, Plug } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { useConfirmDialog } from "@/components/ui/use-confirm-dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { BgShellSection } from "./BgShellSection";
import { CronSection } from "./CronSection";
import { TaskSection } from "./TaskSection";
import { AgentsSection } from "./AgentsSection";
import { McpSection } from "./McpSection";

// Bottom dock: one tab per non-empty data class, summoned by click (slides the
// panel up above the tab strip). The whole dock is absent when every class is
// empty, so a fresh pod stays a clean conversation surface.
export function LoopalBottomDock({ podKey, onExpandTopology }: { podKey: string; onExpandTopology?: () => void }) {
  const t = useTranslations("loopal");
  const session = useLoopalSession(podKey);
  const { dialogProps, confirm } = useConfirmDialog();
  const [active, setActive] = useState<string | null>(null);

  // bg/tasks count only active items (Running shells, non-completed tasks) —
  // matching loopal's panels which hide finished items; cron/agents/mcp have no
  // terminal state so they count all.
  const panels = [
    { id: "bg", label: t("dock.tabs.bgShell"), Icon: Terminal,
      count: session.bg_tasks.filter((b) => b.status === "Running").length,
      render: () => <BgShellSection podKey={podKey} confirm={confirm} /> },
    { id: "cron", label: t("dock.tabs.cron"), Icon: Clock, count: session.crons.length,
      render: () => <CronSection podKey={podKey} confirm={confirm} /> },
    { id: "tasks", label: t("dock.tabs.tasks"), Icon: ListChecks,
      count: session.tasks.filter((tk) => tk.status !== "completed").length,
      render: () => <TaskSection podKey={podKey} /> },
    { id: "agents", label: t("dock.tabs.agents"), Icon: Network, count: session.topology.length,
      render: () => <AgentsSection podKey={podKey} onExpand={onExpandTopology} /> },
    { id: "mcp", label: t("dock.tabs.mcp"), Icon: Plug, count: session.mcp.length,
      render: () => <McpSection podKey={podKey} confirm={confirm} /> },
  ].filter((p) => p.count > 0);

  if (panels.length === 0) return null;

  // A tab whose data drained to empty is filtered out above; find() then yields
  // undefined and the panel area collapses (graceful — the data is genuinely
  // gone). The selection is kept so the panel re-opens if that class repopulates.
  const activePanel = panels.find((p) => p.id === active);

  return (
    <div className="shrink-0 border-t border-border bg-background">
      {activePanel && (
        <div className="max-h-72 overflow-auto border-b border-border">{activePanel.render()}</div>
      )}
      <div className="flex items-stretch overflow-x-auto">
        {panels.map((p) => {
          const on = p.id === active;
          const Icon = p.Icon;
          return (
            <button
              key={p.id}
              type="button"
              onClick={() => setActive(on ? null : p.id)}
              data-testid={`loopal-dock-tab-${p.id}`}
              className={`flex items-center gap-1.5 whitespace-nowrap border-r border-border px-3 py-1.5 text-xs transition-colors ${
                on ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              <span>{p.label}</span>
              <span className="rounded bg-foreground/10 px-1 text-[10px] tabular-nums">{p.count}</span>
            </button>
          );
        })}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
