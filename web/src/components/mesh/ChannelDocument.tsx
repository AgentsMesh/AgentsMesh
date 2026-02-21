"use client";

import { useState } from "react";
import { Markdown } from "@/components/ui/markdown";
import { FileText, ChevronDown, ChevronRight } from "lucide-react";

interface ChannelDocumentProps {
  document: string;
}

/**
 * Collapsible document section that renders channel document as markdown.
 * Displayed between the channel header and message list.
 */
export function ChannelDocument({ document }: ChannelDocumentProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex-shrink-0 border-b border-border bg-muted/30">
      <button
        type="button"
        className="w-full flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
        onClick={() => setExpanded((prev) => !prev)}
      >
        <FileText className="w-3.5 h-3.5 flex-shrink-0" />
        <span className="font-medium">Document</span>
        {expanded ? (
          <ChevronDown className="w-3.5 h-3.5 ml-auto flex-shrink-0" />
        ) : (
          <ChevronRight className="w-3.5 h-3.5 ml-auto flex-shrink-0" />
        )}
      </button>
      {expanded && (
        <div className="px-4 pb-3 max-h-[40vh] overflow-y-auto">
          <Markdown content={document} compact />
        </div>
      )}
    </div>
  );
}

export default ChannelDocument;
