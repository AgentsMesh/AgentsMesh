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
        if (result.current.splitTree!.children[1].type === "leaf") {
          expect(result.current.splitTree!.children[1].paneId).not.toBe("");
        }
      }
      expect(result.current.panes).toHaveLength(2);
      expect(result.current.panes[1].podKey).toBe("pod-2");
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
});

// NOTE: Relay Connection Pool tests live in relayConnection.test.ts.
