import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * Activity types for the IDE sidebar
 */
export type ActivityType =
  | "workspace"
  | "tickets"
  | "channels"
  | "mesh"
  | "loops"
  | "repositories"
  | "runners"
  | "settings";

/**
 * Bottom panel tab types
 */
export type BottomPanelTab = "channels" | "activity" | "autopilot" | "delivery" | "info";

/**
 * IDE UI state management
 */
interface IDEState {
  // Active activity in the activity bar
  activeActivity: ActivityType;
  setActiveActivity: (activity: ActivityType) => void;

  // Side bar state
  sidebarOpen: boolean;
  sidebarWidth: number;
  setSidebarOpen: (open: boolean) => void;
  setSidebarWidth: (width: number) => void;
  toggleSidebar: () => void;

  // Bottom panel state
  bottomPanelOpen: boolean;
  bottomPanelHeight: number;
  bottomPanelTab: BottomPanelTab;
  setBottomPanelOpen: (open: boolean) => void;
  setBottomPanelHeight: (height: number) => void;
  setBottomPanelTab: (tab: BottomPanelTab) => void;
  toggleBottomPanel: () => void;

  // Mobile specific state
  mobileDrawerOpen: boolean;
  mobileMoreMenuOpen: boolean;
  mobileSidebarOpen: boolean;
  setMobileDrawerOpen: (open: boolean) => void;
  setMobileMoreMenuOpen: (open: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;

  // Hydration state for SSR
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useIDEStore = create<IDEState>()(
  persist(
    (set) => ({
      // Activity bar
      activeActivity: "workspace",
      setActiveActivity: (activity) => set({ activeActivity: activity }),

      // Side bar - default values
      sidebarOpen: true,
      sidebarWidth: 280,
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSidebarWidth: (width) => set({ sidebarWidth: width }),
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      // Bottom panel - default values
      bottomPanelOpen: false,
      bottomPanelHeight: 200,
      bottomPanelTab: "channels",
      setBottomPanelOpen: (open) => set({ bottomPanelOpen: open }),
      setBottomPanelHeight: (height) => set({ bottomPanelHeight: height }),
      setBottomPanelTab: (tab) => set({ bottomPanelTab: tab }),
      toggleBottomPanel: () =>
        set((state) => ({ bottomPanelOpen: !state.bottomPanelOpen })),

      // Mobile specific
      mobileDrawerOpen: false,
      mobileMoreMenuOpen: false,
      mobileSidebarOpen: false,
      setMobileDrawerOpen: (open) => set({ mobileDrawerOpen: open }),
      setMobileMoreMenuOpen: (open) => set({ mobileMoreMenuOpen: open }),
      setMobileSidebarOpen: (open) => set({ mobileSidebarOpen: open }),

      // Hydration
      _hasHydrated: false,
      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: "agentsmesh-ide",
      partialize: (state) => ({
        // Only persist these fields
        activeActivity: state.activeActivity,
        sidebarOpen: state.sidebarOpen,
        sidebarWidth: state.sidebarWidth,
        bottomPanelOpen: state.bottomPanelOpen,
        bottomPanelHeight: state.bottomPanelHeight,
        bottomPanelTab: state.bottomPanelTab,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);

/**
 * Activity configuration
 */
export interface ActivityConfig {
  id: ActivityType;
  label: string;
  icon: string;
  mobileVisible: boolean; // Show in mobile bottom tab bar
  mobileOrder?: number; // Order in mobile tab bar (lower = first)
}

export const ACTIVITIES: ActivityConfig[] = [
  {
    id: "workspace",
    label: "Workspace",
    icon: "terminal",
    mobileVisible: true,
    mobileOrder: 2,
  },
  {
    id: "tickets",
    label: "Tickets",
    icon: "ticket",
    mobileVisible: true,
    mobileOrder: 3,
  },
  {
    id: "channels",
    label: "Channels",
    icon: "message-square",
    mobileVisible: true,
    mobileOrder: 1,
  },
  {
    id: "mesh",
    label: "Mesh",
    icon: "network",
    mobileVisible: true,
    mobileOrder: 4,
  },
  {
    id: "loops",
    label: "Loops",
    icon: "repeat",
    mobileVisible: false,
  },
  {
    id: "repositories",
    label: "Repositories",
    icon: "repository",
    mobileVisible: false, // In "More" menu on mobile
  },
  {
    id: "runners",
    label: "Runners",
    icon: "server",
    mobileVisible: false, // In "More" menu on mobile
  },
  {
    id: "settings",
    label: "Settings",
    icon: "settings",
    mobileVisible: false, // In "More" menu on mobile
  },
];

/**
 * Get mobile visible activities sorted by order
 */
export function getMobileActivities(): ActivityConfig[] {
  return ACTIVITIES.filter((a) => a.mobileVisible).sort(
    (a, b) => (a.mobileOrder ?? 99) - (b.mobileOrder ?? 99)
  );
}

/**
 * Get activities for "More" menu on mobile
 */
export function getMoreMenuActivities(): ActivityConfig[] {
  return ACTIVITIES.filter((a) => !a.mobileVisible);
}
