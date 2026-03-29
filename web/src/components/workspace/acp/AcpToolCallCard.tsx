"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, CheckCircle2, XCircle, Loader2, Circle } from "lucide-react";
import type { AcpToolCall } from "@/stores/acpSession";

/**
 * Tool call status has 3 phases in the ACP lifecycle:
 *
 * 1. running (status != "completed")       → spinner (blue)
 * 2. completed, awaiting result (success === undefined) → circle (muted)
 * 3. completed with result:
 *    - success === true  → green check
 *    - success === false → red X
 */
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
  // completed but no result yet (args collected, tool executing)
  return <Circle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />;
}

interface EditArgs {
  file_path?: string;
  old_string?: string;
  new_string?: string;
}

interface MultiEditArgs {
  file_path?: string;
  edits?: EditArgs[];
}

/** Renders a single old→new edit pair with terminal-style background for new content. */
function EditPair({ old_string, new_string }: { old_string?: string; new_string?: string }) {
  return (
    <div className="space-y-1">
      {old_string !== undefined && (
        <pre className="text-xs bg-red-950/40 text-red-300 p-2 rounded overflow-x-auto whitespace-pre-wrap break-all">
          {old_string}
        </pre>
      )}
      {new_string !== undefined && (
        <pre className="text-xs bg-terminal-bg text-terminal-text p-2 rounded overflow-x-auto whitespace-pre-wrap break-all">
          {new_string}
        </pre>
      )}
    </div>
  );
}

/** Structured view for Edit / MultiEdit tool calls. Falls back to raw JSON on parse error. */
function EditArguments({ toolName, argumentsJson }: { toolName: string; argumentsJson: string }) {
  let parsed: EditArgs | MultiEditArgs | null = null;
  try {
    parsed = JSON.parse(argumentsJson);
  } catch {
    // fall through to raw view
  }

  if (!parsed) {
    return (
      <pre className="text-xs bg-muted p-2 rounded overflow-x-auto">{argumentsJson}</pre>
    );
  }

  if (toolName === "Edit") {
    const args = parsed as EditArgs;
    return (
      <div className="space-y-1">
        {args.file_path && (
          <p className="text-[10px] font-mono text-muted-foreground truncate">{args.file_path}</p>
        )}
        <EditPair old_string={args.old_string} new_string={args.new_string} />
      </div>
    );
  }

  // MultiEdit
  const args = parsed as MultiEditArgs;
  return (
    <div className="space-y-1">
      {args.file_path && (
        <p className="text-[10px] font-mono text-muted-foreground truncate">{args.file_path}</p>
      )}
      {args.edits?.map((edit, i) => (
        <EditPair key={i} old_string={edit.old_string} new_string={edit.new_string} />
      ))}
    </div>
  );
}

const EDIT_TOOLS = new Set(["Edit", "MultiEdit"]);

export function AcpToolCallCard({ toolCall }: { toolCall: AcpToolCall }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="py-0.5">
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
        <span className="text-xs font-mono text-muted-foreground truncate">{toolCall.tool_name}</span>
      </button>
      {expanded && (
        <div className="ml-[18px] mt-1 space-y-1">
          {EDIT_TOOLS.has(toolCall.tool_name) ? (
            <EditArguments toolName={toolCall.tool_name} argumentsJson={toolCall.arguments_json} />
          ) : (
            <pre className="text-xs bg-muted p-2 rounded overflow-x-auto">
              {toolCall.arguments_json}
            </pre>
          )}
          {toolCall.result_text && (
            <pre className="text-xs bg-green-50 dark:bg-green-950 p-2 rounded overflow-x-auto">
              {toolCall.result_text}
            </pre>
          )}
          {toolCall.error_message && (
            <pre className="text-xs bg-red-50 dark:bg-red-950 p-2 rounded overflow-x-auto">
              {toolCall.error_message}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
