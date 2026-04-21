"use client";

import React, { useEffect, useMemo } from "react";

import { Button } from "@/components/ui/button";
import { useBlockTypeSpecs } from "@/lib/blockstore/useBlockTypeSpec";
import { useBlockstoreStore, useBlock } from "@/stores/blockstore";

import { BlockRenderer } from "./BlockRenderer";
import { buildAddOptions } from "./editor/documentAddOptions";
import { PendingOpsBadge } from "./editor/PendingOpsBadge";
import { SelectionActionBar } from "./editor/SelectionActionBar";
import { SlashMenu } from "./editor/SlashMenu";
import { useBlockstoreDispatch } from "./editor/useBlockstoreDispatch";

export interface DocumentViewProps {
  workspaceID: string;
  rootBlockID: string;
  /** Controlled slash-menu visibility — the Blocks page lifts this so the
   *  doc header's "+ Block" CTA can toggle the same menu. */
  menuOpen?: boolean;
  onMenuOpenChange?: (open: boolean) => void;
}

// DocumentView is Phase 1's primary rendering surface: a recursive tree of
// blocks rooted at a single page, with an "add block" control at the bottom.
// The catalog of what the add-block button lists lives in documentAddOptions
// so this file stays focused on lifecycle + layout.
export function DocumentView({
  workspaceID,
  rootBlockID,
  menuOpen: menuOpenProp,
  onMenuOpenChange,
}: DocumentViewProps) {
  const root = useBlock(rootBlockID);
  const dispatch = useBlockstoreDispatch(workspaceID);
  const [internalOpen, setInternalOpen] = React.useState(false);
  const isControlled = menuOpenProp !== undefined;
  const menuOpen = isControlled ? (menuOpenProp as boolean) : internalOpen;
  const setMenuOpen = (next: boolean) => {
    if (!isControlled) setInternalOpen(next);
    onMenuOpenChange?.(next);
  };
  const dynamicSpecs = useBlockTypeSpecs(workspaceID);

  useEffect(() => {
    // First-time hydrate: pull the whole nest subtree so the editor has data.
    if (root) return;
    void useBlockstoreStore.getState().actions.loadSubtree(workspaceID, rootBlockID);
  }, [workspaceID, rootBlockID, root]);

  // Memoise the option catalog so we don't rebuild 35+ entries on every
  // keystroke-driven re-render. buildAddOptions is pure in its inputs.
  const addOptions = useMemo(
    () => buildAddOptions({ dispatch, rootBlockID, dynamicSpecs }),
    [dispatch, rootBlockID, dynamicSpecs],
  );

  if (!root) {
    return <div className="p-4 text-sm text-muted-foreground">Loading workspace…</div>;
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-8 pb-16">
      <BlockRenderer blockID={rootBlockID} depth={0} />
      <div className="relative">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setMenuOpen(!menuOpen)}
        >
          + Add block
        </Button>
        <div className="absolute left-0 mt-1">
          <SlashMenu
            open={menuOpen}
            options={addOptions}
            onClose={() => setMenuOpen(false)}
          />
        </div>
      </div>
      <SelectionActionBar workspaceID={workspaceID} />
      <PendingOpsBadge />
    </div>
  );
}
