import { create } from "zustand";
import { useMemo } from "react";
import { getBlockstoreService } from "@/lib/wasm-core";
import { blockstoreApi } from "@/lib/api/blockstoreApi";
import type { Block, BlockRef, Workspace } from "@/lib/api/blockstoreTypes";
import type {
  BlockstoreState,
  BlocksMap,
  RefsMap,
  ChildrenIndex,
  BacklinksIndex,
  WorkspacesMap,
  LastOpIdMap,
} from "./blockstoreTypes";

// Zustand store for Block Store — Rust SSOT edition.
//
// All block/ref/workspace data lives in the Rust BlockstoreService.
// This store only tracks:
//   • `_tick`: invalidation signal bumped on Rust-side mutations. Selector
//     hooks re-derive from `svc().*_json()` whenever tick advances.
//   • UI state: selection, focus, active workspace, comments rail target.
//
// Writes go through Rust (apply_ops / load_subtree / apply_remote_op) and
// the store reacts by bumping tick. There is no JS-side mirror — zero
// drift risk.

const svc = () => getBlockstoreService();
const bump = () => useBlockstoreStore.setState((s) => ({ _tick: s._tick + 1 }));

function safeParse<T>(raw: string | null | undefined, fallback: T): T {
  if (!raw) return fallback;
  try { return JSON.parse(raw) as T; } catch { return fallback; }
}

// ── Non-hook readers (for outside-render / tests) ──

export function readWorkspaces(): WorkspacesMap {
  return safeParse<WorkspacesMap>(svc().workspaces_json(), {});
}

export function readBlock(id: string): Block | null {
  const raw = svc().get_block_json(id);
  if (!raw) return null;
  return typeof raw === "string" ? JSON.parse(raw) : (raw as Block);
}

export function readBlocks(): BlocksMap {
  return safeParse<BlocksMap>(svc().blocks_json(), {});
}

export function readRefs(): RefsMap {
  return safeParse<RefsMap>(svc().refs_json(), {});
}

export function readNestChildren(): ChildrenIndex {
  return safeParse<ChildrenIndex>(svc().nest_children_json(), {});
}

export function readBacklinks(): BacklinksIndex {
  return safeParse<BacklinksIndex>(svc().backlinks_json(), {});
}

export function readLastOpIds(): LastOpIdMap {
  return safeParse<LastOpIdMap>(svc().last_op_ids_json(), {});
}

// ── Selector hooks (tick-driven) ──

export function useWorkspaces(): WorkspacesMap {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readWorkspaces(), [tick]);
}

export function useWorkspace(id: string | null | undefined): Workspace | undefined {
  const tick = useBlockstoreStore((s) => s._tick);
  return useMemo(() => {
    if (!id) return undefined;
    return readWorkspaces()[id];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, id]);
}

export function useBlocks(): BlocksMap {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readBlocks(), [tick]);
}

export function useBlock(id: string | null | undefined): Block | null {
  const tick = useBlockstoreStore((s) => s._tick);
  return useMemo(() => {
    if (!id) return null;
    return readBlock(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, id]);
}

export function useRefs(): RefsMap {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readRefs(), [tick]);
}

export function useNestChildrenIndex(): ChildrenIndex {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readNestChildren(), [tick]);
}

export function useNestChildren(parentId: string | null | undefined): number[] {
  const tick = useBlockstoreStore((s) => s._tick);
  return useMemo(() => {
    if (!parentId) return [];
    return readNestChildren()[parentId] ?? [];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, parentId]);
}

export function useBacklinksIndex(): BacklinksIndex {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readBacklinks(), [tick]);
}

export function useBacklinks(targetId: string | null | undefined): number[] {
  const tick = useBlockstoreStore((s) => s._tick);
  return useMemo(() => {
    if (!targetId) return [];
    return readBacklinks()[targetId] ?? [];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, targetId]);
}

export function useLastOpIds(): LastOpIdMap {
  const tick = useBlockstoreStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readLastOpIds(), [tick]);
}

// ── Store ──

export const useBlockstoreStore = create<BlockstoreState>((set, get) => ({
  _tick: 0,
  pendingFocusBlockID: null,
  selectedBlockIDs: [],
  activeWorkspaceId: null,
  activeCommentBlockID: null,
  loading: false,
  error: null,

  actions: {
    setActiveWorkspaceId(id) {
      set({ activeWorkspaceId: id });
    },

    setActiveCommentBlockID(id) {
      set({ activeCommentBlockID: id });
    },

    async loadWorkspaces() {
      set({ loading: true, error: null });
      try {
        await blockstoreApi.listWorkspaces();
        set({ loading: false });
        bump();
      } catch (e) {
        set({ loading: false, error: (e as Error).message });
      }
    },

    async ensureDefaultWorkspace() {
      const ws = await blockstoreApi.ensureDefaultWorkspace();
      bump();
      return ws;
    },

    async loadSubtree(workspaceID, rootID) {
      await blockstoreApi.getSubtree(workspaceID, rootID);
      // Seed watermark if not yet present so the WS filter recognises the workspace.
      if (!(workspaceID in readLastOpIds())) {
        svc().set_last_op_id(workspaceID, 0);
      }
      bump();
    },

    async loadTypeDefs(workspaceID) {
      await blockstoreApi.listTypeDefs(workspaceID);
      bump();
    },

    async catchup(workspaceID) {
      // Rust's catchup applies all server ops atomically. No JS-side replay.
      await blockstoreApi.catchupOps(workspaceID);
      bump();
    },

    setLastOpId(workspaceID, id) {
      svc().set_last_op_id(workspaceID, id);
      bump();
    },

    requestFocus(blockID) {
      set({ pendingFocusBlockID: blockID });
    },

    clearPendingFocus() {
      set((s) => (s.pendingFocusBlockID === null ? s : { pendingFocusBlockID: null }));
    },

    toggleSelection(blockID: string) {
      set((s) => {
        const has = s.selectedBlockIDs.includes(blockID);
        return {
          selectedBlockIDs: has
            ? s.selectedBlockIDs.filter((id) => id !== blockID)
            : [...s.selectedBlockIDs, blockID],
        };
      });
    },

    clearSelection() {
      set((s) => (s.selectedBlockIDs.length === 0 ? s : { selectedBlockIDs: [] }));
    },

    reset() {
      set({
        _tick: get()._tick + 1,
        pendingFocusBlockID: null,
        selectedBlockIDs: [],
        activeWorkspaceId: null,
        activeCommentBlockID: null,
        loading: false,
        error: null,
      });
    },
  },
}));

// Re-export selector hook types for convenience.
export type { Block, BlockRef, Workspace };
