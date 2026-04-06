import type { SplitTreeNode, SplitTreeLeaf } from "./workspaceTypes";

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
