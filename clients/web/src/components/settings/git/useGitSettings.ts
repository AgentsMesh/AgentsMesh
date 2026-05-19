"use client";

import { useState, useCallback } from "react";
import { useAsyncData } from "@/hooks";
import {
  RepositoryProviderData,
  GitCredentialData,
  RunnerLocalCredentialData,
  CredentialType,
} from "@/lib/api";
import { getUserCredentialService } from "@/lib/wasm-core";

export interface GitSettingsData {
  providers: RepositoryProviderData[];
  credentials: GitCredentialData[];
  runnerLocal: RunnerLocalCredentialData | null;
  defaultCredentialId: number | null | "runner_local";
}

export interface UseGitSettingsResult {
  data: GitSettingsData | null;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;

  successMessage: string | null;
  errorMessage: string | null;
  setSuccessMessage: (msg: string | null) => void;
  setErrorMessage: (msg: string | null) => void;

  handleSetDefault: (credentialId: number | null) => Promise<void>;
  handleDeleteProvider: (id: number) => Promise<boolean>;
  handleDeleteCredential: (id: number) => Promise<boolean>;
  handleTestConnection: (id: number) => Promise<void>;
}

export function useGitSettings(t: (key: string) => string): UseGitSettingsResult {
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const fetcher = useCallback(async (): Promise<GitSettingsData> => {
    const [providersRes, credentialsRes] = await Promise.all([
      getUserCredentialService().list_repo_providers().then((j: string) => JSON.parse(j)),
      getUserCredentialService().list_git_credentials().then((j: string) => JSON.parse(j)),
    ]);

    const providers = providersRes.providers || [];
    const credentials = credentialsRes.credentials || [];
    const runnerLocal = credentialsRes.runner_local;

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

  const handleSetDefault = useCallback(
    async (credentialId: number | null) => {
      try {
        setErrorMessage(null);
        await getUserCredentialService().set_default_git_credential(
          JSON.stringify({ credential_id: credentialId })
        );

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

  const handleDeleteProvider = useCallback(
    async (id: number): Promise<boolean> => {
      try {
        await getUserCredentialService().delete_repo_provider(BigInt(id));
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

  const handleDeleteCredential = useCallback(
    async (id: number): Promise<boolean> => {
      try {
        await getUserCredentialService().delete_git_credential(BigInt(id));
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

  const handleTestConnection = useCallback(
    async (id: number) => {
      try {
        setErrorMessage(null);
        await getUserCredentialService().test_repo_provider(BigInt(id));
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

export function getAllSelectableCredentials(data: GitSettingsData) {
  const items: Array<{
    id: number | "runner_local";
    name: string;
    type: string;
    isDefault: boolean;
  }> = [];

  if (data.runnerLocal) {
    items.push({
      id: "runner_local",
      name: data.runnerLocal.name,
      type: CredentialType.RUNNER_LOCAL,
      isDefault: data.defaultCredentialId === "runner_local",
    });
  }

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
