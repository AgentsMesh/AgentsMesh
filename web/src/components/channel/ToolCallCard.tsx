"use client";

import { Wrench } from "lucide-react";
import type { MessageContent } from "@/lib/api/channel-message-types";

interface ToolCallCardProps {
  content: MessageContent;
}

/**
 * Inline tool-call card rendered inside a Pod message body. Design spec:
 * gray pill frame, 🔧 icon + monospace tool name, secondary line for the
 * target (e.g. `src/auth.ts (142 lines)`). Pulled from the message content
 * blocks — the first block's text becomes the tool name, a second block (or
 * `attachment_key`) becomes the target.
 */
export function ToolCallCard({ content }: ToolCallCardProps) {
  const toolName = extractText(content, 0) || "tool";
  const target = extractText(content, 1) || content.attachment_key || "";

  return (
    <div className="w-full max-w-[520px] rounded-md border border-border bg-muted/50 px-3 py-2">
      <div className="flex items-center gap-1.5">
        <Wrench className="h-3 w-3 text-primary" />
        <span className="font-mono text-[12px] font-medium text-primary">{toolName}</span>
      </div>
      {target && (
        <p className="mt-1 font-mono text-[11px] text-muted-foreground">{target}</p>
      )}
    </div>
  );
}

function extractText(content: MessageContent, blockIdx: number): string {
  const block = content.blocks?.[blockIdx];
  if (!block?.elements) return "";
  return block.elements
    .map((el) => {
      if (el.type === "text" || el.type === "link") return el.text ?? "";
      if (el.type === "mention") return el.display ?? "";
      return "";
    })
    .join("")
    .trim();
}

export default ToolCallCard;
