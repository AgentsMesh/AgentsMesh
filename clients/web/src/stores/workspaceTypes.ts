import type { Terminal as XTerm } from "@xterm/xterm";

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

export interface WorkspacePane {
  id: string;
  podKey: string;
}

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
  children: SplitTreeNode[];
  sizes: number[];
};

export type SplitTreeNode = SplitTreeLeaf | SplitTreeSplit;

export interface WorkspaceState {
  panes: WorkspacePane[];
  activePane: string | null;
  splitTree: SplitTreeNode | null;
  mobileActiveIndex: number;
  terminalFontSize: number;

  addPane: (podKey: string) => string;
  removePane: (paneId: string) => void;
  setActivePane: (paneId: string | null) => void;
  splitPane: (paneId: string, direction: SplitDirection, podKey: string) => void;
  closePaneFromTree: (paneId: string) => void;
  updateSplitSizes: (splitId: string, sizes: number[]) => void;
  setMobileActiveIndex: (index: number) => void;
  setTerminalFontSize: (size: number) => void;
  removePaneByPodKey: (podKey: string) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => WorkspacePane | undefined;

  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}
