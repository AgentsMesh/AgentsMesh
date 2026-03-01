"use client";

import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

interface MarkdownProps {
  content: string;
  className?: string;
  /** Compact mode with smaller text for embedded use */
  compact?: boolean;
  /** Enable @mention highlighting in text nodes */
  highlightMentions?: boolean;
}

const remarkPlugins = [remarkGfm];

/**
 * Render text with @mention highlighting.
 * Splits text on @word patterns and wraps matches in styled spans.
 */
function TextWithMentions({ children }: { children: string }) {
  const mentionRegex = /(@[\w.\-]+)/g;
  const parts = children.split(mentionRegex);

  return (
    <>
      {parts.map((part, i) => {
        if (mentionRegex.test(part)) {
          mentionRegex.lastIndex = 0;
          return (
            <span
              key={i}
              className="text-primary font-medium bg-primary/10 rounded px-0.5"
            >
              {part}
            </span>
          );
        }
        mentionRegex.lastIndex = 0;
        return part;
      })}
    </>
  );
}

/**
 * Custom components for react-markdown that highlight @mentions in text nodes.
 */
const mentionComponents: Components = {
  p({ children }) {
    return <p>{processMentions(children)}</p>;
  },
  li({ children }) {
    return <li>{processMentions(children)}</li>;
  },
  td({ children }) {
    return <td>{processMentions(children)}</td>;
  },
  th({ children }) {
    return <th>{processMentions(children)}</th>;
  },
};

/**
 * Process children to replace plain string nodes with mention-highlighted versions.
 */
function processMentions(children: React.ReactNode): React.ReactNode {
  if (!children) return children;
  if (typeof children === "string") {
    return <TextWithMentions>{children}</TextWithMentions>;
  }
  if (Array.isArray(children)) {
    return children.map((child, i) => {
      if (typeof child === "string") {
        return <TextWithMentions key={i}>{child}</TextWithMentions>;
      }
      return child;
    });
  }
  return children;
}

/**
 * Markdown renderer component using react-markdown with GFM support
 */
export function Markdown({
  content,
  className,
  compact = false,
  highlightMentions = false,
}: MarkdownProps) {
  return (
    <div
      className={cn(
        "prose prose-neutral dark:prose-invert max-w-none",
        compact && "prose-sm",
        // Override prose defaults for compact mode
        compact && "[&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0.5 [&_h1]:text-base [&_h2]:text-sm [&_h3]:text-xs",
        className
      )}
    >
      <ReactMarkdown
        remarkPlugins={remarkPlugins}
        components={highlightMentions ? mentionComponents : undefined}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

export default Markdown;
