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
  gridPosition?: {
    x: number;
    y: number;
    w: number;
    h: number;
  };
}

/**
 * Grid layout configuration
 */
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
  gridLayout: GridLayout;
  mobileActiveIndex: number;
  terminalFontSize: number;

  // Actions
  addPane: (podKey: string) => string;
  removePane: (paneId: string) => void;
  setActivePane: (paneId: string | null) => void;
  updatePanePosition: (paneId: string, position: TerminalPane["gridPosition"]) => void;
  setGridLayout: (layout: GridLayout) => void;
  setMobileActiveIndex: (index: number) => void;
  setTerminalFontSize: (size: number) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => TerminalPane | undefined;

  // Hydration
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

const generatePaneId = () => `pane-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      panes: [],
      activePane: null,
      gridLayout: { type: "1x1", rows: 1, cols: 1 },
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
        const newPane: TerminalPane = {
          id,
          podKey,
          gridPosition: {
            x: panes.length % 2,
            y: Math.floor(panes.length / 2),
            w: 1,
            h: 1,
          },
        };

        set((state) => ({
          panes: [...state.panes, newPane],
          activePane: id,
          mobileActiveIndex: state.panes.length,
        }));

        return id;
      },

      removePane: (paneId) => {
        set((state) => {
          const removedIndex = state.panes.findIndex((p) => p.id === paneId);
          const newPanes = state.panes.filter((p) => p.id !== paneId);
          const wasActive = state.activePane === paneId;

          let newMobileIndex: number;
          if (wasActive) {
            // Active pane removed — fall back to first pane
            newMobileIndex = 0;
          } else if (removedIndex >= 0 && removedIndex < state.mobileActiveIndex) {
            // Non-active pane removed BEFORE current index — shift down
            newMobileIndex = state.mobileActiveIndex - 1;
          } else {
            newMobileIndex = state.mobileActiveIndex;
          }
          // Safety clamp
          newMobileIndex = Math.min(newMobileIndex, Math.max(0, newPanes.length - 1));

          return {
            panes: newPanes,
            activePane: wasActive ? (newPanes[0]?.id || null) : state.activePane,
            mobileActiveIndex: newMobileIndex,
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

      updatePanePosition: (paneId, position) => {
        set((state) => ({
          panes: state.panes.map((p) => (p.id === paneId ? { ...p, gridPosition: position } : p)),
        }));
      },

      setGridLayout: (layout) => {
        set({ gridLayout: layout });
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
        set({ panes: [], activePane: null, mobileActiveIndex: 0 });
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
      version: 2,
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
          // v1 → v2: remove obsolete `isActive` field (derived from activePane now)
          state.panes = (state.panes as Record<string, unknown>[]).map(
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            ({ isActive, ...rest }) => rest,
          );
        }
        return state as unknown as WorkspaceState;
      },
      partialize: (state) => ({
        panes: state.panes,
        activePane: state.activePane,
        gridLayout: state.gridLayout,
        mobileActiveIndex: state.mobileActiveIndex,
        terminalFontSize: state.terminalFontSize,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
