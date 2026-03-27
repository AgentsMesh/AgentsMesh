"use client";

import React from "react";
import { cn } from "@/lib/utils";
import type { PodData } from "@/lib/api/pod";
import { getPodDisplayName } from "@/lib/pod-utils";
import { getPodStatusInfo } from "@/stores/mesh";
import { Link2 } from "lucide-react";

interface RelatedPodsListProps {
  relatedPods: PodData[];
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function RelatedPodsList({ relatedPods, t }: RelatedPodsListProps) {
  return (
    <div className="border-t border-border pt-2">
      <div className="flex items-center gap-1.5 mb-1.5">
        <Link2 className="w-3 h-3 text-muted-foreground" />
        <span className="text-xs font-medium">
          {t("ide.bottomPanel.infoTab.relatedPods", {
            count: relatedPods.length,
          })}
        </span>
      </div>
      <div className="space-y-1">
        {relatedPods.map((rp) => {
          const rpStatus = getPodStatusInfo(rp.status);
          return (
            <div
              key={rp.pod_key}
              className="flex items-center gap-2 px-2 py-1 rounded bg-muted/50 text-xs"
            >
              <span
                className={cn(
                  "w-1.5 h-1.5 rounded-full flex-shrink-0",
                  rpStatus.bgColor
                )}
              />
              <span className="truncate flex-1">
                {getPodDisplayName(rp)}
              </span>
              <span
                className={cn(
                  "text-[10px] whitespace-nowrap",
                  rpStatus.color
                )}
              >
                {rpStatus.label}
              </span>
              {rp.agent && (
                <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                  {rp.agent.name}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
