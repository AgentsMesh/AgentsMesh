import { describe, it, expect, beforeEach, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import {
  useIDEStore,
  ACTIVITIES,
  getMobileActivities,
  getMoreMenuActivities,
} from "../ide";

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();
Object.defineProperty(window, "localStorage", { value: localStorageMock });

describe("IDE Store", () => {
  beforeEach(() => {
    localStorageMock.clear();
    // Reset store to initial state
    useIDEStore.setState({
      activeActivity: "workspace",
      sidebarOpen: true,
      sidebarWidth: 280,
      bottomPanelOpen: false,
      bottomPanelHeight: 200,
      bottomPanelTab: "output",
      mobileDrawerOpen: false,
      mobileMoreMenuOpen: false,
      _hasHydrated: false,
    });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const { result } = renderHook(() => useIDEStore());

      expect(result.current.activeActivity).toBe("workspace");
      expect(result.current.sidebarOpen).toBe(true);
      expect(result.current.sidebarWidth).toBe(280);
      expect(result.current.bottomPanelOpen).toBe(false);
      expect(result.current.bottomPanelHeight).toBe(200);
      expect(result.current.bottomPanelTab).toBe("output");
      expect(result.current.mobileDrawerOpen).toBe(false);
      expect(result.current.mobileMoreMenuOpen).toBe(false);
    });
  });

  describe("activity bar", () => {
    it("should set active activity", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setActiveActivity("tickets");
      });

      expect(result.current.activeActivity).toBe("tickets");
    });

    it("should support all activity types", () => {
      const { result } = renderHook(() => useIDEStore());
      const activities = [
        "workspace",
        "tickets",
        "mesh",
        "repositories",
        "runners",
        "settings",
      ] as const;

      activities.forEach((activity) => {
        act(() => {
          result.current.setActiveActivity(activity);
        });
        expect(result.current.activeActivity).toBe(activity);
      });
    });
  });

  describe("sidebar", () => {
    it("should set sidebar open state", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setSidebarOpen(false);
      });

      expect(result.current.sidebarOpen).toBe(false);
    });

    it("should set sidebar width", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setSidebarWidth(350);
      });

      expect(result.current.sidebarWidth).toBe(350);
    });

    it("should toggle sidebar", () => {
      const { result } = renderHook(() => useIDEStore());

      expect(result.current.sidebarOpen).toBe(true);

      act(() => {
        result.current.toggleSidebar();
      });

      expect(result.current.sidebarOpen).toBe(false);

      act(() => {
        result.current.toggleSidebar();
      });

      expect(result.current.sidebarOpen).toBe(true);
    });
  });

  describe("bottom panel", () => {
    it("should set bottom panel open state", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setBottomPanelOpen(true);
      });

      expect(result.current.bottomPanelOpen).toBe(true);
    });

    it("should set bottom panel height", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setBottomPanelHeight(300);
      });

      expect(result.current.bottomPanelHeight).toBe(300);
    });

    it("should set bottom panel tab", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setBottomPanelTab("channels");
      });

      expect(result.current.bottomPanelTab).toBe("channels");
    });

    it("should toggle bottom panel", () => {
      const { result } = renderHook(() => useIDEStore());

      expect(result.current.bottomPanelOpen).toBe(false);

      act(() => {
        result.current.toggleBottomPanel();
      });

      expect(result.current.bottomPanelOpen).toBe(true);

      act(() => {
        result.current.toggleBottomPanel();
      });

      expect(result.current.bottomPanelOpen).toBe(false);
    });
  });

  describe("mobile state", () => {
    it("should set mobile drawer open state", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setMobileDrawerOpen(true);
      });

      expect(result.current.mobileDrawerOpen).toBe(true);
    });

    it("should set mobile more menu open state", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setMobileMoreMenuOpen(true);
      });

      expect(result.current.mobileMoreMenuOpen).toBe(true);
    });
  });

  describe("hydration", () => {
    it("should set hydration state", () => {
      const { result } = renderHook(() => useIDEStore());

      act(() => {
        result.current.setHasHydrated(true);
      });

      expect(result.current._hasHydrated).toBe(true);
    });
  });
});

describe("Activity Configuration", () => {
  describe("ACTIVITIES constant", () => {
    it("should have all required activities", () => {
      const ids = ACTIVITIES.map((a) => a.id);
      expect(ids).toContain("workspace");
      expect(ids).toContain("tickets");
      expect(ids).toContain("mesh");
      expect(ids).toContain("repositories");
      expect(ids).toContain("runners");
      expect(ids).toContain("settings");
    });

    it("should have labels for all activities", () => {
      ACTIVITIES.forEach((activity) => {
        expect(activity.label).toBeTruthy();
        expect(typeof activity.label).toBe("string");
      });
    });

    it("should have icons for all activities", () => {
      ACTIVITIES.forEach((activity) => {
        expect(activity.icon).toBeTruthy();
        expect(typeof activity.icon).toBe("string");
      });
    });
  });

  describe("getMobileActivities", () => {
    it("should return only mobile visible activities", () => {
      const mobileActivities = getMobileActivities();
      mobileActivities.forEach((activity) => {
        expect(activity.mobileVisible).toBe(true);
      });
    });

    it("should be sorted by mobile order", () => {
      const mobileActivities = getMobileActivities();
      for (let i = 1; i < mobileActivities.length; i++) {
        const prevOrder = mobileActivities[i - 1].mobileOrder ?? 99;
        const currOrder = mobileActivities[i].mobileOrder ?? 99;
        expect(prevOrder).toBeLessThanOrEqual(currOrder);
      }
    });

    it("should not include settings or runners", () => {
      const mobileActivities = getMobileActivities();
      const ids = mobileActivities.map((a) => a.id);
      expect(ids).not.toContain("settings");
      expect(ids).not.toContain("runners");
    });
  });

  describe("getMoreMenuActivities", () => {
    it("should return non-mobile visible activities", () => {
      const moreActivities = getMoreMenuActivities();
      moreActivities.forEach((activity) => {
        expect(activity.mobileVisible).toBe(false);
      });
    });

    it("should include settings and runners", () => {
      const moreActivities = getMoreMenuActivities();
      const ids = moreActivities.map((a) => a.id);
      expect(ids).toContain("settings");
      expect(ids).toContain("runners");
    });
  });
});
