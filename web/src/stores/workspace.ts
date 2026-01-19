import { create } from "zustand";
import { persist } from "zustand/middleware";

// Re-export terminalPool for component convenience
export { terminalPool } from "./terminalConnection";

/**
 * Terminal pane configuration
 */
export interface TerminalPane {
  id: string;
  podKey: string;
  title: string;
  isActive: boolean;
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
  addPane: (podKey: string, title?: string) => string;
  removePane: (paneId: string) => void;
  setActivePane: (paneId: string | null) => void;
  updatePanePosition: (paneId: string, position: TerminalPane["gridPosition"]) => void;
  updatePaneTitle: (podKey: string, title: string) => void;
  setGridLayout: (layout: GridLayout) => void;
  setMobileActiveIndex: (index: number) => void;
  setTerminalFontSize: (size: number) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => TerminalPane | undefined;

  // Hydration
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

const generatePaneId = () => `pane-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      panes: [],
      activePane: null,
      gridLayout: { type: "1x1", rows: 1, cols: 1 },
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,

      addPane: (podKey, title) => {
        const existingPane = get().panes.find((p) => p.podKey === podKey);
        if (existingPane) {
          set({ activePane: existingPane.id });
          return existingPane.id;
        }

        const id = generatePaneId();
        const panes = get().panes;
        const newPane: TerminalPane = {
          id,
          podKey,
          title: title || `Pod ${podKey.substring(0, 8)}`,
          isActive: true,
          gridPosition: {
            x: panes.length % 2,
            y: Math.floor(panes.length / 2),
            w: 1,
            h: 1,
          },
        };

        set((state) => ({
          panes: [...state.panes.map((p) => ({ ...p, isActive: false })), newPane],
          activePane: id,
        }));

        return id;
      },

      removePane: (paneId) => {
        set((state) => {
          const newPanes = state.panes.filter((p) => p.id !== paneId);
          const wasActive = state.activePane === paneId;
          return {
            panes: newPanes,
            activePane: wasActive ? (newPanes[0]?.id || null) : state.activePane,
            mobileActiveIndex: Math.min(state.mobileActiveIndex, Math.max(0, newPanes.length - 1)),
          };
        });
      },

      setActivePane: (paneId) => {
        set((state) => ({
          panes: state.panes.map((p) => ({ ...p, isActive: p.id === paneId })),
          activePane: paneId,
        }));
      },

      updatePanePosition: (paneId, position) => {
        set((state) => ({
          panes: state.panes.map((p) => (p.id === paneId ? { ...p, gridPosition: position } : p)),
        }));
      },

      updatePaneTitle: (podKey, title) => {
        set((state) => ({
          panes: state.panes.map((p) => (p.podKey === podKey ? { ...p, title } : p)),
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
