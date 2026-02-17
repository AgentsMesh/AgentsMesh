"use client";

import { memo } from "react";
import { type NodeProps } from "@xyflow/react";
import { useTranslations } from "next-intl";

interface RunnerGroupData {
  runnerNodeId: string;
  runnerStatus: string;
  podCount: number;
}

function RunnerGroupNode({ data }: NodeProps) {
  const t = useTranslations("mesh");
  const { runnerNodeId, runnerStatus, podCount } = data as unknown as RunnerGroupData;
  const isOnline = runnerStatus === "online";

  return (
    <div className="rounded-lg border-2 border-dashed border-muted-foreground/30 bg-muted/20 min-w-[440px]">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-dashed border-muted-foreground/20">
        <div
          className={`w-2 h-2 rounded-full ${
            isOnline ? "bg-green-500" : "bg-gray-400"
          }`}
        />
        <span className="text-sm font-medium text-foreground truncate">
          {runnerNodeId}
        </span>
        <span className="text-xs text-muted-foreground">
          {isOnline ? t("runnerGroup.online") : t("runnerGroup.offline")}
        </span>
        <span className="ml-auto text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
          {t("runnerGroup.podCount", { count: podCount })}
        </span>
      </div>
      {/* Content area - pods are placed here via parentId mechanism */}
      <div className="p-4 min-h-[160px]" />
    </div>
  );
}

export default memo(RunnerGroupNode);
