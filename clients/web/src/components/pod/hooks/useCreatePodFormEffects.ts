import { useState, useEffect, useRef } from "react";
import type { AgentData, RepositoryData, CredentialProfileData } from "@/lib/api";
import { getUserCredentialService } from "@/lib/wasm-core";
import { usePodCreationStore } from "@/stores/podCreation";
import { RUNNER_HOST_PROFILE_ID } from "./useCreatePodFormTypes";

/**
 * Hook managing auto-fill from saved preferences when agents/repos become available.
 * Returns a ref that tracks whether preferences have been initialized.
 *
 * `overrides` lets callers declare explicit context (e.g. ticket repository) that
 * must beat saved prefs. When a field is provided here, the matching pref branch
 * is skipped so the explicit value can be applied by its own initializer without
 * a race against this effect.
 */
export function usePrefsAutoFill(
  availableAgents: AgentData[],
  repositories: RepositoryData[],
  setSelectedAgent: (slug: string | null) => void,
  setSelectedRepository: (id: number | null) => void,
  setSelectedBranch: (branch: string) => void,
  overrides?: { repositoryId?: number | null },
) {
  const { lastAgentSlug, lastRepositoryId, lastBranchName } = usePodCreationStore();
  const prefsInitializedRef = useRef(false);
  const overrideRepositoryId = overrides?.repositoryId ?? null;

  useEffect(() => {
    if (prefsInitializedRef.current || availableAgents.length === 0) return;

    if (lastAgentSlug && availableAgents.find(a => a.slug === lastAgentSlug)) {
      setSelectedAgent(lastAgentSlug);
    }
    if (
      !overrideRepositoryId &&
      lastRepositoryId &&
      repositories.find(r => r.id === lastRepositoryId)
    ) {
      setSelectedRepository(lastRepositoryId);
    }
    if (lastBranchName) {
      setSelectedBranch(lastBranchName);
    }

    prefsInitializedRef.current = true;
  }, [availableAgents, repositories, lastAgentSlug, lastRepositoryId, lastBranchName, overrideRepositoryId, setSelectedAgent, setSelectedRepository, setSelectedBranch]);

  return prefsInitializedRef;
}

/**
 * Hook that loads credential profiles when the selected agent changes.
 * Auto-selects the saved preference, default, or RunnerHost profile.
 */
export function useCredentialProfiles(selectedAgent: string | null) {
  const { lastCredentialProfileId } = usePodCreationStore();
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [loadingCredentials, setLoadingCredentials] = useState(false);
  const [selectedCredentialProfile, setSelectedCredentialProfile] = useState<number>(RUNNER_HOST_PROFILE_ID);

  useEffect(() => {
    if (!selectedAgent) {
      setCredentialProfiles([]);
      setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
      return;
    }

    const loadCredentials = async () => {
      setLoadingCredentials(true);
      try {
        const res = JSON.parse(
          await getUserCredentialService().list_agent_credentials_for_agent(selectedAgent)
        );
        const profiles = res.profiles || [];
        setCredentialProfiles(profiles);

        // Auto-select: prefer saved preference, then default profile, then RunnerHost
        const savedProfile = lastCredentialProfileId && profiles.find((p: CredentialProfileData) => p.id === lastCredentialProfileId);
        const defaultProfile = profiles.find((p: CredentialProfileData) => p.is_default);
        if (savedProfile) {
          setSelectedCredentialProfile(savedProfile.id);
        } else if (defaultProfile) {
          setSelectedCredentialProfile(defaultProfile.id);
        } else {
          setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
        }
      } catch (err) {
        console.error("Failed to load credential profiles:", err);
        setCredentialProfiles([]);
        setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
      } finally {
        setLoadingCredentials(false);
      }
    };

    loadCredentials();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedAgent]);

  return {
    credentialProfiles,
    setCredentialProfiles,
    loadingCredentials,
    selectedCredentialProfile,
    setSelectedCredentialProfile,
  };
}
