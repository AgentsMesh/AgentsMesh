import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Terminal as XTerm } from "@xterm/xterm";

// Re-export terminalPool for component convenience
export { terminalPool } from "./terminalConnection";

/**
 * Terminal instance registry for cross-component access
 * Allows TerminalToolbar to access xterm instances from TerminalPane
 */
class TerminalRegistry {
  private terminals: Map<string, XTerm> = new Map();

  register(podKey: string, terminal: XTerm): void {
    this.terminals.set(podKey, terminal);
  }

  unregister(podKey: string): void {
    this.terminals.delete(podKey);
  }

  get(podKey: string): XTerm | undefined {
    return this.terminals.get(podKey);
  }

  scrollToBottom(podKey: string): void {
    const terminal = this.terminals.get(podKey);
    if (terminal) {
      terminal.scrollToBottom();
    }
  }
}

export const terminalRegistry = new TerminalRegistry();

/**
 * Terminal pane configuration
 */
export interface TerminalPane {
  id: string;
  podKey: string;
}

/**
 * Split tree types for flexible split layouts
 */
export type SplitDirection = "horizontal" | "vertical";

export type SplitTreeLeaf = {
  type: "leaf";
  id: string;
  paneId: string;
};

export type SplitTreeSplit = {
  type: "split";
  id: string;
  direction: SplitDirection;
  children: [SplitTreeNode, SplitTreeNode];
  sizes: [number, number];
};

export type SplitTreeNode = SplitTreeLeaf | SplitTreeSplit;

// Keep GridLayout type for migration compatibility
export type GridLayoutType = "1x1" | "1x2" | "2x1" | "2x2" | "custom";

export interface GridLayout {
  type: GridLayoutType;
  rows: number;
  cols: number;
}

/**
 * Workspace state management
 */
interface WorkspaceState {
  panes: TerminalPane[];
  activePane: string | null;
  splitTree: SplitTreeNode | null;
  mobileActiveIndex: number;
  terminalFontSize: number;

  // Actions
  addPane: (podKey: string) => string;
  removePane: (paneId: string) => void;
  setActivePane: (paneId: string | null) => void;
  splitPane: (paneId: string, direction: SplitDirection, podKey: string) => void;
  closePaneFromTree: (paneId: string) => void;
  updateSplitSizes: (splitId: string, sizes: [number, number]) => void;
  setMobileActiveIndex: (index: number) => void;
  setTerminalFontSize: (size: number) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => TerminalPane | undefined;

