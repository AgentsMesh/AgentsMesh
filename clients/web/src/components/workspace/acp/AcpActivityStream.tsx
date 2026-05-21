"use client";

import { useEffect, useMemo, useRef } from "react";
import { useTranslations } from "next-intl";
import { useAcpSession } from "@/stores/acpSession";
import type { AcpToolCall, AcpThinking, AcpLog } from "@/stores/acpSession";
import { AcpToolCallCard } from "./AcpToolCallCard";
import { Markdown } from "@/components/ui/markdown";
import { AlertTriangle, XCircle } from "lucide-react";

interface AcpActivityStreamProps {
  podKey: string;
}

type TimelineItem =
  | { kind: "message"; key: string; timestamp: number; role: string; text: string; complete: boolean }
  | { kind: "tool"; key: string; timestamp: number; data: AcpToolCall }
  | { kind: "thinking"; key: string; timestamp: number; data: AcpThinking }
  | { kind: "log"; key: string; timestamp: number; data: AcpLog };

export function AcpActivityStream({ podKey }: AcpActivityStreamProps) {
  const t = useTranslations("acp.activityStream");
  const session = useAcpSession(podKey);
  const bottomRef = useRef<HTMLDivElement>(null);

  const timeline = useMemo<TimelineItem[]>(() => {
    if (!session) return [];
    const items: TimelineItem[] = [];

    for (let i = 0; i < session.messages.length; i++) {
      const msg = session.messages[i];
      items.push({
        kind: "message",
        key: `msg-${msg.role}-${msg.timestamp}-${i}`,
        timestamp: msg.timestamp,
        role: msg.role,
        text: msg.text,
        complete: msg.complete ?? true,
      });
    }

    for (const tc of Object.values(session.toolCalls)) {
      items.push({ kind: "tool", key: `tc-${tc.toolCallId}`, timestamp: tc.timestamp, data: tc });
    }

    for (let i = 0; i < session.thinkings.length; i++) {
      const th = session.thinkings[i];
      items.push({ kind: "thinking", key: `th-${th.timestamp}-${i}`, timestamp: th.timestamp, data: th });
    }

    for (let i = 0; i < session.logs.length; i++) {
      const log = session.logs[i];
      items.push({ kind: "log", key: `log-${log.timestamp}-${i}`, timestamp: log.timestamp, data: log });
    }

    items.sort((a, b) => a.timestamp - b.timestamp);
    return items;
  }, [session]);

  const messageCount = session?.messages.length ?? 0;
  const toolCallCount = session ? Object.keys(session.toolCalls).length : 0;
  const thinkingCount = session?.thinkings.length ?? 0;
  const logCount = session?.logs.length ?? 0;

  const hasActiveSpinner = useMemo(() => {
    if (!session) return false;
    const lastThinking = session.thinkings[session.thinkings.length - 1];
    if (lastThinking && !lastThinking.complete) return true;
    for (const tc of Object.values(session.toolCalls)) {
      if (tc.status !== "completed") return true;
    }
    return false;
  }, [session]);

  const showWorkingPlaceholder =
    session?.state === "processing" && !hasActiveSpinner;

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messageCount, toolCallCount, thinkingCount, logCount, showWorkingPlaceholder]);

  if (!session) {
    return (
      <div className="text-muted-foreground text-center py-8">
        {t("waitingSession")}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {timeline.map((item) => {
        switch (item.kind) {
          case "message":
            return item.role === "user" ? (
              <UserInstruction key={item.key} text={item.text} />
            ) : (
              <AssistantOutput key={item.key} text={item.text} complete={item.complete} />
            );
          case "tool":
            return <AcpToolCallCard key={item.key} toolCall={item.data} />;
          case "thinking":
            return <ThinkingIndicator key={item.key} thinking={item.data} />;
          case "log":
            return <LogEntry key={item.key} log={item.data} />;
        }
      })}
      {showWorkingPlaceholder && <WorkingPlaceholder label={t("agentWorking")} />}
      <div ref={bottomRef} />
    </div>
  );
}

function WorkingPlaceholder({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 py-1 text-muted-foreground text-sm italic">
      <span className="inline-block h-3 w-3 border-2 border-muted-foreground/40 border-t-muted-foreground rounded-full animate-spin" />
      <span>{label}</span>
    </div>
  );
}

function UserInstruction({ text }: { text: string }) {
  const isSlashCommand = text.startsWith("/");
  return (
    <div className="flex items-start gap-2 py-1">
      <span className="text-muted-foreground select-none shrink-0 font-mono text-sm">&gt;</span>
      <span
        className={
          isSlashCommand
            ? "text-blue-500 dark:text-blue-400 text-sm font-mono"
            : "text-muted-foreground text-sm whitespace-pre-wrap"
        }
      >
        {text}
      </span>
    </div>
  );
}

function AssistantOutput({ text, complete }: { text: string; complete: boolean }) {
  return (
    <div className="py-1">
      <Markdown content={text} compact />
      {!complete && <StreamingCaret />}
    </div>
  );
}

function StreamingCaret() {
  return (
    <span
      aria-hidden
      className="inline-block w-[7px] h-[14px] ml-0.5 align-text-bottom bg-foreground/70 animate-pulse"
    />
  );
}

function ThinkingIndicator({ thinking }: { thinking: AcpThinking }) {
  const t = useTranslations("acp.activityStream");
  return (
    <details className="py-1 group">
      <summary className="text-muted-foreground text-sm italic cursor-pointer select-none list-none flex items-center gap-1.5">
        {thinking.complete ? (
          <span className="inline-block h-3 w-3 text-muted-foreground/40 text-center leading-3 text-[10px]">&#x25CB;</span>
        ) : (
          <span className="inline-block h-3 w-3 border-2 border-muted-foreground/40 border-t-muted-foreground rounded-full animate-spin" />
        )}
        <span>{t("thinking")}</span>
      </summary>
      <div className="mt-1 ml-[18px] text-muted-foreground text-xs whitespace-pre-wrap">
        {thinking.text}
      </div>
    </details>
  );
}

function LogEntry({ log }: { log: AcpLog }) {
  const isError = log.level === "error";
  return (
    <div
      className={`flex items-start gap-2 py-1 px-2 rounded text-xs ${
        isError
          ? "bg-red-500/10 text-red-600 dark:text-red-400"
          : "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400"
      }`}
    >
      {isError ? (
        <XCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
      ) : (
        <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
      )}
      <span className="whitespace-pre-wrap">{log.message}</span>
    </div>
  );
}
