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
  | "blocks"
  | "infra"
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
  /** Visual grouping in the desktop ActivityBar — items in the same group render
   *  contiguously; a divider separates groups. */
  group: "comm" | "build" | "ops" | "system";
  mobileVisible: boolean; // Show in mobile bottom tab bar
  mobileOrder?: number; // Order in mobile tab bar (lower = first)
}

export const ACTIVITIES: ActivityConfig[] = [
  // ─── Group 1 · Communicate (channels / blocks / mesh) ───
  {
    id: "channels",
    label: "Channels",
    icon: "message-square",
    group: "comm",
    mobileVisible: true,
    mobileOrder: 1,
  },
  {
    id: "blocks",
    label: "Blocks",
    icon: "blocks",
    group: "comm",
    mobileVisible: false,
  },
  {
    id: "mesh",
    label: "Mesh",
    icon: "network",
    group: "comm",
    mobileVisible: true,
    mobileOrder: 2,
  },
  // ─── Group 2 · Build (workspace / tickets / loops) ───
  {
    id: "workspace",
    label: "Workspace",
    icon: "terminal",
    group: "build",
    mobileVisible: true,
    mobileOrder: 3,
  },
  {
    id: "tickets",
    label: "Tickets",
    icon: "ticket",
    group: "build",
    mobileVisible: true,
    mobileOrder: 4,
  },
  {
    id: "loops",
    label: "Loops",
    icon: "repeat",
    group: "build",
    mobileVisible: false,
  },
  // ─── Group 3 · Operate (infra) ───
  {
    id: "infra",
    label: "Infra",
    icon: "layers",
    group: "ops",
    mobileVisible: false,
  },
  // ─── Bottom · system ───
  {
    id: "settings",
    label: "Settings",
    icon: "settings",
    group: "system",
    mobileVisible: false,
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
