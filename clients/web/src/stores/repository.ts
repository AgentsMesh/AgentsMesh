import { create } from "zustand";
import { useMemo } from "react";
import { getRepoState, getRepositoryService } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import type { RepositoryData } from "@/lib/api/repositoryTypes";

// Single source of truth for the repository shape is `RepositoryData`
// in @/lib/api/repositoryTypes. The store re-exports it as `Repository`
// for ergonomic naming at call sites that already speak the domain.
export type Repository = RepositoryData;

interface RepositoryState {
  _tick: number;
  isLoading: boolean;
  error: string | null;
  fetchRepositories: () => Promise<void>;
  deleteRepository: (id: number) => Promise<void>;
}

const rs = () => getRepoState();
const bump = () => useRepositoryStore.setState((s) => ({ _tick: s._tick + 1 }));

export function useRepositories(): Repository[] {
  const tick = useRepositoryStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => JSON.parse(rs().repositories_json()), [tick]);
}

export const useRepositoryStore = create<RepositoryState>((set) => ({
  _tick: 0,
  isLoading: false,
  error: null,

  fetchRepositories: async () => {
    set({ isLoading: true, error: null });
    try {
      const raw = await getRepositoryService().list();
      const parsed = JSON.parse(raw) as { repositories?: Repository[] };
      rs().set_repositories(JSON.stringify(parsed.repositories ?? []));
      bump();
      set({ isLoading: false });
    } catch (e) {
      set({ isLoading: false, error: getErrorMessage(e, "Failed to fetch repositories") });
    }
  },

  deleteRepository: async (id: number) => {
    await getRepositoryService().delete(BigInt(id));
    rs().remove_repository(String(id));
    bump();
  },
}));
