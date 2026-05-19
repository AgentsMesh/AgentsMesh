import { create } from "zustand";
import { persist } from "zustand/middleware";

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

export type BottomPanelTab = "channels" | "activity" | "autopilot" | "delivery" | "info";

interface IDEState {
  activeActivity: ActivityType;
  setActiveActivity: (activity: ActivityType) => void;

  sidebarOpen: boolean;
  sidebarWidth: number;
  setSidebarOpen: (open: boolean) => void;
  setSidebarWidth: (width: number) => void;
  toggleSidebar: () => void;

  bottomPanelOpen: boolean;
  bottomPanelHeight: number;
  bottomPanelTab: BottomPanelTab;
  setBottomPanelOpen: (open: boolean) => void;
  setBottomPanelHeight: (height: number) => void;
  setBottomPanelTab: (tab: BottomPanelTab) => void;
  toggleBottomPanel: () => void;

  mobileDrawerOpen: boolean;
  mobileMoreMenuOpen: boolean;
  mobileSidebarOpen: boolean;
  setMobileDrawerOpen: (open: boolean) => void;
  setMobileMoreMenuOpen: (open: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;

  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useIDEStore = create<IDEState>()(
  persist(
    (set) => ({
      activeActivity: "workspace",
      setActiveActivity: (activity) => set({ activeActivity: activity }),

      sidebarOpen: true,
      sidebarWidth: 280,
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSidebarWidth: (width) => set({ sidebarWidth: width }),
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      bottomPanelOpen: false,
      bottomPanelHeight: 200,
      bottomPanelTab: "channels",
      setBottomPanelOpen: (open) => set({ bottomPanelOpen: open }),
      setBottomPanelHeight: (height) => set({ bottomPanelHeight: height }),
      setBottomPanelTab: (tab) => set({ bottomPanelTab: tab }),
      toggleBottomPanel: () =>
        set((state) => ({ bottomPanelOpen: !state.bottomPanelOpen })),

      mobileDrawerOpen: false,
      mobileMoreMenuOpen: false,
      mobileSidebarOpen: false,
      setMobileDrawerOpen: (open) => set({ mobileDrawerOpen: open }),
      setMobileMoreMenuOpen: (open) => set({ mobileMoreMenuOpen: open }),
      setMobileSidebarOpen: (open) => set({ mobileSidebarOpen: open }),

      _hasHydrated: false,
      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: "agentsmesh-ide",
      partialize: (state) => ({
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

export interface ActivityConfig {
  id: ActivityType;
  label: string;
  icon: string;
  group: "comm" | "build" | "ops" | "system";
  mobileVisible: boolean;
  mobileOrder?: number;
}

export const ACTIVITIES: ActivityConfig[] = [
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
  {
    id: "infra",
    label: "Infra",
    icon: "layers",
    group: "ops",
    mobileVisible: false,
  },
  {
    id: "settings",
    label: "Settings",
    icon: "settings",
    group: "system",
    mobileVisible: false,
  },
];

export function getMobileActivities(): ActivityConfig[] {
  return ACTIVITIES.filter((a) => a.mobileVisible).sort(
    (a, b) => (a.mobileOrder ?? 99) - (b.mobileOrder ?? 99)
  );
}

export function getMoreMenuActivities(): ActivityConfig[] {
  return ACTIVITIES.filter((a) => !a.mobileVisible);
}
