"use client";

import { useState, useCallback } from "react";
import { useAsyncData } from "@/hooks";
import {
  RepositoryProviderData,
  GitCredentialData,
  RunnerLocalCredentialData,
  CredentialType,
} from "@/lib/api";
import { listRepositoryProviders, deleteRepositoryProvider, testRepositoryProviderConnection } from "@/lib/api/userRepositoryProvider";
import { listGitCredentials, deleteGitCredential, setDefaultGitCredential } from "@/lib/api/userGitCredential";

export interface GitSettingsData {
  providers: RepositoryProviderData[];
  credentials: GitCredentialData[];
  runnerLocal: RunnerLocalCredentialData | null;
  defaultCredentialId: number | null | "runner_local";
}

// Synthesize the virtual `runner_local` credential entry the legacy REST API
// always returned (user_git_credentials.go:67) — the Connect surface omits it
// and exposes the default toggle via runner_local_is_default instead.
function buildRunnerLocal(isDefault: boolean): RunnerLocalCredentialData {
  return {
    id: "runner_local",
    name: "Runner Local",
    credential_type: "runner_local",
    is_default: isDefault,
  };
}

export interface UseGitSettingsResult {
  // Data
  data: GitSettingsData | null;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;

  // Messages
  successMessage: string | null;
  errorMessage: string | null;
  setSuccessMessage: (msg: string | null) => void;
  setErrorMessage: (msg: string | null) => void;

  // Actions
  handleSetDefault: (credentialId: number | null) => Promise<void>;
  handleDeleteProvider: (id: number) => Promise<boolean>;
  handleDeleteCredential: (id: number) => Promise<boolean>;
  handleTestConnection: (id: number) => Promise<void>;
}

/**
 * Custom hook for Git settings data and operations
 *
 * Extracts all data fetching and mutation logic from GitSettingsContent
 */
export function useGitSettings(t: (key: string) => string): UseGitSettingsResult {
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Fetch data
  const fetcher = useCallback(async (): Promise<GitSettingsData> => {
    const [providersRes, credentialsRes] = await Promise.all([
      listRepositoryProviders(),
      listGitCredentials(),
    ]);

    const providers = providersRes.items;
    const credentials = credentialsRes.items;
    const runnerLocal = buildRunnerLocal(credentialsRes.runner_local_is_default);

    // Determine default credential
    let defaultCredentialId: number | null | "runner_local";
    if (runnerLocal.is_default) {
      defaultCredentialId = "runner_local";
    } else {
      const defaultCred = credentials.find((c: GitCredentialData) => c.is_default);
      defaultCredentialId = defaultCred?.id || "runner_local";
    }

    return {
      providers,
      credentials,
      runnerLocal,
      defaultCredentialId,
    };
  }, []);

  const {
    data,
    loading,
    error,
    refetch,
    setData,
  } = useAsyncData(fetcher, [], {
    onError: () => {
      setErrorMessage(t("settings.gitSettings.failedToLoad"));
    },
  });

  // Set default credential
  const handleSetDefault = useCallback(
    async (credentialId: number | null) => {
      try {
        setErrorMessage(null);
        await setDefaultGitCredential(credentialId === null ? undefined : credentialId);

        // Update local state
        setData((prev) =>
          prev
            ? {
                ...prev,
                defaultCredentialId: credentialId || "runner_local",
              }
            : null
        );

        setSuccessMessage(t("settings.gitSettings.defaultSet"));
        setTimeout(() => setSuccessMessage(null), 3000);
      } catch (err) {
        console.error("Failed to set default:", err);
        setErrorMessage(t("settings.gitSettings.failedToSetDefault"));
      }
    },
    [t, setData]
  );

  // Delete provider
  const handleDeleteProvider = useCallback(
    async (id: number): Promise<boolean> => {
      try {
        await deleteRepositoryProvider(id);
        await refetch();
        return true;
      } catch (err) {
        console.error("Failed to delete provider:", err);
        setErrorMessage(t("settings.gitSettings.failedToDeleteProvider"));
        return false;
      }
    },
    [t, refetch]
  );

  // Delete credential
  const handleDeleteCredential = useCallback(
    async (id: number): Promise<boolean> => {
      try {
        await deleteGitCredential(id);
        await refetch();
        return true;
      } catch (err) {
        console.error("Failed to delete credential:", err);
        setErrorMessage(t("settings.gitSettings.failedToDeleteCredential"));
        return false;
      }
    },
    [t, refetch]
  );

  // Test provider connection
  const handleTestConnection = useCallback(
    async (id: number) => {
      try {
        setErrorMessage(null);
        const res = await testRepositoryProviderConnection(id);
        if (!res.success) {
          throw new Error(res.message);
        }
        setSuccessMessage(t("settings.gitSettings.connectionSuccess"));
        setTimeout(() => {
          setSuccessMessage(null);
          setErrorMessage(null);
        }, 3000);
      } catch (err) {
        console.error("Failed to test connection:", err);
        setErrorMessage(t("settings.gitSettings.connectionFailed"));
      }
    },
    [t]
  );

  return {
    data,
    loading,
    error,
    refetch,
    successMessage,
    errorMessage,
    setSuccessMessage,
    setErrorMessage,
    handleSetDefault,
    handleDeleteProvider,
    handleDeleteCredential,
    handleTestConnection,
  };
}

/**
 * Get all selectable credentials for default picker
 */
export function getAllSelectableCredentials(data: GitSettingsData) {
  const items: Array<{
    id: number | "runner_local";
    name: string;
    type: string;
    isDefault: boolean;
  }> = [];

  // Add runner local first
  if (data.runnerLocal) {
    items.push({
      id: "runner_local",
      name: data.runnerLocal.name,
      type: CredentialType.RUNNER_LOCAL,
      isDefault: data.defaultCredentialId === "runner_local",
    });
  }

  // Add OAuth credentials from providers
  data.credentials
    .filter((c) => c.credential_type === CredentialType.OAUTH)
    .forEach((c) => {
      items.push({
        id: c.id,
        name: c.name,
        type: c.credential_type,
        isDefault: data.defaultCredentialId === c.id,
      });
    });

  // Add PAT and SSH credentials
  data.credentials
    .filter(
      (c) =>
        c.credential_type === CredentialType.PAT ||
        c.credential_type === CredentialType.SSH_KEY
    )
    .forEach((c) => {
      items.push({
        id: c.id,
        name: c.name,
        type: c.credential_type,
        isDefault: data.defaultCredentialId === c.id,
      });
    });

  return items;
}
