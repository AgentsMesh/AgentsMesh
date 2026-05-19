"use client";

import { useState, useEffect, useCallback } from "react";
import {
  type ConfigField,
  type CredentialField,
  type AgentData,
  type CredentialProfileData,
} from "@/lib/api";
import { getAgentService, getUserCredentialService } from "@/lib/wasm-core";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import type { AgentConfigState, AgentConfigActions, CredentialFormData } from "./types";

export function useAgentConfig(
  agentSlug: string,
  t: (key: string) => string
): AgentConfigState & AgentConfigActions {
  const [loading, setLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);

  const [agent, setAgent] = useState<AgentData | null>(null);
  const [configFields, setConfigFields] = useState<ConfigField[]>([]);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [credentialFields, setCredentialFields] = useState<CredentialField[]>([]);
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [isRunnerHostDefault, setIsRunnerHostDefault] = useState(true);

  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Load agents to find the one matching the slug
      const agentsRes = JSON.parse(await getAgentService().list_agents());
      const allAgents = [...(agentsRes.builtin_agents || []), ...(agentsRes.custom_agents || []), ...(agentsRes.agents || [])];
      const foundAgent = allAgents.find(
        (a: AgentData) => a.slug === agentSlug
      );

      if (!foundAgent) {
        setError(t("settings.agentConfig.agentNotFound"));
        setLoading(false);
        return;
      }

      setAgent(foundAgent);

      const [schemaRes, credentialsRes] = await Promise.all([
        getAgentService().get_config_schema(foundAgent.slug).then((j: string) => JSON.parse(j)).catch(() => ({ fields: [], credential_fields: [] })),
        getUserCredentialService().list_agent_credentials().then((j: string) => JSON.parse(j)).catch(() => ({ items: [] })),
      ]);

      const fields = schemaRes.fields || [];
      setConfigFields(fields);

      setCredentialFields(schemaRes.credential_fields || []);

      const defaultValues: Record<string, unknown> = {};
      for (const field of fields) {
        if (field.default !== undefined) {
          defaultValues[field.name] = field.default;
        }
      }

      try {
        const userConfigRes = JSON.parse(await getAgentService().get_user_config(foundAgent.slug));
        if (userConfigRes.config?.config_values) {
          setConfigValues({ ...defaultValues, ...userConfigRes.config.config_values });
        } else {
          setConfigValues(defaultValues);
        }
      } catch {
        setConfigValues(defaultValues);
      }

      const agentCredentials = credentialsRes.items?.find(
        (item: { agent_slug: string }) => item.agent_slug === foundAgent.slug
      );
      const profiles = agentCredentials?.profiles || [];
      setCredentialProfiles(profiles);

      const hasCustomDefault = profiles.some((p: CredentialProfileData) => p.is_default);
      setIsRunnerHostDefault(!hasCustomDefault);
    } catch (err) {
      console.error("Failed to load agent config:", err);
      setError(t("settings.agentConfig.loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [agentSlug, t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleConfigChange = useCallback((fieldName: string, value: unknown) => {
    setConfigValues((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  }, []);

  const handleSaveConfig = useCallback(async () => {
    if (!agent) return;

    try {
      setSavingConfig(true);
      setError(null);

      const cleanedConfig: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(configValues)) {
        if (value !== undefined) {
          cleanedConfig[key] = value;
        }
      }

      await getAgentService().set_user_config(agent.slug, JSON.stringify(cleanedConfig));
      setSuccess(t("settings.agentConfig.configSaved"));
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to save config:", err);
      const msg = getLocalizedErrorMessage(err, t, t("settings.agentConfig.configSaveFailed"));
      setError(msg);
      toast.error(msg);
    } finally {
      setSavingConfig(false);
    }
  }, [agent, configValues, t]);

  const handleSetRunnerHostDefault = useCallback(async () => {
    try {
      setError(null);
      const currentDefault = credentialProfiles.find((p) => p.is_default);
      if (currentDefault) {
        await getUserCredentialService().update_agent_credential(BigInt(currentDefault.id), JSON.stringify({ is_default: false }));
      }
      setSuccess(t("settings.agentCredentials.defaultSet"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set RunnerHost as default:", err);
      const msg = getLocalizedErrorMessage(err, t, t("settings.agentCredentials.failedToSetDefault"));
      setError(msg);
      toast.error(msg);
    }
  }, [credentialProfiles, loadData, t]);

  const handleSetDefault = useCallback(async (profileId: number) => {
    try {
      setError(null);
      await getUserCredentialService().set_default_agent_credential(BigInt(profileId));
      setSuccess(t("settings.agentCredentials.defaultSet"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set default:", err);
      const msg = getLocalizedErrorMessage(err, t, t("settings.agentCredentials.failedToSetDefault"));
      setError(msg);
      toast.error(msg);
    }
  }, [loadData, t]);

  const handleDeleteProfile = useCallback(async (profileId: number) => {
    try {
      setError(null);
      await getUserCredentialService().delete_agent_credential(BigInt(profileId));
      setSuccess(t("settings.agentCredentials.profileDeleted"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to delete profile:", err);
      const msg = getLocalizedErrorMessage(err, t, t("settings.agentCredentials.failedToDelete"));
      setError(msg);
      toast.error(msg);
    }
  }, [loadData, t]);

  const handleSaveProfile = useCallback(async (
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => {
    if (!agent) return;

    const credentials = Object.keys(data.credentials).length > 0 ? data.credentials : undefined;

    if (editingProfile) {
      await getUserCredentialService().update_agent_credential(BigInt(editingProfile.id), JSON.stringify({
        name: data.name,
        description: data.description || undefined,
        is_runner_host: false,
        credentials,
      }));
      setSuccess(t("settings.agentCredentials.profileUpdated"));
    } else {
      await getUserCredentialService().create_agent_credential(agent.slug, JSON.stringify({
        name: data.name,
        description: data.description || undefined,
        is_runner_host: false,
        credentials: data.credentials,
      }));
      setSuccess(t("settings.agentCredentials.profileCreated"));
    }

    await loadData();
    setTimeout(() => setSuccess(null), 3000);
  }, [agent, loadData, t]);

  return {
    loading,
    savingConfig,
    agent,
    configFields,
    configValues,
    credentialFields,
    credentialProfiles,
    isRunnerHostDefault,
    error,
    success,

    handleConfigChange,
    handleSaveConfig,
    handleSetRunnerHostDefault,
    handleSetDefault,
    handleDeleteProfile,
    handleSaveProfile,
    setError,
    setSuccess,
    loadData,
  };
}
