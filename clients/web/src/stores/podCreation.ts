import { create } from "zustand";
import { persist } from "zustand/middleware";

interface PodCreationPreferences {
  lastAgentSlug: string | null;
  lastRepositoryId: number | null;
  lastCredentialProfileId: number | null;
  lastBranchName: string | null;

  setLastChoices: (
    choices: Partial<
      Pick<
        PodCreationPreferences,
        "lastAgentSlug" | "lastRepositoryId" | "lastCredentialProfileId" | "lastBranchName"
      >
    >
  ) => void;
  clearLastChoices: () => void;

  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

export const usePodCreationStore = create<PodCreationPreferences>()(
  persist(
    (set) => ({
      lastAgentSlug: null,
      lastRepositoryId: null,
      lastCredentialProfileId: null,
      lastBranchName: null,

      setLastChoices: (choices) => set((state) => ({ ...state, ...choices })),
      clearLastChoices: () =>
        set({
          lastAgentSlug: null,
          lastRepositoryId: null,
          lastCredentialProfileId: null,
          lastBranchName: null,
        }),

      _hasHydrated: false,
      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: "agentsmesh-pod-creation",
      partialize: (state) => ({
        lastAgentSlug: state.lastAgentSlug,
        lastRepositoryId: state.lastRepositoryId,
        lastCredentialProfileId: state.lastCredentialProfileId,
        lastBranchName: state.lastBranchName,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
