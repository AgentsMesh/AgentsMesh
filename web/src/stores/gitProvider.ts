import { create } from "zustand";

export interface GitProvider {
  id: number;
  organization_id: number;
  provider_type: "gitlab" | "github" | "gitee";
  name: string;
  base_url: string;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface GitProviderProject {
  id: string;
  name: string;
  slug: string;
  default_branch: string;
  web_url: string;
  description?: string;
}

interface GitProviderState {
  providers: GitProvider[];
  currentProvider: GitProvider | null;
  availableProjects: GitProviderProject[];
  isLoading: boolean;
  isSyncing: boolean;
  error: string | null;

  // Actions
  setProviders: (providers: GitProvider[]) => void;
  setCurrentProvider: (provider: GitProvider | null) => void;
  addProvider: (provider: GitProvider) => void;
  updateProvider: (id: number, updates: Partial<GitProvider>) => void;
  removeProvider: (id: number) => void;
  setAvailableProjects: (projects: GitProviderProject[]) => void;
  setLoading: (loading: boolean) => void;
  setSyncing: (syncing: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState = {
  providers: [],
  currentProvider: null,
  availableProjects: [],
  isLoading: false,
  isSyncing: false,
  error: null,
};

export const useGitProviderStore = create<GitProviderState>((set) => ({
  ...initialState,

  setProviders: (providers) => set({ providers }),

  setCurrentProvider: (provider) => set({ currentProvider: provider }),

  addProvider: (provider) =>
    set((state) => ({
      providers: [...state.providers, provider],
    })),

  updateProvider: (id, updates) =>
    set((state) => ({
      providers: state.providers.map((p) =>
        p.id === id ? { ...p, ...updates } : p
      ),
      currentProvider:
        state.currentProvider?.id === id
          ? { ...state.currentProvider, ...updates }
          : state.currentProvider,
    })),

  removeProvider: (id) =>
    set((state) => ({
      providers: state.providers.filter((p) => p.id !== id),
      currentProvider:
        state.currentProvider?.id === id ? null : state.currentProvider,
    })),

  setAvailableProjects: (availableProjects) => set({ availableProjects }),

  setLoading: (isLoading) => set({ isLoading }),

  setSyncing: (isSyncing) => set({ isSyncing }),

  setError: (error) => set({ error }),

  reset: () => set(initialState),
}));
