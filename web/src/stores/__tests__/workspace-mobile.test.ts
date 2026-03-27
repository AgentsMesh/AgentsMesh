import { describe, it, expect, beforeEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useWorkspaceStore } from "../workspace";

describe("Workspace Store - Additional", () => {
  beforeEach(() => {
    localStorage.clear();
    useWorkspaceStore.setState({
      panes: [],
      activePane: null,
      splitTree: null,
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,
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

      expect(result.current.mobileActiveIndex).toBe(2);

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

      act(() => {
        result.current.setActivePane(result.current.panes[1].id);
      });
      expect(result.current.mobileActiveIndex).toBe(1);

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

      expect(result.current.mobileActiveIndex).toBe(2);

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

      act(() => {
        result.current.setActivePane(result.current.panes[0].id);
      });
      expect(result.current.mobileActiveIndex).toBe(0);

      act(() => {
        result.current.removePane(result.current.panes[2].id);
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.activePane).toBe(result.current.panes[0].id);
    });
  });
});
