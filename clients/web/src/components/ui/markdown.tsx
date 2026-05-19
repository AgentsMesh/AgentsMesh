"use client";

import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

interface MarkdownProps {
  content: string;
  className?: string;
  compact?: boolean;
  highlightMentions?: boolean;
}

const remarkPlugins = [remarkGfm];

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

export function Markdown({
  content,
  className,
  compact = false,
  highlightMentions = false,
}: MarkdownProps) {
  return (
    <div
      className={cn(
        "prose max-w-none",
        compact && "prose-sm",
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
