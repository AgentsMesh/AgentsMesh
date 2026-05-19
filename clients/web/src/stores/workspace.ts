import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { WorkspacePane, SplitTreeLeaf, SplitTreeSplit, SplitTreeNode, WorkspaceState } from "./workspaceTypes";
import {
  generatePaneId, generateNodeId,
  findLeafByPaneId, findParentSplit,
  replaceNode, removeLeaf, updateSizes, insertChildAt,
} from "./workspaceSplitTree";

export { relayPool } from "./relayConnection";
export { terminalRegistry } from "./workspaceTypes";
export type {
  WorkspacePane, SplitDirection, SplitTreeLeaf, SplitTreeSplit, SplitTreeNode,
  WorkspaceState,
} from "./workspaceTypes";

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
        const existing = panes.find((p) => p.podKey === podKey);
        if (existing) {
          set({ activePane: existing.id, mobileActiveIndex: panes.indexOf(existing) });
          return existing.id;
        }

        const id = generatePaneId();
        const newPane: WorkspacePane = { id, podKey };
        const tree = get().splitTree;
        const newLeaf: SplitTreeLeaf = { type: "leaf", id: generateNodeId(), paneId: id };

        let newTree;
        if (!tree) {
          newTree = newLeaf;
        } else {
          newTree = addPaneToTree(tree, newLeaf, get().activePane);
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

          let newTree = state.splitTree;
          if (newTree) {
            const leaf = findLeafByPaneId(newTree, paneId);
            if (leaf) newTree = removeLeaf(newTree, leaf.id);
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
          return { activePane: paneId, mobileActiveIndex: Math.max(0, mobileIndex) };
        });
      },

      splitPane: (paneId, direction, podKey) => {
        set((state) => {
          const tree = state.splitTree;
          if (!tree) return state;
          const leaf = findLeafByPaneId(tree, paneId);
          if (!leaf) return state;

          const newPaneId = generatePaneId();
          const newPane: WorkspacePane = { id: newPaneId, podKey };
          const newLeaf: SplitTreeLeaf = { type: "leaf", id: generateNodeId(), paneId: newPaneId };

          const newTree = splitLeafInTree(tree, leaf, newLeaf, direction);
          return { panes: [...state.panes, newPane], activePane: newPaneId, splitTree: newTree };
        });
      },

      closePaneFromTree: (paneId) => { get().removePane(paneId); },
      removePaneByPodKey: (podKey) => {
        const pane = get().panes.find((p) => p.podKey === podKey);
        if (pane) get().removePane(pane.id);
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

      getPaneByPodKey: (podKey) => get().panes.find((p) => p.podKey === podKey),

      setHasHydrated: (state) => { set({ _hasHydrated: state }); },
    }),
    {
      name: "agentsmesh-workspace",
      version: 4,
      partialize: (state) => ({
        panes: state.panes,
        activePane: state.activePane,
        splitTree: state.splitTree,
        mobileActiveIndex: state.mobileActiveIndex,
        terminalFontSize: state.terminalFontSize,
      }),
      // Workaround: pre-v4 localStorage serialised `panes: null`; zustand merge would shadow `[]` initial.
      merge: (persisted, current) => {
        const p = (persisted ?? {}) as Partial<WorkspaceState>;
        return {
          ...current,
          ...p,
          panes: Array.isArray(p.panes) ? p.panes : current.panes,
        };
      },
      onRehydrateStorage: () => (state) => { state?.setHasHydrated(true); },
    }
  )
);

function addPaneToTree(
  tree: SplitTreeNode,
  newLeaf: SplitTreeLeaf,
  activePaneId: string | null,
): SplitTreeNode {
  const activeLeaf = activePaneId ? findLeafByPaneId(tree, activePaneId) : null;
  const parent = activeLeaf ? findParentSplit(tree, activeLeaf.id) : null;

  if (parent && activeLeaf) {
    const idx = parent.children.findIndex((c) => c.id === activeLeaf.id);
    const evenSize = 100 / (parent.children.length + 1);
    const evenSizes = Array.from({ length: parent.children.length + 1 }, () => evenSize);
    return insertChildAt(tree, parent.id, newLeaf, idx, evenSizes);
  }

  const split: SplitTreeSplit = {
    type: "split", id: generateNodeId(), direction: "horizontal",
    children: [tree, newLeaf], sizes: [50, 50],
  };
  return split;
}

function splitLeafInTree(
  tree: SplitTreeNode,
  leaf: SplitTreeLeaf,
  newLeaf: SplitTreeLeaf,
  direction: "horizontal" | "vertical",
): SplitTreeNode {
  const parent = findParentSplit(tree, leaf.id);

  if (parent && parent.direction === direction) {
    const idx = parent.children.findIndex((c) => c.id === leaf.id);
    const targetSize = parent.sizes[idx];
    const newSizes = [...parent.sizes];
    newSizes[idx] = targetSize / 2;
    newSizes.splice(idx + 1, 0, targetSize / 2);
    return insertChildAt(tree, parent.id, newLeaf, idx, newSizes);
  }

  const splitNode: SplitTreeSplit = {
    type: "split", id: generateNodeId(), direction,
    children: [{ ...leaf }, newLeaf], sizes: [50, 50],
  };
  return replaceNode(tree, leaf.id, splitNode);
}
