import { create } from "zustand";
import { persist } from "zustand/middleware";

interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string;
}

interface Organization {
  id: number;
  name: string;
  slug: string;
  role?: string;
  logo_url?: string;
  subscription_plan?: string;
  subscription_status?: string;
  created_at?: string;
  updated_at?: string;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  currentOrg: Organization | null;
  organizations: Organization[];
  _hasHydrated: boolean;

  // Actions
  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  setOrganizations: (orgs: Organization[]) => void;
  setCurrentOrg: (org: Organization) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      refreshToken: null,
      user: null,
      currentOrg: null,
      organizations: [],
      _hasHydrated: false,

      setAuth: (token, user, refreshToken) => set({ token, user, refreshToken: refreshToken || null }),

      setTokens: (token, refreshToken) => set({ token, refreshToken }),

      setOrganizations: (organizations) => {
        set({ organizations });
        // Auto-select first org if none selected
        if (!get().currentOrg && organizations.length > 0) {
          set({ currentOrg: organizations[0] });
        }
      },

      setCurrentOrg: (org) => set({ currentOrg: org }),

      logout: () =>
        set({
          token: null,
          refreshToken: null,
          user: null,
          currentOrg: null,
          organizations: [],
        }),

      isAuthenticated: () => !!get().token,

      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: "agentsmesh-auth",
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
        currentOrg: state.currentOrg,
        organizations: state.organizations,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
