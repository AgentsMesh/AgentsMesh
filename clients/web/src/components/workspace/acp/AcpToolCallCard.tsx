"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, CheckCircle2, XCircle, Loader2, Circle } from "lucide-react";
import type { AcpToolCall } from "@/stores/acpSession";

function ToolStatusIcon({ toolCall }: { toolCall: AcpToolCall }) {
  if (toolCall.status !== "completed") {
    return <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500 shrink-0" />;
  }
  if (toolCall.success === false) {
    return <XCircle className="h-3.5 w-3.5 text-red-500 shrink-0" />;
  }
  if (toolCall.success === true) {
    return <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />;
  }
  return <Circle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />;
}

export function AcpToolCallCard({ toolCall }: { toolCall: AcpToolCall }) {
  const [expanded, setExpanded] = useState(false);
  const inProgress = toolCall.status !== "completed";

  return (
    <div className={inProgress ? "py-0.5 rounded bg-blue-500/5 animate-pulse" : "py-0.5"}>
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1.5 w-full text-left hover:bg-muted/50 rounded px-1 py-0.5 -mx-1 transition-colors"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
        )}
        <ToolStatusIcon toolCall={toolCall} />
        <span className="text-xs font-mono text-muted-foreground truncate">{toolCall.toolName}</span>
      </button>
      {expanded && (
        <div className="ml-[18px] mt-1 space-y-1">
          <pre className="text-xs bg-muted p-2 rounded overflow-x-auto">
            {toolCall.argumentsJson}
          </pre>
          {toolCall.resultText && (
            <pre className="text-xs bg-green-50 dark:bg-green-950 p-2 rounded overflow-x-auto">
              {toolCall.resultText}
            </pre>
          )}
          {toolCall.errorMessage && (
            <pre className="text-xs bg-red-50 dark:bg-red-950 p-2 rounded overflow-x-auto">
              {toolCall.errorMessage}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
