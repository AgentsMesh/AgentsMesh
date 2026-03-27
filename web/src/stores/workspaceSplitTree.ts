import type { SplitTreeNode, SplitTreeLeaf, SplitTreeSplit, SplitDirection } from "./workspaceTypes";

export const generatePaneId = () => `pane-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
export const generateNodeId = () => `node-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;

/** Find the last leaf node in the tree (for addPane auto-split) */
export function findLastLeaf(node: SplitTreeNode): SplitTreeLeaf | null {
  if (node.type === "leaf") return node;
  return findLastLeaf(node.children[1]) || findLastLeaf(node.children[0]);
}

/** Find a leaf by paneId */
export function findLeafByPaneId(node: SplitTreeNode, paneId: string): SplitTreeLeaf | null {
  if (node.type === "leaf") return node.paneId === paneId ? node : null;
  return findLeafByPaneId(node.children[0], paneId) || findLeafByPaneId(node.children[1], paneId);
}

/** Replace a node in the tree by its id, returning a new tree */
export function replaceNode(tree: SplitTreeNode, nodeId: string, replacement: SplitTreeNode): SplitTreeNode {
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
export function removeLeaf(tree: SplitTreeNode, leafId: string): SplitTreeNode | null {
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
export function updateSizes(tree: SplitTreeNode, splitId: string, sizes: [number, number]): SplitTreeNode {
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

/**
 * Persist migration: upgrade workspace state across schema versions.
 * Called by Zustand's persist middleware on rehydration.
 */
export function migrateWorkspaceState(persistedState: unknown, version: number): unknown {
  const state = persistedState as Record<string, unknown>;
  if (version < 1 && Array.isArray(state.panes)) {
    state.panes = (state.panes as Record<string, unknown>[]).map(
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      ({ title, ...rest }) => rest,
    );
  }
  if (version < 2 && Array.isArray(state.panes)) {
    state.panes = (state.panes as Record<string, unknown>[]).map(
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      ({ isActive, ...rest }) => rest,
    );
  }
  if (version < 3 && Array.isArray(state.panes)) {
    const panes = state.panes as { id: string; podKey: string }[];
    state.panes = panes.map(({ id, podKey }) => ({ id, podKey }));

    const grid = state.gridLayout as { type: string; rows: number; cols: number } | undefined;
    delete state.gridLayout;

    if (panes.length === 0) {
      state.splitTree = null;
    } else if (panes.length === 1) {
      state.splitTree = { type: "leaf", id: generateNodeId(), paneId: panes[0].id };
    } else {
      const direction: SplitDirection =
        grid?.type === "2x1" ? "vertical" :
        grid?.type === "2x2" ? "vertical" : "horizontal";

      if (panes.length === 2) {
        state.splitTree = {
          type: "split", id: generateNodeId(), direction,
          children: [
            { type: "leaf", id: generateNodeId(), paneId: panes[0].id },
            { type: "leaf", id: generateNodeId(), paneId: panes[1].id },
          ],
          sizes: [50, 50],
        };
      } else if (panes.length <= 4 && grid?.type === "2x2") {
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
  return state;
}
