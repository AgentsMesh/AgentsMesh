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
  onNavigateRoot?: () => void;
}

export function BlocksDocHeader({
  rootTitle,
  rootIcon,
  currentTitle,
  currentIcon,
  isRoot,
  onAddBlock,
  onNavigateRoot,
}: BlocksDocHeaderProps) {
  const rootLabel = (
    <>
      {rootIcon && <span className="mr-1" aria-hidden="true">{rootIcon}</span>}
      {rootTitle}
    </>
  );
  return (
    <div className="flex flex-col gap-1.5 border-b border-border px-12 pb-3 pt-4">
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
        <span>Pages</span>
        <span className="text-border">›</span>
        {!isRoot && onNavigateRoot ? (
          <button
            type="button"
            onClick={onNavigateRoot}
            data-testid="blocks-breadcrumb-root"
            className="inline-flex items-center rounded text-muted-foreground transition-colors hover:text-foreground hover:underline"
          >
            {rootLabel}
          </button>
        ) : (
          <span className={isRoot ? "font-medium text-foreground" : "text-muted-foreground"}>
            {rootLabel}
          </span>
        )}
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
