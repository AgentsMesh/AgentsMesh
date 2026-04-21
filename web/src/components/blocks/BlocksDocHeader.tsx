"use client";

import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface BlocksDocHeaderProps {
  rootTitle: string;
  rootIcon?: string;
  currentTitle: string;
  currentIcon?: string;
  isRoot: boolean;
  onAddBlock: () => void;
}

/**
 * Doc header shown above the DocumentView:
 *   breadcrumb (Pages › rootTitle [› currentTitle]) + title row + +Block CTA.
 * No workspace chip — org is the workspace boundary.
 */
export function BlocksDocHeader({
  rootTitle,
  rootIcon,
  currentTitle,
  currentIcon,
  isRoot,
  onAddBlock,
}: BlocksDocHeaderProps) {
  return (
    <div className="flex flex-col gap-1.5 border-b border-border px-12 pb-3 pt-4">
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
        <span>Pages</span>
        <span className="text-border">›</span>
        <span className={isRoot ? "font-medium text-foreground" : undefined}>
          {rootIcon && <span className="mr-1" aria-hidden="true">{rootIcon}</span>}
          {rootTitle}
        </span>
        {!isRoot && (
          <>
            <span className="text-border">›</span>
            <span className="font-medium text-foreground">{currentTitle}</span>
          </>
        )}
      </div>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="text-[22px] leading-none" aria-hidden="true">
            {currentIcon ?? "📖"}
          </span>
          <h1 className="truncate text-[24px] font-semibold leading-tight text-foreground">
            {currentTitle}
          </h1>
        </div>
        <Button
          type="button"
          onClick={onAddBlock}
          className="h-[30px] gap-1.5 px-3.5"
          data-testid="blocks-doc-add-block"
        >
          <Plus className="h-3.5 w-3.5" />
          <span>Block</span>
        </Button>
      </div>
    </div>
  );
}

export default BlocksDocHeader;
