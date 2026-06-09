"use client";

import type { InlineElement } from "@/lib/viewModels/channelMessage";
import { usePod } from "@/stores/pod";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { cn } from "@/lib/utils";

export function RenderInline({ element }: { element: InlineElement }) {
  switch (element.type) {
    case "text":
      return <TextSpan element={element} />;
    case "mention":
      return <MentionSpan element={element} />;
    case "link": {
      const safe = element.url?.startsWith("http://") || element.url?.startsWith("https://") || element.url?.startsWith("mailto:");
      if (!safe) return <span>{element.text}</span>;
      return (
        <a href={element.url} target="_blank" rel="noopener noreferrer" className="text-primary underline">
          {element.text}
        </a>
      );
    }
    case "linebreak":
      return <br />;
    default:
      return null;
  }
}

function TextSpan({ element }: { element: InlineElement }) {
  const s = element.style;
  const bold = s?.bold ?? element.bold;
  const italic = s?.italic ?? element.italic;
  const strike = s?.strike ?? element.strike;
  const code = s?.code ?? element.code;

  let content: React.ReactNode = element.text;
  if (code) content = <code className="px-1 py-0.5 bg-muted rounded text-sm">{content}</code>;
  if (bold) content = <strong>{content}</strong>;
  if (italic) content = <em>{content}</em>;
  if (strike) content = <del>{content}</del>;
  return <>{content}</>;
}

function MentionSpan({ element }: { element: InlineElement }) {
  const podKey = element.entity_type === "pod" ? element.entity_key : undefined;
  const pod = usePod(podKey);

  let displayName = element.display ?? element.entity_key ?? "unknown";
  if (pod) displayName = getPodDisplayName(pod);

  const s = element.style;
  return (
    <span
      className={cn(
        "text-primary font-medium bg-primary/10 rounded px-0.5",
        s?.bold && "font-bold",
        s?.italic && "italic",
        s?.strike && "line-through",
      )}
    >
      @{displayName}
    </span>
  );
}
