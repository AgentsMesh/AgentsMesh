"use client";

import { useTranslations } from "next-intl";
import { Network } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import { useTerminalStatus } from "@/hooks/useTerminalStatus";
import { thinkingKey } from "../prompt/loopalThinking";
import { LoopalGoalIndicator } from "./LoopalGoalIndicator";
import { LoopalModeBadge } from "./LoopalModeBadge";
import { AcpPermissionModeSelector } from "@/components/workspace/acp/AcpPermissionModeSelector";

const CONN_TONE: Record<string, string> = {
  connected: "bg-green-500",
  connecting: "bg-yellow-500",
  disconnected: "bg-red-500",
  error: "bg-red-500",
  none: "bg-muted-foreground",
};

// Status bar: real mode/thinking/model (single source: loopalSession) + goal
// indicator + live connection dot. Replaces the old control bar / session
// controls / goal panel — those controls now live in slash commands.
export function LoopalTopBar({ podKey, onOpenTopology }: { podKey: string; onOpenTopology?: () => void }) {
  const t = useTranslations("loopal");
  const { thinking, model, topology } = useLoopalSession(podKey);
  const { status } = useTerminalStatus(podKey);
  const thinkKey = thinkingKey(thinking);

  return (
    <header className="flex items-center justify-between gap-3 border-b border-border px-6 py-2.5">
      <div className="flex min-w-0 items-center gap-3">
        <h1 className="shrink-0 text-[15px] font-semibold text-foreground">{t("status.title")}</h1>
        <LoopalGoalIndicator podKey={podKey} />
      </div>
      <div className="flex shrink-0 items-center gap-2.5 text-xs text-muted-foreground">
        <LoopalModeBadge podKey={podKey} />
        <AcpPermissionModeSelector podKey={podKey} />
        {thinkKey && <span data-testid="loopal-thinking">{t("status.thinking", { value: t("thinking." + thinkKey) })}</span>}
        {model && <span className="font-mono" data-testid="loopal-model">{model}</span>}
        <span className="h-2 w-2 rounded-full" title={status}>
          <span className={`block h-2 w-2 rounded-full ${CONN_TONE[status] ?? CONN_TONE.none}`} />
        </span>
        <span className="font-mono">{podKey}</span>
        {onOpenTopology && topology.length > 0 && (
          <button
            type="button"
            onClick={onOpenTopology}
            title={t("status.topology")}
            data-testid="loopal-topbar-topology"
            className="rounded p-1 hover:bg-muted hover:text-foreground"
          >
            <Network className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </header>
  );
}
