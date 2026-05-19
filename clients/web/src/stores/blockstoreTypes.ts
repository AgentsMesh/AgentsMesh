import type { Block, BlockRef, Workspace } from "@/lib/api/blockstoreTypes";

export interface BlockstoreState {
  _tick: number;

  pendingFocusBlockID: string | null;
  activeWorkspaceId: string | null;
  activeCommentBlockID: string | null;
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
  requestFocus(blockID: string): void;
  clearPendingFocus(): void;
  toggleSelection(blockID: string): void;
  clearSelection(): void;
  reset(): void;
}

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
