"use client";

import { memo } from "react";
import { useTranslations } from "next-intl";
import { Handle, Position } from "@xyflow/react";

interface LoopalAgentNodeData {
  name: string;
  model: string | null;
  isRoot: boolean;
}

function LoopalAgentNodeBase({ data }: { data: LoopalAgentNodeData }) {
  const t = useTranslations("loopal");
  return (
    <div
      data-testid="loopal-agent-node"
      data-agent-name={data.name}
      className="min-w-[180px] rounded-lg border-2 border-border bg-background px-3 py-2 shadow-md"
    >
      <Handle type="target" position={Position.Top} className="!bg-primary" />
      <Handle type="source" position={Position.Bottom} className="!bg-primary" />
      <div className="flex items-center justify-between gap-2">
        <code className="truncate font-mono text-xs text-foreground">{data.name}</code>
        {data.isRoot && (
          <span className="shrink-0 rounded bg-muted px-1 text-[10px] text-muted-foreground">{t("topology.root")}</span>
        )}
      </div>
      {data.model && (
        <div className="mt-1 truncate font-mono text-[10px] text-muted-foreground">{data.model}</div>
      )}
    </div>
  );
}

export default memo(LoopalAgentNodeBase);
