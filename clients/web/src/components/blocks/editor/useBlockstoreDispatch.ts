"use client";

import { useMemo } from "react";

import type { JSONMap } from "@/lib/api/blockstoreTypes";
import {
  addRefOp,
  createBlockOp,
  deleteBlockOp,
  removeRefOp,
  updateBlockOp,
  updateRefOp,
} from "@/lib/blockstore/opBuilder";
import { keyAfter, keyBetween } from "@/lib/blockstore/fractionalIndex";
import { randomUUID } from "@/lib/blockstore/uuid";
import { dispatchOps } from "@/stores/blockstoreDispatch";
import {
  readBlock,
  readRefs,
  readNestChildren,
  useBlockstoreStore,
} from "@/stores/blockstore";

export function useBlockstoreDispatch(workspaceID: string) {
  return useMemo(() => makeDispatcher(workspaceID), [workspaceID]);
}

function makeDispatcher(workspaceID: string) {
  const dispatcher = {
    /**
     * Updates a block's data payload, optionally synchronising the top-level
     * Block.text (search / semantic-embedding source). Entity doc notes that
     * writers maintain `text` — the server never derives it — so paragraph /
     * task renderers must pass `opts.text` whenever the user-visible string
     * changes. Omitting `opts.text` leaves the top-level text untouched.
     */
    async updateBlockData(id: string, patch: JSONMap, opts?: { text?: string | null }) {
      const existing = readBlock(id);
      const nextData = { ...(existing?.data ?? {}), ...patch };
      const op = updateBlockOp({
        id,
        data: nextData,
        ...(opts?.text !== undefined ? { text: opts.text } : {}),
      });
      await dispatchOps(workspaceID, [op]);
    },

    async updateBlockMeta(id: string, patch: JSONMap) {
      const existing = readBlock(id);
      const nextMeta = { ...(existing?.meta ?? {}), ...patch };
      await dispatchOps(workspaceID, [updateBlockOp({ id, meta: nextMeta })]);
    },

    async setBlockVisibility(id: string, visibility: "workspace" | "private") {
      const existing = readBlock(id);
      const currentACL = (existing?.meta?.acl as JSONMap | undefined) ?? {};
      const nextACL: JSONMap = { ...currentACL, visibility };
      const nextMeta = { ...(existing?.meta ?? {}), acl: nextACL };
      await dispatchOps(workspaceID, [updateBlockOp({ id, meta: nextMeta })]);
    },

    async insertChild(parentID: string, type: string, initialData?: JSONMap, opts?: { text?: string | null }) {
      const newID = randomUUID();
      const lastKey = lastChildOrderKey(parentID);
      await dispatchOps(workspaceID, [
        createBlockOp({
          id: newID, type, data: initialData ?? {},
          ...(opts?.text !== undefined ? { text: opts.text } : {}),
        }),
        addRefOp({
          from: parentID,
          to: newID,
          rel: "nest",
          order_key: keyAfter(lastKey),
        }),
      ]);
      return newID;
    },

    async insertBetween(parentID: string, type: string, afterChildID: string | null, beforeChildID: string | null, initialData?: JSONMap, opts?: { text?: string | null }) {
      const afterKey = afterChildID ? childOrderKey(parentID, afterChildID) : null;
      const beforeKey = beforeChildID ? childOrderKey(parentID, beforeChildID) : null;
      const newID = randomUUID();
      await dispatchOps(workspaceID, [
        createBlockOp({
          id: newID, type, data: initialData ?? {},
          ...(opts?.text !== undefined ? { text: opts.text } : {}),
        }),
        addRefOp({ from: parentID, to: newID, rel: "nest", order_key: keyBetween(afterKey, beforeKey) }),
      ]);
      return newID;
    },

    async insertSiblingAfter(siblingID: string, type: string, initialData?: JSONMap, opts?: { text?: string | null }) {
      const parent = nestParentOf(siblingID);
      if (!parent) return null;
      const siblings = nestSiblingIDs(parent.id);
      const idx = siblings.indexOf(siblingID);
      const beforeID = idx >= 0 && idx + 1 < siblings.length ? siblings[idx + 1] : null;
      const newID = await dispatcher.insertBetween(parent.id, type, siblingID, beforeID, initialData, opts);
      if (newID) useBlockstoreStore.getState().actions.requestFocus(newID);
      return newID;
    },

    async moveChild(childID: string, newParentID: string, afterChildID: string | null, beforeChildID: string | null) {
      const refs = readRefs();
      const nestRefID = Object.values(refs).find(
        (r) => r.rel === "nest" && r.to_id === childID,
      )?.id;
      if (!nestRefID) return;
      const afterKey = afterChildID ? childOrderKey(newParentID, afterChildID) : null;
      const beforeKey = beforeChildID ? childOrderKey(newParentID, beforeChildID) : null;
      await dispatchOps(workspaceID, [
        updateRefOp({
          ref_id: nestRefID,
          from: newParentID,
          order_key: keyBetween(afterKey, beforeKey),
        }),
      ]);
    },

    async removeBlock(id: string) {
      await dispatchOps(workspaceID, [deleteBlockOp(id)]);
    },

    async detachChild(childID: string) {
      const refs = readRefs();
      const nestRefID = Object.values(refs).find(
        (r) => r.rel === "nest" && r.to_id === childID,
      )?.id;
      if (!nestRefID) return;
      await dispatchOps(workspaceID, [removeRefOp(nestRefID)]);
    },

    async duplicate(blockID: string) {
      const source = readBlock(blockID);
      if (!source) return null;
      const parent = nestParentOf(blockID);
      if (!parent) return null;
      const siblings = nestSiblingIDs(parent.id);
      const idx = siblings.indexOf(blockID);
      const beforeID = idx + 1 < siblings.length ? siblings[idx + 1] : null;
      return dispatcher.insertBetween(
        parent.id,
        source.type,
        blockID,
        beforeID,
        { ...source.data },
      );
    },

    async createCommentOn(targetID: string, text: string) {
      const newID = randomUUID();
      await dispatchOps(workspaceID, [
        createBlockOp({ id: newID, type: "comment", data: { text }, text }),
        addRefOp({ from: newID, to: targetID, rel: "comments_on" }),
      ]);
      return newID;
    },
  };
  return dispatcher;
}

function lastChildOrderKey(parentID: string): string | null {
  const nestChildren = readNestChildren();
  const refs = readRefs();
  const refIDs = nestChildren[parentID];
  if (!refIDs || refIDs.length === 0) return null;
  const last = refs[refIDs[refIDs.length - 1]];
  return last?.order_key ?? null;
}

function childOrderKey(parentID: string, childID: string): string | null {
  const nestChildren = readNestChildren();
  const refs = readRefs();
  const refIDs = nestChildren[parentID] ?? [];
  for (const rid of refIDs) {
    const r = refs[rid];
    if (r?.to_id === childID) return r.order_key ?? null;
  }
  return null;
}

function nestParentOf(childID: string): { id: string } | null {
  const refs = readRefs();
  for (const r of Object.values(refs)) {
    if (r.rel === "nest" && r.to_id === childID) return { id: r.from_id };
  }
  return null;
}

function nestSiblingIDs(parentID: string): string[] {
  const nestChildren = readNestChildren();
  const refs = readRefs();
  const refIDs: number[] = nestChildren[parentID] ?? [];
  return refIDs.map((rid: number) => refs[rid]?.to_id).filter((id): id is string => Boolean(id));
}
