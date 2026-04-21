import type { Block, BlockRef, Workspace } from "@/lib/api/blockstoreTypes";

// UI-only state. All block/ref/workspace data lives in Rust BlockstoreService
// and is read through selectors; this store only tracks transient UI signals.
export interface BlockstoreState {
  _tick: number;

  /** Transient one-shot focus request. A renderer whose block id matches grabs
   *  the DOM focus on next render and clears the signal so it fires exactly once. */
  pendingFocusBlockID: string | null;
  /** The workspace the Blocks page is currently viewing. Sidebar + main
   *  read from this so the master-detail layout stays in sync without
   *  prop-drilling through IDEShell. */
  activeWorkspaceId: string | null;
  /** Block whose comments are currently shown in the right rail. Null when
   *  no block is selected; the rail then shows an empty state. */
  activeCommentBlockID: string | null;
  /** Selected block ids for batch operations (delete / duplicate / move). */
  selectedBlockIDs: string[];
  loading: boolean;
  error: string | null;

  actions: BlockstoreActions;
}

export interface BlockstoreActions {
  loadWorkspaces(): Promise<void>;
  ensureDefaultWorkspace(): Promise<Workspace>;
  loadSubtree(workspaceID: string, rootID: string): Promise<void>;
  loadTypeDefs(workspaceID: string): Promise<void>;
  catchup(workspaceID: string): Promise<void>;
  setActiveWorkspaceId(id: string | null): void;
  setActiveCommentBlockID(id: string | null): void;

  setLastOpId(workspaceID: string, id: number): void;
  /** Mark a block as "should grab focus on next render". */
  requestFocus(blockID: string): void;
  /** Consumed by the renderer that took the focus; clears the signal. */
  clearPendingFocus(): void;
  /** Toggle a block id in the selection set (used by shift/ctrl/cmd+click). */
  toggleSelection(blockID: string): void;
  clearSelection(): void;
  reset(): void;
}

// sortedInsertByOrder maintains nestChildren[parent] sorted by order_key (nulls last).
export function compareOrderKey(a: BlockRef | undefined, b: BlockRef | undefined): number {
  const ak = a?.order_key ?? null;
  const bk = b?.order_key ?? null;
  if (ak === bk) return (a?.id ?? 0) - (b?.id ?? 0);
  if (ak === null) return 1;
  if (bk === null) return -1;
  if (ak < bk) return -1;
  if (ak > bk) return 1;
  return 0;
}

export type BlocksMap = Record<string, Block>;
export type RefsMap = Record<number, BlockRef>;
export type ChildrenIndex = Record<string, number[]>;
export type BacklinksIndex = Record<string, number[]>;
export type WorkspacesMap = Record<string, Workspace>;
export type LastOpIdMap = Record<string, number>;
