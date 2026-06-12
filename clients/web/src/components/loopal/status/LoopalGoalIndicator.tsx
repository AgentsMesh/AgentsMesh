"use client";

import { useTranslations } from "next-intl";
import { Target } from "lucide-react";
import { useLoopalSession } from "@/stores/loopalConsole";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { loopalControl } from "../loopalControl";
import { goalStatusTone } from "./loopalGoalStatus";

const ACTIONS: ReadonlyArray<[string, string]> = [
  ["pause", "loopal.goalPause"],
  ["resume", "loopal.goalResume"],
  ["complete", "loopal.goalComplete"],
  ["reopen", "loopal.goalReopen"],
  ["clear", "loopal.goalClear"],
];

// Top-bar goal indicator: ◆ objective [status], colored by status. Absent when
// no goal exists. Click opens the goal lifecycle menu (loopal's /goal actions).
export function LoopalGoalIndicator({ podKey }: { podKey: string }) {
  const t = useTranslations("loopal");
  const { thread_goal } = useLoopalSession(podKey);
  if (!thread_goal) return null;
  const tone = goalStatusTone(thread_goal.status);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="loopal-goal-indicator"
          className="flex min-w-0 items-center gap-1.5 rounded px-2 py-1 text-xs hover:bg-muted"
        >
          <Target className={`h-3.5 w-3.5 shrink-0 ${tone}`} />
          <span className="min-w-0 max-w-[32ch] truncate text-foreground">{thread_goal.objective}</span>
          <span className={`shrink-0 ${tone}`}>[{thread_goal.status}]</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {ACTIONS.map(([key, subtype]) => (
          <DropdownMenuItem
            key={subtype}
            onClick={() => loopalControl(podKey, subtype)}
            data-testid={`loopal-goal-${key}`}
          >
            {t(`status.goal.${key}`)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
