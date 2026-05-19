"use client";

import React from "react";
import { AtSign } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";

import { BlockChrome } from "../editor/BlockChrome";
import { UserPicker } from "../editor/UserPicker";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

// MentionRenderer shows "@user" inline with a live UserPicker popover backed
// by /users/search. data.user_id holds the numeric reference, data.display
// caches the display label so the pill renders immediately without waiting
// for a user-directory fetch.
export function MentionRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const userID = (block.data?.user_id as number | undefined) ?? 0;
  const display = (block.data?.display as string | undefined) ?? "";

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className="inline-flex"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      {userID && display ? (
        <span className="inline-flex items-center gap-0.5 rounded bg-primary/10 px-1.5 py-0.5 text-sm text-primary">
          <AtSign className="h-3 w-3" />
          {display}
          <ChangeButton
            onChange={(next) => {
              if (!next) {
                dispatch.updateBlockData(block.id, { user_id: 0, display: "" }, { text: "" });
              } else {
                const label = next.name || next.username;
                dispatch.updateBlockData(
                  block.id,
                  { user_id: next.id, display: label },
                  { text: `@${label}` },
                );
              }
            }}
            value={userID}
          />
        </span>
      ) : (
        <UserPicker
          value={userID}
          placeholder="Pick a user…"
          onPick={(next) => {
            if (!next) return;
            const label = next.name || next.username;
            dispatch.updateBlockData(
              block.id,
              { user_id: next.id, display: label },
              { text: `@${label}` },
            );
          }}
        />
      )}
    </BlockChrome>
  );
}

function ChangeButton({
  value,
  onChange,
}: {
  value: number;
  onChange: (next: import("@/lib/api/user").UserSummary | null) => void;
}) {
  return (
    <span className="ml-1">
      <UserPicker value={value} onPick={onChange} placeholder="change" />
    </span>
  );
}
