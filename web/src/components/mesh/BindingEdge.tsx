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
  const scopeCount = (data?.grantedScopes?.length || 0) + (data?.pendingScopes?.length || 0);

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        className={`${statusInfo.color} ${selected ? "stroke-[3px]" : "stroke-2"}`}
        style={{
          strokeDasharray: data?.status === "pending" ? "5 5" : undefined,
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
              className={`px-2 py-1 text-xs rounded-full bg-background border border-border shadow-sm ${
                selected ? "ring-2 ring-primary/20" : ""
              }`}
              title={data?.grantedScopes?.join(", ")}
            >
              {scopeCount} scope{scopeCount !== 1 ? "s" : ""}
            </div>
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}

export default memo(BindingEdge);
