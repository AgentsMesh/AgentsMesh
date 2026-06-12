"use client";

import { useTranslations } from "next-intl";
import { useLoopalSession } from "@/stores/loopalConsole";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { loopalControl } from "../loopalControl";

// Mode badge shows loopal's real act/plan mode (from the _loopal/mode event,
// not optimistic). Plan is tinted purple per loopal TUI; click to switch.
export function LoopalModeBadge({ podKey }: { podKey: string }) {
  const t = useTranslations("loopal");
  const { mode } = useLoopalSession(podKey);
  const label = mode === "plan" ? t("status.mode.plan") : mode === "act" ? t("status.mode.act") : "—";
  const tone =
    mode === "plan"
      ? "bg-purple-500/15 text-purple-600"
      : mode === "act"
        ? "bg-muted text-foreground"
        : "bg-muted text-muted-foreground";
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="loopal-mode-badge"
          className={`rounded px-1.5 py-0.5 text-[11px] font-medium ${tone}`}
        >
          {label}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem onClick={() => loopalControl(podKey, "loopal.mode", { mode: "act" })}>
          {t("status.mode.act")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => loopalControl(podKey, "loopal.mode", { mode: "plan" })}>
          {t("status.mode.plan")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
