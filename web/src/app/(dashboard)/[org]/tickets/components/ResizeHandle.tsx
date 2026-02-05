"use client";

import { Separator } from "react-resizable-panels";
import { GripVertical, GripHorizontal } from "lucide-react";
import { cn } from "@/lib/utils";

interface ResizeHandleProps {
  direction: "horizontal" | "vertical";
}

/**
 * VS Code style resize handle - hidden by default, highlights on hover
 */
export function ResizeHandle({ direction }: ResizeHandleProps) {
  const isHorizontal = direction === "horizontal";

  return (
    <Separator
      className={cn(
        "group relative flex items-center justify-center bg-transparent transition-colors",
        isHorizontal
          ? "w-1 cursor-col-resize hover:bg-primary"
          : "h-1 cursor-row-resize hover:bg-primary"
      )}
    >
      {/* Expand hit area */}
      <div
        className={cn(
          "absolute z-10",
          isHorizontal ? "w-3 h-full -left-1" : "h-3 w-full -top-1"
        )}
      />
      {/* Grip indicator */}
      <div className={cn(
        "opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground"
      )}>
        {isHorizontal ? (
          <GripVertical className="h-4 w-4" />
        ) : (
          <GripHorizontal className="h-4 w-4" />
        )}
      </div>
    </Separator>
  );
}
