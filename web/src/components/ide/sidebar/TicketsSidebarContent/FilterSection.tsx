"use client";

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown, ChevronRight } from "lucide-react";

interface FilterSectionProps {
  title: string;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  selectedCount: number;
  showBorder?: boolean;
  children: React.ReactNode;
}

/**
 * Generic collapsible filter section used by status, priority, and repository filters.
 */
export function FilterSection({
  title,
  expanded,
  onExpandedChange,
  selectedCount,
  showBorder = false,
  children,
}: FilterSectionProps) {
  return (
    <Collapsible open={expanded} onOpenChange={onExpandedChange}>
      <CollapsibleTrigger asChild>
        <div className={`flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/50 ${showBorder ? 'border-t border-border' : ''}`}>
          <span className="text-xs font-medium">{title}</span>
          <div className="flex items-center gap-1">
            {selectedCount > 0 && (
              <span className="text-xs bg-primary/10 text-primary px-1.5 rounded">
                {selectedCount}
              </span>
            )}
            {expanded ? (
              <ChevronDown className="w-3.5 h-3.5 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
            )}
          </div>
        </div>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="px-3 pb-2 space-y-1">
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