  // Hydration
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

const generatePaneId = () => `pane-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
const generateNodeId = () => `node-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;

// --- Split tree helpers ---

/** Find the last leaf node in the tree (for addPane auto-split) */
function findLastLeaf(node: SplitTreeNode): SplitTreeLeaf | null {
  if (node.type === "leaf") return node;
  return findLastLeaf(node.children[1]) || findLastLeaf(node.children[0]);
}

/** Find a leaf by paneId */
function findLeafByPaneId(node: SplitTreeNode, paneId: string): SplitTreeLeaf | null {
  if (node.type === "leaf") return node.paneId === paneId ? node : null;
  return findLeafByPaneId(node.children[0], paneId) || findLeafByPaneId(node.children[1], paneId);
}

/** Replace a node in the tree by its id, returning a new tree */
function replaceNode(tree: SplitTreeNode, nodeId: string, replacement: SplitTreeNode): SplitTreeNode {
  if (tree.id === nodeId) return replacement;
  if (tree.type === "leaf") return tree;
  return {
    ...tree,
    children: [
      replaceNode(tree.children[0], nodeId, replacement),
      replaceNode(tree.children[1], nodeId, replacement),
    ],
  };
}

/** Remove a leaf from the tree — its sibling replaces the parent split */
function removeLeaf(tree: SplitTreeNode, leafId: string): SplitTreeNode | null {
  if (tree.type === "leaf") {
    return tree.id === leafId ? null : tree;
  }
  const [left, right] = tree.children;
  if (left.id === leafId) return right;
  if (right.id === leafId) return left;
  const newLeft = removeLeaf(left, leafId);
  const newRight = removeLeaf(right, leafId);
  if (!newLeft) return newRight;
  if (!newRight) return newLeft;
  return { ...tree, children: [newLeft, newRight] };
}

/** Update sizes on a split node by id */
function updateSizes(tree: SplitTreeNode, splitId: string, sizes: [number, number]): SplitTreeNode {
  if (tree.type === "leaf") return tree;
  if (tree.id === splitId) return { ...tree, sizes };
  return {
    ...tree,
    children: [
      updateSizes(tree.children[0], splitId, sizes),
      updateSizes(tree.children[1], splitId, sizes),
    ],
  };
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      panes: [],
      activePane: null,
      splitTree: null,
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,

      addPane: (podKey) => {
        const panes = get().panes;
        const existingIndex = panes.findIndex((p) => p.podKey === podKey);
        if (existingIndex >= 0) {
          const existingPane = panes[existingIndex];
          set({ activePane: existingPane.id, mobileActiveIndex: existingIndex });
          return existingPane.id;
        }

        const id = generatePaneId();
        const newPane: TerminalPane = { id, podKey };
        const tree = get().splitTree;
        const leafNode: SplitTreeLeaf = { type: "leaf", id: generateNodeId(), paneId: id };

        let newTree: SplitTreeNode;
        if (!tree) {
          // First pane — single leaf
          newTree = leafNode;
        } else {
          // Split the last leaf horizontally to add new pane
          const lastLeaf = findLastLeaf(tree);
          if (lastLeaf) {
            const splitNode: SplitTreeSplit = {
              type: "split",
              id: generateNodeId(),
              direction: "horizontal",
              children: [{ ...lastLeaf }, leafNode],
              sizes: [50, 50],
            };
            newTree = replaceNode(tree, lastLeaf.id, splitNode);
          } else {
            newTree = leafNode;
          }
        }

        set((state) => ({
          panes: [...state.panes, newPane],
          activePane: id,
          mobileActiveIndex: state.panes.length,
          splitTree: newTree,
        }));

        return id;
      },

      removePane: (paneId) => {
        set((state) => {
          const removedIndex = state.panes.findIndex((p) => p.id === paneId);
          const newPanes = state.panes.filter((p) => p.id !== paneId);
          const wasActive = state.activePane === paneId;

          // Remove from split tree
          let newTree = state.splitTree;
          if (newTree) {
            // Find the leaf node for this pane
            const leaf = findLeafByPaneId(newTree, paneId);
            if (leaf) {
              newTree = removeLeaf(newTree, leaf.id);
            }
          }

          let newMobileIndex: number;
          if (wasActive) {
            newMobileIndex = 0;
          } else if (removedIndex >= 0 && removedIndex < state.mobileActiveIndex) {
            newMobileIndex = state.mobileActiveIndex - 1;
          } else {
            newMobileIndex = state.mobileActiveIndex;
          }
          newMobileIndex = Math.min(newMobileIndex, Math.max(0, newPanes.length - 1));

          return {
            panes: newPanes,
            activePane: wasActive ? (newPanes[0]?.id || null) : state.activePane,
            mobileActiveIndex: newMobileIndex,
            splitTree: newTree || null,
          };
        });
      },

      setActivePane: (paneId) => {
        set((state) => {
          const mobileIndex = paneId ? state.panes.findIndex((p) => p.id === paneId) : 0;
          return {
            activePane: paneId,
            mobileActiveIndex: Math.max(0, mobileIndex),
          };
        });
      },

      splitPane: (paneId, direction, podKey) => {
        set((state) => {
          const tree = state.splitTree;
          if (!tree) return state;

          const leaf = findLeafByPaneId(tree, paneId);
          if (!leaf) return state;

          // Create a new pane with the selected pod
          const newPaneId = generatePaneId();
          const newPane: TerminalPane = { id: newPaneId, podKey };
          const newLeaf: SplitTreeLeaf = { type: "leaf", id: generateNodeId(), paneId: newPaneId };
          const splitNode: SplitTreeSplit = {
            type: "split",
            id: generateNodeId(),
            direction,
            children: [{ ...leaf }, newLeaf],
            sizes: [50, 50],
          };
          const newTree = replaceNode(tree, leaf.id, splitNode);
          return {
            panes: [...state.panes, newPane],
            activePane: newPaneId,
            splitTree: newTree,
          };
        });
      },

      closePaneFromTree: (paneId) => {
        // Alias for removePane — removes from both panes array and tree
        get().removePane(paneId);
      },

      updateSplitSizes: (splitId, sizes) => {
        set((state) => {
          if (!state.splitTree) return state;
          return { splitTree: updateSizes(state.splitTree, splitId, sizes) };
        });
      },

      setMobileActiveIndex: (index) => {
        const panes = get().panes;
        if (index >= 0 && index < panes.length) {
          set({ mobileActiveIndex: index, activePane: panes[index]?.id || null });
        }
      },

      setTerminalFontSize: (size) => {
        set({ terminalFontSize: Math.min(Math.max(size, 10), 24) });
      },

      clearAllPanes: () => {
        set({ panes: [], activePane: null, mobileActiveIndex: 0, splitTree: null });
      },

      getPaneByPodKey: (podKey) => {
        return get().panes.find((p) => p.podKey === podKey);
      },

      setHasHydrated: (state) => {
        set({ _hasHydrated: state });
      },
    }),
    {
      name: "agentsmesh-workspace",
      version: 3,
      migrate: (persistedState: unknown, version: number) => {
        const state = persistedState as Record<string, unknown>;
        if (version < 1 && Array.isArray(state.panes)) {
          // v0 → v1: remove obsolete `title` field from persisted panes
          state.panes = (state.panes as Record<string, unknown>[]).map(
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            ({ title, ...rest }) => rest,
          );
        }
        if (version < 2 && Array.isArray(state.panes)) {
          // v1 → v2: remove obsolete `isActive` field
          state.panes = (state.panes as Record<string, unknown>[]).map(
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            ({ isActive, ...rest }) => rest,
          );
        }
        if (version < 3 && Array.isArray(state.panes)) {
          // v2 → v3: migrate gridLayout + panes to splitTree
          const panes = state.panes as { id: string; podKey: string }[];
          // Remove obsolete gridPosition from panes
          state.panes = panes.map(({ id, podKey }) => ({ id, podKey }));

          // Build split tree from existing panes + gridLayout
          const grid = state.gridLayout as { type: string; rows: number; cols: number } | undefined;
          delete state.gridLayout;

          if (panes.length === 0) {
            state.splitTree = null;
          } else if (panes.length === 1) {
            state.splitTree = { type: "leaf", id: generateNodeId(), paneId: panes[0].id };
          } else {
            // Build tree matching old grid layout
            const direction: SplitDirection =
              grid?.type === "2x1" ? "vertical" :
              grid?.type === "2x2" ? "vertical" : "horizontal";

            if (panes.length === 2) {
              state.splitTree = {
                type: "split",
                id: generateNodeId(),
                direction,
                children: [
                  { type: "leaf", id: generateNodeId(), paneId: panes[0].id },
                  { type: "leaf", id: generateNodeId(), paneId: panes[1].id },
                ],
                sizes: [50, 50],
              };
            } else if (panes.length <= 4 && grid?.type === "2x2") {
              // 2x2 grid: vertical split with 2 horizontal splits inside
              const topRow: SplitTreeNode = panes.length >= 2 ? {
                type: "split", id: generateNodeId(), direction: "horizontal",
                children: [
                  { type: "leaf", id: generateNodeId(), paneId: panes[0].id },
                  { type: "leaf", id: generateNodeId(), paneId: panes[1].id },
                ],
                sizes: [50, 50],
              } : { type: "leaf", id: generateNodeId(), paneId: panes[0].id };

              const bottomRow: SplitTreeNode = panes.length >= 4 ? {
                type: "split", id: generateNodeId(), direction: "horizontal",
                children: [
                  { type: "leaf", id: generateNodeId(), paneId: panes[2].id },
                  { type: "leaf", id: generateNodeId(), paneId: panes[3].id },
                ],
                sizes: [50, 50],
              } : panes.length >= 3 ? {
                type: "split", id: generateNodeId(), direction: "horizontal",
                children: [
                  { type: "leaf", id: generateNodeId(), paneId: panes[2].id },
                  { type: "leaf", id: generateNodeId(), paneId: "" },
                ],
                sizes: [50, 50],
              } : { type: "leaf", id: generateNodeId(), paneId: "" };

              state.splitTree = {
                type: "split", id: generateNodeId(), direction: "vertical",
                children: [topRow, bottomRow],
                sizes: [50, 50],
              };
            } else {
              // Fallback: chain horizontal splits
              let tree: SplitTreeNode = { type: "leaf", id: generateNodeId(), paneId: panes[0].id };
              for (let i = 1; i < panes.length; i++) {
                tree = {
                  type: "split", id: generateNodeId(), direction: "horizontal",
                  children: [tree, { type: "leaf", id: generateNodeId(), paneId: panes[i].id }],
                  sizes: [50, 50],
                };
              }
              state.splitTree = tree;
            }
          }
        }
        return state as unknown as WorkspaceState;
      },
      partialize: (state) => ({
        panes: state.panes,
        activePane: state.activePane,
        splitTree: state.splitTree,
        mobileActiveIndex: state.mobileActiveIndex,
        terminalFontSize: state.terminalFontSize,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
