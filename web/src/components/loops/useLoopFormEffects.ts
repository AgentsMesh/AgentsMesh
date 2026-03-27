"use client";

import { useState, useEffect } from "react";
import { CredentialProfileData, userAgentCredentialApi } from "@/lib/api";
import type { ConfigField } from "@/lib/api";
import type { RepositoryData, AgentData } from "@/lib/api";
import type { LoopData } from "@/lib/api/loop";
import { RUNNER_HOST_PROFILE_ID } from "./types";

/**
 * Manages credential profile loading for a selected agent.
 * In edit mode, preserves the editLoop's credential_profile_id on first load.
 */
export function useCredentialLoader(
  open: boolean,
  selectedAgentSlug: string | null,
  editLoop?: LoopData,
) {
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [loadingCredentials, setLoadingCredentials] = useState(false);
  const [selectedCredentialProfileId, setSelectedCredentialProfileId] = useState<number>(
    editLoop?.credential_profile_id || RUNNER_HOST_PROFILE_ID
  );
  const [credentialInitialized, setCredentialInitialized] = useState(false);

  useEffect(() => {
    // Reset initialization flag when dialog re-opens
    if (!open) {
      setCredentialInitialized(false);
      return;
    }
  }, [open]);

  useEffect(() => {
    if (!selectedAgentSlug) {
      // eslint-disable-next-line react-compiler/react-compiler -- batch state reset on agent deselection
      setCredentialProfiles([]);
      setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
      setCredentialInitialized(false);
      return;
    }

    const loadCredentials = async () => {
      setLoadingCredentials(true);
      try {
        const res = await userAgentCredentialApi.listForAgent(selectedAgentSlug);
        const profiles = res.profiles || [];
        setCredentialProfiles(profiles);

        // In edit mode, preserve editLoop's credential on initial load
        if (editLoop?.credential_profile_id && !credentialInitialized) {
          setSelectedCredentialProfileId(editLoop.credential_profile_id);
          setCredentialInitialized(true);
        } else {
          const defaultProfile = profiles.find((p) => p.is_default);
          if (defaultProfile) {
            setSelectedCredentialProfileId(defaultProfile.id);
          } else {
            setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
          }
        }
      } catch {
        setCredentialProfiles([]);
        setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
      } finally {
        setLoadingCredentials(false);
      }
    };

    loadCredentials();
  }, [selectedAgentSlug, editLoop, credentialInitialized]);

  return {
    credentialProfiles,
    loadingCredentials,
    selectedCredentialProfileId,
    setSelectedCredentialProfileId,
  };
}

/**
 * Restores config_overrides from editLoop once config fields have loaded.
 */
export function useConfigRestore(
  open: boolean,
  configFields: ConfigField[],
  handleConfigChange: (key: string, value: unknown) => void,
  editLoop?: LoopData,
) {
  const [configOverridesRestored, setConfigOverridesRestored] = useState(false);
  useEffect(() => {
    if (!open) {
      // eslint-disable-next-line react-compiler/react-compiler -- reset flag on dialog close
      setConfigOverridesRestored(false);
      return;
    }
    if (editLoop?.config_overrides && configFields.length > 0 && !configOverridesRestored) {
      Object.entries(editLoop.config_overrides).forEach(([key, value]) => {
        handleConfigChange(key, value);
      });
      setConfigOverridesRestored(true);
    }
  }, [open, editLoop, configFields, configOverridesRestored, handleConfigChange]);
}

/**
 * Auto-fills branch when repository changes.
 */
export function useBranchAutoFill(
  selectedRepositoryId: number | null,
  repositories: RepositoryData[],
  setSelectedBranch: (branch: string) => void,
) {
  useEffect(() => {
    if (!selectedRepositoryId) {
      setSelectedBranch("");
      return;
    }
    const repo = repositories.find((r) => r.id === selectedRepositoryId);
    if (repo?.default_branch) {
      setSelectedBranch(repo.default_branch);
    }
  }, [selectedRepositoryId, repositories, setSelectedBranch]);
}

/**
 * Resets agent selection if the current agent is not available in the runner's agents.
 */
export function useAgentReset(
  availableAgents: AgentData[],
  selectedAgentSlug: string | null,
  setSelectedAgentSlug: (slug: string | null) => void,
) {
  useEffect(() => {
    if (availableAgents.length > 0 && selectedAgentSlug && !availableAgents.find((a) => a.slug === selectedAgentSlug)) {
      setSelectedAgentSlug(null);
    }
  }, [availableAgents, selectedAgentSlug, setSelectedAgentSlug]);
}
