"use client";

import { cn } from "@/lib/utils";
import { Terminal, ArrowRight, ArrowLeft } from "lucide-react";
import { getShortPodKey } from "@/lib/pod-utils";
import type { ActivityTabContentProps } from "./types";

/**
 * Activity tab content - shows incoming/outgoing bindings for selected pod
 */
export function ActivityTabContent({
  selectedPodKey,
  incomingBindings,
  outgoingBindings,
  getPodInfo,
  t,
}: ActivityTabContentProps) {
  // No pod selected
  if (!selectedPodKey) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
        <Terminal className="w-4 h-4 mr-2" />
        <span>{t("ide.bottomPanel.selectPodFirst")}</span>
      </div>
    );
  }

  const hasBindings = incomingBindings.length > 0 || outgoingBindings.length > 0;

  if (!hasBindings) {
    return (
      <div className="text-xs text-muted-foreground">
        <p>{t("ide.bottomPanel.noBindings")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Incoming bindings */}
      {incomingBindings.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-2 flex items-center gap-1">
            <ArrowRight className="w-3 h-3" />
            {t("ide.bottomPanel.incomingBindings")} ({incomingBindings.length})
          </h4>
          <div className="space-y-1">
            {incomingBindings.map((edge) => {
              const sourcePod = getPodInfo(edge.source);
              return (
                <div
                  key={`${edge.source}-${edge.target}`}
                  className="flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 text-xs"
                >
                  <span className="font-mono text-muted-foreground">
                    {getShortPodKey(edge.source)}
                  </span>
                  {sourcePod?.model && (
                    <span className="text-muted-foreground">
                      ({sourcePod.model})
                    </span>
                  )}
                  <ArrowRight className="w-3 h-3 text-green-500" />
                  <span className="font-medium">{t("ide.bottomPanel.currentPod")}</span>
                  <span className={cn(
                    "ml-auto px-1.5 py-0.5 rounded text-[10px]",
                    edge.status === "active"
                      ? "bg-green-500/10 text-green-500"
                      : "bg-yellow-500/10 text-yellow-500"
                  )}>
                    {edge.status}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Outgoing bindings */}
      {outgoingBindings.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-2 flex items-center gap-1">
            <ArrowLeft className="w-3 h-3" />
            {t("ide.bottomPanel.outgoingBindings")} ({outgoingBindings.length})
          </h4>
          <div className="space-y-1">
            {outgoingBindings.map((edge) => {
              const targetPod = getPodInfo(edge.target);
              return (
                <div
                  key={`${edge.source}-${edge.target}`}
                  className="flex items-center gap-2 px-2 py-1.5 rounded bg-muted/50 text-xs"
                >
                  <span className="font-medium">{t("ide.bottomPanel.currentPod")}</span>
                  <ArrowRight className="w-3 h-3 text-blue-500" />
                  <span className="font-mono text-muted-foreground">
                    {getShortPodKey(edge.target)}
                  </span>
                  {targetPod?.model && (
                    <span className="text-muted-foreground">
                      ({targetPod.model})
                    </span>
                  )}
                  <span className={cn(
                    "ml-auto px-1.5 py-0.5 rounded text-[10px]",
                    edge.status === "active"
                      ? "bg-green-500/10 text-green-500"
                      : "bg-yellow-500/10 text-yellow-500"
                  )}>
                    {edge.status}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

export default ActivityTabContent;
