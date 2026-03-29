import { create } from "zustand";

export interface Repository {
  id: number;
  organization_id: number;
  git_provider_id: number;
  external_id: string;
  name: string;
  slug: string;
  default_branch: string;
  ticket_prefix?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  // Joined data
  git_provider_name?: string;
  git_provider_type?: string;
}

export interface Branch {
  name: string;
  commit_sha: string;
  is_default: boolean;
  is_protected: boolean;
}

interface RepositoryState {
  repositories: Repository[];
  currentRepository: Repository | null;
  branches: Branch[];
  isLoading: boolean;
  error: string | null;

  // Actions
  setRepositories: (repos: Repository[]) => void;
  setCurrentRepository: (repo: Repository | null) => void;
  addRepository: (repo: Repository) => void;
  updateRepository: (id: number, updates: Partial<Repository>) => void;
  removeRepository: (id: number) => void;
  setBranches: (branches: Branch[]) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;

  // Selectors
  getRepositoriesByProvider: (providerId: number) => Repository[];
}

const initialState = {
  repositories: [],
  currentRepository: null,
  branches: [],
  isLoading: false,
  error: null,
};

export const useRepositoryStore = create<RepositoryState>((set, get) => ({
  ...initialState,

  setRepositories: (repositories) => set({ repositories }),

  setCurrentRepository: (repo) => set({ currentRepository: repo }),

  addRepository: (repo) =>
    set((state) => ({
      repositories: [...state.repositories, repo],
    })),

  updateRepository: (id, updates) =>
    set((state) => ({
      repositories: state.repositories.map((r) =>
        r.id === id ? { ...r, ...updates } : r
      ),
      currentRepository:
        state.currentRepository?.id === id
          ? { ...state.currentRepository, ...updates }
          : state.currentRepository,
    })),

  removeRepository: (id) =>
    set((state) => ({
      repositories: state.repositories.filter((r) => r.id !== id),
      currentRepository:
        state.currentRepository?.id === id ? null : state.currentRepository,
    })),

  setBranches: (branches) => set({ branches }),

  setLoading: (isLoading) => set({ isLoading }),

  setError: (error) => set({ error }),

  reset: () => set(initialState),

  // Selectors
  getRepositoriesByProvider: (providerId) =>
    get().repositories.filter((r) => r.git_provider_id === providerId),
}));
