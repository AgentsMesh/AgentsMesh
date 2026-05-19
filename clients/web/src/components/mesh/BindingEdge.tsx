"use client";

import { memo } from "react";
import { BaseEdge, EdgeLabelRenderer, getBezierPath, type Position } from "@xyflow/react";
import { getBindingStatusInfo } from "@/stores/mesh";

interface BindingEdgeProps {
  id: string;
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  sourcePosition: Position;
  targetPosition: Position;
  data?: {
    status?: string;
    grantedScopes?: string[];
    pendingScopes?: string[];
  };
  selected?: boolean;
}

function BindingEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected,
}: BindingEdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const statusInfo = getBindingStatusInfo(data?.status || "active");
  const grantedScopes = data?.grantedScopes ?? [];
  const pendingScopes = data?.pendingScopes ?? [];
  const scopeCount = grantedScopes.length + pendingScopes.length;

  const isWrite = grantedScopes.some((s) => s.endsWith(":write") || s === "pod:write");
  const isPending = data?.status === "pending";

  const strokeWidth = selected ? 3 : isWrite ? 2 : 1;
  const dash = isPending ? "5 5" : isWrite ? undefined : "4 4";

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        className={`${statusInfo.color} ${isWrite ? "opacity-100" : "opacity-60"}`}
        style={{
          strokeWidth,
          strokeDasharray: dash,
        }}
      />
      {scopeCount > 0 && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: "absolute",
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: "all",
            }}
            className="nodrag nopan"
          >
            <div
              className={`px-2 py-0.5 text-xs rounded-full bg-background border border-border shadow-xs ${
                selected ? "ring-1 ring-primary" : ""
              }`}
              title={grantedScopes.join(", ") || undefined}
            >
              {isWrite ? "W" : "R"} · {scopeCount}
            </div>
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}

export default memo(BindingEdge);
