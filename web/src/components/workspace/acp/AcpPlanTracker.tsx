"use client";

import { useAcpSessionStore } from "@/stores/acpSession";
import { CheckCircle2, Circle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export function AcpPlanTracker({ podKey }: { podKey: string }) {
  const plan = useAcpSessionStore((s) => s.sessions[podKey]?.plan);

  if (!plan || plan.length === 0) return null;

  return (
    <div className="border-b px-4 py-2">
      <div className="flex items-center gap-1.5 flex-wrap">
        <span className="text-xs font-medium text-muted-foreground mr-1">Plan</span>
        {plan.map((step, i) => {
          const isCompleted = step.status === "completed";
          const isInProgress = step.status === "in_progress";

          return (
            <span
              key={i}
              className={cn(
                "inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full",
                isCompleted && "bg-green-500/10 text-green-600 dark:text-green-400",
                isInProgress && "bg-blue-500/10 text-blue-600 dark:text-blue-400",
                !isCompleted && !isInProgress && "bg-muted text-muted-foreground",
              )}
            >
              {isCompleted ? (
                <CheckCircle2 className="h-3 w-3" />
              ) : isInProgress ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Circle className="h-3 w-3" />
              )}
              {step.title}
            </span>
          );
        })}
      </div>
    </div>
  );
}
