import { describe, it, expect, beforeEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useWorkspaceStore } from "../workspace";

describe("Workspace Store", () => {
  beforeEach(() => {
    localStorage.clear();
    // Reset store to initial state
    useWorkspaceStore.setState({
      panes: [],
      activePane: null,
      splitTree: null,
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,
    });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      expect(result.current.panes).toEqual([]);
      expect(result.current.activePane).toBeNull();
      expect(result.current.splitTree).toBeNull();
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.terminalFontSize).toBe(14);
    });
  });

  describe("panes management", () => {
    it("should add a new pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-123");
      });

      expect(result.current.panes).toHaveLength(1);
      expect(result.current.panes[0].podKey).toBe("pod-123");
      expect(result.current.activePane).toBe(paneId!);
    });

    it("should add a new pane with podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-123");
      });

      expect(result.current.panes[0].podKey).toBe("pod-123");
    });

    it("should create a split tree leaf for first pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-123");
      });

      expect(result.current.splitTree).not.toBeNull();
      expect(result.current.splitTree!.type).toBe("leaf");
    });

    it("should create a split node when adding second pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      expect(result.current.splitTree).not.toBeNull();
      expect(result.current.splitTree!.type).toBe("split");
      if (result.current.splitTree!.type === "split") {
        expect(result.current.splitTree!.direction).toBe("horizontal");
      }
    });

    it("should return existing pane id if pod already open", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      let secondId: string;
      act(() => {
        firstId = result.current.addPane("pod-123");
        secondId = result.current.addPane("pod-123");
      });

      expect(firstId!).toBe(secondId!);
      expect(result.current.panes).toHaveLength(1);
    });

    it("should remove a pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-123");
      });

      expect(result.current.panes).toHaveLength(1);

      act(() => {
        result.current.removePane(paneId!);
      });

      expect(result.current.panes).toHaveLength(0);
      expect(result.current.activePane).toBeNull();
      expect(result.current.splitTree).toBeNull();
    });

    it("should set next pane as active when active pane is removed", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      act(() => {
        firstId = result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.activePane).toBe(result.current.panes[1].id);

      act(() => {
        result.current.removePane(result.current.panes[1].id);
      });

      expect(result.current.panes).toHaveLength(1);
      expect(result.current.activePane).toBe(firstId!);
    });

    it("should clear all panes", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      expect(result.current.panes).toHaveLength(3);

      act(() => {
        result.current.clearAllPanes();
      });

      expect(result.current.panes).toHaveLength(0);
      expect(result.current.activePane).toBeNull();
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.splitTree).toBeNull();
    });
  });

  describe("active pane", () => {
    it("should set active pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      act(() => {
        firstId = result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      act(() => {
        result.current.setActivePane(firstId!);
      });

      expect(result.current.activePane).toBe(firstId!);
    });

    it("should set active pane to null", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.setActivePane(null);
      });

      expect(result.current.activePane).toBeNull();
    });
  });

  describe("split tree", () => {
    it("should split pane horizontally", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-1");
      });

      act(() => {
        result.current.splitPane(paneId!, "horizontal", "pod-2");
      });

      expect(result.current.splitTree).not.toBeNull();
      expect(result.current.splitTree!.type).toBe("split");
      if (result.current.splitTree!.type === "split") {
        expect(result.current.splitTree!.direction).toBe("horizontal");
        expect(result.current.splitTree!.children[1].type).toBe("leaf");
        // Second child has a real pane
        if (result.current.splitTree!.children[1].type === "leaf") {
          expect(result.current.splitTree!.children[1].paneId).not.toBe("");
        }
      }
      // New pane should be added
      expect(result.current.panes).toHaveLength(2);
      expect(result.current.panes[1].podKey).toBe("pod-2");
      // Active pane should be the new pane
      expect(result.current.activePane).toBe(result.current.panes[1].id);
    });

    it("should split pane vertically", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-1");
      });

      act(() => {
        result.current.splitPane(paneId!, "vertical", "pod-3");
      });

      expect(result.current.splitTree!.type).toBe("split");
      if (result.current.splitTree!.type === "split") {
        expect(result.current.splitTree!.direction).toBe("vertical");
      }
      expect(result.current.panes).toHaveLength(2);
      expect(result.current.panes[1].podKey).toBe("pod-3");
    });

    it("should update split sizes", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      const tree = result.current.splitTree;
      expect(tree).not.toBeNull();
      expect(tree!.type).toBe("split");

      act(() => {
        result.current.updateSplitSizes(tree!.id, [30, 70]);
      });

      if (result.current.splitTree!.type === "split") {
        expect(result.current.splitTree!.sizes).toEqual([30, 70]);
      }
    });
  });

  describe("mobile state", () => {
    it("should set mobile active index", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      act(() => {
        result.current.setMobileActiveIndex(1);
      });

      expect(result.current.mobileActiveIndex).toBe(1);
      expect(result.current.activePane).toBe(result.current.panes[1].id);
    });

    it("should not set invalid mobile active index", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      const initialIndex = result.current.mobileActiveIndex;

      act(() => {
        result.current.setMobileActiveIndex(99);
      });

      expect(result.current.mobileActiveIndex).toBe(initialIndex);
    });
  });

  describe("terminal settings", () => {
    it("should set terminal font size", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(16);
      });

      expect(result.current.terminalFontSize).toBe(16);
    });

    it("should clamp font size to minimum", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(5);
      });

      expect(result.current.terminalFontSize).toBe(10);
    });

    it("should clamp font size to maximum", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(50);
      });

      expect(result.current.terminalFontSize).toBe(24);
    });
  });

  describe("getPaneByPodKey", () => {
    it("should find pane by podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-123");
      });

      const pane = result.current.getPaneByPodKey("pod-123");
      expect(pane).toBeDefined();
      expect(pane?.podKey).toBe("pod-123");
    });

    it("should return undefined for non-existent podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      const pane = result.current.getPaneByPodKey("non-existent");
      expect(pane).toBeUndefined();
    });
  });

  describe("hydration", () => {
    it("should set hydration state", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setHasHydrated(true);
      });

      expect(result.current._hasHydrated).toBe(true);
    });
  });

  describe("mobileActiveIndex sync", () => {
    it("setActivePane should sync mobileActiveIndex", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      act(() => {
        firstId = result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      // Active is last pane (index 2)
      expect(result.current.mobileActiveIndex).toBe(2);

      // Switch to first pane
      act(() => {
        result.current.setActivePane(firstId!);
      });

      expect(result.current.activePane).toBe(firstId!);
      expect(result.current.mobileActiveIndex).toBe(0);
    });

    it("setActivePane(null) should reset mobileActiveIndex to 0", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      act(() => {
        result.current.setActivePane(null);
      });

      expect(result.current.mobileActiveIndex).toBe(0);
    });

    it("addPane (existing pod) should sync mobileActiveIndex", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      // Re-add pod-1 — should switch to index 0
      act(() => {
        result.current.addPane("pod-1");
      });

      expect(result.current.activePane).toBe(result.current.panes[0].id);
      expect(result.current.mobileActiveIndex).toBe(0);
    });

    it("addPane (new pod) should set mobileActiveIndex to new pane index", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      act(() => {
        result.current.addPane("pod-3");
      });

      // New pane appended at index 2
      expect(result.current.mobileActiveIndex).toBe(2);
      expect(result.current.panes[2].podKey).toBe("pod-3");
    });

    it("removePane should shift mobileActiveIndex when removing pane before active", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      // Switch to middle pane (index 1)
      act(() => {
        result.current.setActivePane(result.current.panes[1].id);
      });
      expect(result.current.mobileActiveIndex).toBe(1);

      // Remove first pane (before active) — index should shift down
      act(() => {
        result.current.removePane(result.current.panes[0].id);
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.panes[0].podKey).toBe("pod-2");
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.activePane).toBe(result.current.panes[0].id);
    });

    it("removePane (active pane) should reset mobileActiveIndex to 0", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      // Active is last pane (index 2)
      expect(result.current.mobileActiveIndex).toBe(2);

      // Remove active pane — falls back to first
      act(() => {
        result.current.removePane(result.current.panes[2].id);
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.activePane).toBe(result.current.panes[0].id);
      expect(result.current.mobileActiveIndex).toBe(0);
    });

    it("removePane should not shift mobileActiveIndex when removing pane after active", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      // Switch to first pane (index 0)
      act(() => {
        result.current.setActivePane(result.current.panes[0].id);
      });
      expect(result.current.mobileActiveIndex).toBe(0);

      // Remove last pane (after active) — index stays
      act(() => {
        result.current.removePane(result.current.panes[2].id);
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.activePane).toBe(result.current.panes[0].id);
    });
  });
});

// NOTE: Terminal Connection Pool tests have been removed because the API has fundamentally changed.
// The terminalConnection.ts now uses Relay architecture with:
// 1. Async connect() that requires API call to get Relay connection info
// 2. Binary protocol message encoding/decoding (not JSON)
// 3. Different message types (MsgType.Input, MsgType.Resize, etc.)
//
// TODO: Rewrite these tests when Relay architecture is stable. Tests should:
// 1. Mock podApi.getTerminalConnection to return relay info
// 2. Use async/await for connect()
// 3. Test binary protocol message encoding/decoding
// 4. Test Relay-specific message types (Snapshot, Output, RunnerDisconnected, etc.)
