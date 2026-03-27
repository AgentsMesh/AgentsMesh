import type { Terminal as XTerm } from "@xterm/xterm";

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

/** Terminal pane configuration */
export interface WorkspacePane {
  id: string;
  podKey: string;
}

/** Split tree types for flexible split layouts */
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

export interface WorkspaceState {
  panes: WorkspacePane[];
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
  removePaneByPodKey: (podKey: string) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => WorkspacePane | undefined;

  // Hydration
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}
