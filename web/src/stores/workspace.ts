import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { WorkspacePane, SplitTreeLeaf, SplitTreeSplit, WorkspaceState } from "./workspaceTypes";
import {
  generatePaneId, generateNodeId,
  findLastLeaf, findLeafByPaneId,
  replaceNode, removeLeaf, updateSizes,
  migrateWorkspaceState,
} from "./workspaceSplitTree";

// Re-export types and singletons for consumer convenience
export { relayPool } from "./relayConnection";
export { terminalRegistry } from "./workspaceTypes";
export type {
  WorkspacePane, SplitDirection, SplitTreeLeaf, SplitTreeSplit, SplitTreeNode,
  GridLayoutType, GridLayout, WorkspaceState,
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
        const existingIndex = panes.findIndex((p) => p.podKey === podKey);
        if (existingIndex >= 0) {
          const existingPane = panes[existingIndex];
          set({ activePane: existingPane.id, mobileActiveIndex: existingIndex });
          return existingPane.id;
        }

        const id = generatePaneId();
        const newPane: WorkspacePane = { id, podKey };
        const tree = get().splitTree;
        const leafNode: SplitTreeLeaf = { type: "leaf", id: generateNodeId(), paneId: id };

        let newTree;
        if (!tree) {
          newTree = leafNode;
        } else {
          const lastLeaf = findLastLeaf(tree);
          if (lastLeaf) {
            const splitNode: SplitTreeSplit = {
              type: "split", id: generateNodeId(), direction: "horizontal",
              children: [{ ...lastLeaf }, leafNode], sizes: [50, 50],
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
          const splitNode: SplitTreeSplit = {
            type: "split", id: generateNodeId(), direction,
            children: [{ ...leaf }, newLeaf], sizes: [50, 50],
          };
          const newTree = replaceNode(tree, leaf.id, splitNode);
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
      version: 3,
      migrate: migrateWorkspaceState as (state: unknown, version: number) => WorkspaceState,
      partialize: (state) => ({
        panes: state.panes,
        activePane: state.activePane,
        splitTree: state.splitTree,
        mobileActiveIndex: state.mobileActiveIndex,
        terminalFontSize: state.terminalFontSize,
      }),
      onRehydrateStorage: () => (state) => { state?.setHasHydrated(true); },
    }
  )
);
