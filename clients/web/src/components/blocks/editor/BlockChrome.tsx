"use client";

import React from "react";
import * as ContextMenu from "@radix-ui/react-context-menu";
import { Copy, GripVertical, Lock, LockOpen, MessageSquare, Trash } from "lucide-react";

import { cn } from "@/lib/utils";
import type { JSONMap } from "@/lib/api/blockstoreTypes";
import { useBlockstoreStore, useBlock } from "@/stores/blockstore";

import { useDragHandle } from "./SortableNest";
import { BlockCommentsPopover } from "./BlockCommentsPopover";

export interface BlockChromeProps {
  children: React.ReactNode;
  onDelete: () => void;
  onDuplicate: () => void;
  onToggleVisibility?: (next: "workspace" | "private") => void;
  className?: string;
  blockID?: string;
}

export function BlockChrome({
  children,
  onDelete,
  onDuplicate,
  onToggleVisibility,
  className,
  blockID,
}: BlockChromeProps) {
  const dragListeners = useDragHandle();
  const block = useBlock(blockID);
  const selected = useBlockstoreStore((s) =>
    blockID ? s.selectedBlockIDs.includes(blockID) : false,
  );
  const toggleSelection = useBlockstoreStore((s) => s.actions.toggleSelection);
  const activeCommentBlockID = useBlockstoreStore((s) => s.activeCommentBlockID);
  const setActiveComment = useBlockstoreStore((s) => s.actions.setActiveCommentBlockID);

  const acl = (block?.meta?.acl as JSONMap | undefined) ?? {};
  const isPrivate = acl.visibility === "private";
  const isCommentActive = !!blockID && activeCommentBlockID === blockID;

  const handleClickCapture = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!blockID) return;
    if (!(e.shiftKey || e.metaKey || e.ctrlKey)) return;
    e.preventDefault();
    e.stopPropagation();
    toggleSelection(blockID);
  };

  const handleToggleComment = () => {
    if (!blockID) return;
    setActiveComment(isCommentActive ? null : blockID);
  };

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>
        <div
          id={blockID ? `block-${blockID}` : undefined}
          className={cn(
            "group relative rounded",
            selected && "bg-primary/10 ring-2 ring-primary/60",
            isCommentActive && "ring-2 ring-primary/40",
            isPrivate && "bg-amber-50/60",
            className,
          )}
          onClickCapture={handleClickCapture}
        >
          <button
            type="button"
            aria-label="Drag to reorder"
            className="pointer-events-none absolute -left-5 top-0.5 cursor-grab rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-accent hover:text-foreground active:cursor-grabbing"
            {...(dragListeners ?? {})}
          >
            <GripVertical className="h-3.5 w-3.5" />
          </button>
          {children}
          {isCommentActive && blockID && block?.workspace_id && (
            <BlockCommentsPopover
              blockID={blockID}
              workspaceID={block.workspace_id}
              onClose={() => setActiveComment(null)}
            />
          )}
          <div className="pointer-events-none absolute -right-1 top-0 flex items-center gap-0.5 opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100">
            {isPrivate && (
              <span
                title="Private — only you and explicitly-allowed users can see this block"
                className="rounded bg-amber-100 p-1 text-amber-700"
              >
                <Lock className="h-3 w-3" />
              </span>
            )}
            <ToolbarButton
              onClick={handleToggleComment}
              aria-label="Comment"
              aria-pressed={isCommentActive}
            >
              <MessageSquare className="h-3.5 w-3.5" />
            </ToolbarButton>
            <ToolbarButton onClick={onDuplicate} aria-label="Duplicate block">
              <Copy className="h-3.5 w-3.5" />
            </ToolbarButton>
            <ToolbarButton onClick={onDelete} aria-label="Delete block">
              <Trash className="h-3.5 w-3.5" />
            </ToolbarButton>
          </div>
        </div>
      </ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content className="z-50 min-w-[160px] rounded-md border border-border bg-popover p-1 shadow-md">
          <ContextMenuItem onSelect={handleToggleComment}>
            <MessageSquare className="mr-2 h-3.5 w-3.5" /> Comment
          </ContextMenuItem>
          <ContextMenuItem onSelect={onDuplicate}>Duplicate</ContextMenuItem>
          {onToggleVisibility && (
            isPrivate ? (
              <ContextMenuItem onSelect={() => onToggleVisibility("workspace")}>
                <LockOpen className="mr-2 h-3.5 w-3.5" /> Make workspace-visible
              </ContextMenuItem>
            ) : (
              <ContextMenuItem onSelect={() => onToggleVisibility("private")}>
                <Lock className="mr-2 h-3.5 w-3.5" /> Make private
              </ContextMenuItem>
            )
          )}
          <ContextMenuItem onSelect={onDelete} destructive>
            Delete
          </ContextMenuItem>
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  );
}

function ToolbarButton({
  children,
  onClick,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onClick?.(e);
      }}
      className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
      {...rest}
    >
      {children}
    </button>
  );
}

function ContextMenuItem({
  children,
  onSelect,
  destructive,
}: {
  children: React.ReactNode;
  onSelect: () => void;
  destructive?: boolean;
}) {
  return (
    <ContextMenu.Item
      onSelect={onSelect}
      className={cn(
        "flex cursor-default items-center rounded px-2 py-1.5 text-sm outline-none",
        destructive
          ? "text-destructive focus:bg-destructive/10"
          : "focus:bg-accent focus:text-foreground",
      )}
    >
      {children}
    </ContextMenu.Item>
  );
}
