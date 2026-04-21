"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useAcpSession } from "@/stores/acpSession";
import { ChevronDown, ChevronRight, ArrowDown, ArrowUp, Bug } from "lucide-react";

interface AcpDebugPanelProps {
  podKey: string;
}

interface DebugEntry {
  timestamp: number;
  direction: "in" | "out";
  type: string;
  detail: string;
  raw?: string;
}

/**
 * ACP debug panel for monitoring protocol messages.
 * Shows tool calls, state changes, and permission requests.
 */
export function AcpDebugPanel({ podKey }: AcpDebugPanelProps) {
  const [open, setOpen] = useState(false);
  const session = useAcpSession(podKey);
  const bottomRef = useRef<HTMLDivElement>(null);

  const entries = useMemo<DebugEntry[]>(() => {
    if (!session) return [];
    const items: DebugEntry[] = [];

    // Tool calls
    for (const tc of Object.values(session.toolCalls)) {
      items.push({
        timestamp: tc.timestamp,
        direction: "in",
        type: "toolCall",
        detail: `${tc.toolName} [${tc.status}]${tc.success === false ? " ERROR" : ""}`,
        raw: tc.argumentsJson,
      });
    }

    // Logs
    for (const log of session.logs) {
      items.push({
        timestamp: log.timestamp,
        direction: "in",
        type: `log:${log.level}`,
        detail: log.message,
      });
    }

    // Pending permissions (use latest tool call timestamp as approximation)
    const latestTs = items.length > 0 ? items[items.length - 1].timestamp : 0;
    for (const perm of session.pendingPermissions) {
      items.push({
        timestamp: latestTs + 1,
        direction: "in",
        type: "permissionRequest",
        detail: `${perm.toolName}: ${perm.description}`,
        raw: perm.argumentsJson,
      });
    }

    items.sort((a, b) => a.timestamp - b.timestamp);
    return items.slice(-100); // keep last 100
  }, [session]);

  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [entries.length, open]);

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="absolute bottom-12 right-2 z-10 rounded-full bg-muted p-1.5 opacity-50 hover:opacity-100 transition-opacity"
        title="ACP Debug"
      >
        <Bug className="h-3.5 w-3.5 text-muted-foreground" />
      </button>
    );
  }

  return (
    <div className="border-t bg-muted/50 max-h-[200px] overflow-y-auto text-[10px] font-mono">
      <div className="sticky top-0 flex items-center justify-between bg-muted px-2 py-0.5 border-b">
        <span className="text-muted-foreground font-medium">ACP Debug</span>
        <button onClick={() => setOpen(false)} className="text-muted-foreground hover:text-foreground">
          close
        </button>
      </div>
      {entries.map((e, i) => (
        <DebugRow key={`${e.timestamp}-${i}`} entry={e} />
      ))}
      <div ref={bottomRef} />
    </div>
  );
}

function DebugRow({ entry }: { entry: DebugEntry }) {
  const [expanded, setExpanded] = useState(false);
  const Icon = entry.direction === "in" ? ArrowDown : ArrowUp;
  const color = entry.direction === "in" ? "text-blue-500" : "text-green-500";

  return (
    <div className="border-b border-border/50">
      <button
        onClick={() => entry.raw && setExpanded(!expanded)}
        className="flex items-center gap-1 w-full text-left px-2 py-0.5 hover:bg-muted/80"
      >
        {entry.raw ? (
          expanded ? <ChevronDown className="h-2.5 w-2.5 shrink-0" /> : <ChevronRight className="h-2.5 w-2.5 shrink-0" />
        ) : (
          <span className="w-2.5" />
        )}
        <Icon className={`h-2.5 w-2.5 shrink-0 ${color}`} />
        <span className="text-muted-foreground">{entry.type}</span>
        <span className="truncate">{entry.detail}</span>
      </button>
      {expanded && entry.raw && (
        <pre className="px-6 py-1 text-[9px] text-muted-foreground overflow-x-auto whitespace-pre-wrap">
          {formatJSON(entry.raw)}
        </pre>
      )}
    </div>
  );
}

function formatJSON(s: string): string {
  try {
    return JSON.stringify(JSON.parse(s), null, 2);
  } catch {
    return s;
  }
}
