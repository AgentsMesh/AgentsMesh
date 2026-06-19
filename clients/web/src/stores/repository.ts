import { create } from "zustand";
import { useMemo } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import { getRepoState } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import { repositoryApi } from "@/lib/api/facade/repository";
import type { RepositoryData } from "@/lib/viewModels/repository";
import {
  ReplaceCachedRepositoriesRequestSchema,
} from "@proto/repo_state/v1/repo_state_pb";
import { repositoryToCache } from "@agentsmesh/electron-adapter/projections";

export type Repository = RepositoryData;

interface RepositoryState {
  _tick: number;
  isLoading: boolean;
  fetched: boolean;
  error: string | null;
  fetchRepositories: () => Promise<void>;
  deleteRepository: (id: number) => Promise<void>;
}

const rs = () => getRepoState();
const bump = () => useRepositoryStore.setState((s) => ({ _tick: s._tick + 1 }));

// Read side (B, zero-JSON): UI is a projection of state proto bytes
// (repositories_bytes) decoded via fromBinary + repositoryToCache (shared projection).
export function useRepositories(): Repository[] {
  const tick = useRepositoryStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(
    () => fromBinary(ReplaceCachedRepositoriesRequestSchema, rs().repositories_bytes()).repositories.map(repositoryToCache),
    [tick],
  );
}

export const useRepositoryStore = create<RepositoryState>((set) => ({
  _tick: 0,
  isLoading: false,
  fetched: false,
  error: null,

  fetchRepositories: async () => {
    set({ isLoading: true, error: null });
    try {
      const respBytes = await repositoryApi.listRaw();
      rs().apply_fetched_repositories(respBytes);
      bump();
      set({ isLoading: false, fetched: true });
    } catch (e) {
      set({ isLoading: false, error: getErrorMessage(e, "Failed to fetch repositories") });
    }
  },

  deleteRepository: async (id: number) => {
    await repositoryApi.delete(id);
    rs().remove_repository(String(id));
    bump();
  },
}));
