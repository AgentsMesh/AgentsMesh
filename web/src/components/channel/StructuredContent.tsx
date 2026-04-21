"use client";

import type { MessageContent, Block, InlineElement } from "@/lib/api/channel-message-types";
import { usePods } from "@/stores/pod";
import { getPodDisplayName } from "@/lib/pod-display-name";
import { cn } from "@/lib/utils";

interface StructuredContentProps {
  content: MessageContent;
  className?: string;
}

const SUPPORTED_SCHEMA_VERSION = 1;

export function StructuredContent({ content, className }: StructuredContentProps) {
  if (!content.blocks?.length) return null;

  if (content.schema_version && content.schema_version > SUPPORTED_SCHEMA_VERSION) {
    return (
      <p className="text-sm text-muted-foreground italic">
        This message uses a newer format. Please update your client.
      </p>
    );
  }

  return (
    <div className={cn("prose prose-sm max-w-none", className)}>
      {content.blocks.map((block, i) => (
        <RenderBlock key={i} block={block} />
      ))}
    </div>
  );
}

function RenderBlock({ block }: { block: Block }) {
  switch (block.type) {
    case "paragraph":
      return (
        <>
          <p>
            {block.elements?.map((el, i) => (
              <RenderInline key={i} element={el} />
            ))}
          </p>
          <BlockChildren block={block} />
        </>
      );
    case "code_block":
      return (
        <pre className="p-3 bg-muted rounded-md text-sm overflow-x-auto">
          <code>{block.text}</code>
        </pre>
      );
    case "heading": {
      const Tag = `h${Math.min(block.level ?? 1, 3)}` as "h1" | "h2" | "h3";
      return (
        <>
          <Tag>
            {block.elements?.map((el, i) => (
              <RenderInline key={i} element={el} />
            ))}
          </Tag>
          <BlockChildren block={block} />
        </>
      );
    }
    case "quote":
      return (
        <blockquote>
          {block.elements?.map((el, i) => (
            <RenderInline key={i} element={el} />
          ))}
          <BlockChildren block={block} />
        </blockquote>
      );
    case "list": {
      const Tag = block.ordered ? "ol" : "ul";
      return (
        <>
          <Tag>
            {block.items?.map((item, i) => (
              <li key={i}>
                {item.map((el, j) => (
                  <RenderInline key={j} element={el} />
                ))}
              </li>
            ))}
          </Tag>
          <BlockChildren block={block} />
        </>
      );
    }
    default:
      return null;
  }
}

function BlockChildren({ block }: { block: Block }) {
  if (!block.children?.length) return null;
  return <>{block.children.map((child, i) => <RenderBlock key={i} block={child} />)}</>;
}

function RenderInline({ element }: { element: InlineElement }) {
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
  // Support both new Style object and old flat booleans (backward compat)
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
  const allPods = usePods();

  let displayName = element.display ?? element.entity_key ?? "unknown";

  if (element.entity_type === "pod" && element.entity_key) {
    const pod = allPods.find((p) => p.pod_key === element.entity_key);
    if (pod) displayName = getPodDisplayName(pod);
  }

  return (
    <span className="text-primary font-medium bg-primary/10 rounded px-0.5">
      @{displayName}
    </span>
  );
}

export default StructuredContent;
