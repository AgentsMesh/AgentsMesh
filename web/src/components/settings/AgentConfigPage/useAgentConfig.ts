"use client";

import { useState, useEffect, useCallback } from "react";
import {
  agentApi,
  userAgentConfigApi,
  userAgentCredentialApi,
  type ConfigField,
  type AgentTypeData,
  type CredentialProfileData,
} from "@/lib/api";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import type { AgentConfigState, AgentConfigActions, CredentialFormData } from "./types";

/**
 * Custom hook for managing agent configuration state and actions
 */
export function useAgentConfig(
  agentSlug: string,
  t: (key: string) => string
): AgentConfigState & AgentConfigActions {
  // Loading states
  const [loading, setLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);

  // Data states
  const [agentType, setAgentType] = useState<AgentTypeData | null>(null);
  const [configFields, setConfigFields] = useState<ConfigField[]>([]);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [isRunnerHostDefault, setIsRunnerHostDefault] = useState(true);

  // UI states
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Load all data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Load agent types to find the one matching the slug
      const typesRes = await agentApi.listTypes();
      const foundAgentType = typesRes.agent_types?.find(
        (at: AgentTypeData) => at.slug === agentSlug
      );

      if (!foundAgentType) {
        setError(t("settings.agentConfig.agentNotFound"));
        setLoading(false);
        return;
      }

      setAgentType(foundAgentType);

      // Load data in parallel
      const [schemaRes, credentialsRes] = await Promise.all([
        agentApi.getConfigSchema(foundAgentType.id).catch(() => ({ schema: { fields: [] } })),
        userAgentCredentialApi.list().catch(() => ({ items: [] })),
      ]);

      // Set config schema fields
      const fields = schemaRes.schema?.fields || [];
      setConfigFields(fields);

      // Initialize config values with defaults from schema
      const defaultValues: Record<string, unknown> = {};
      for (const field of fields) {
        if (field.default !== undefined) {
          defaultValues[field.name] = field.default;
        }
      }

      // Try to load user's saved config
      try {
        const userConfigRes = await userAgentConfigApi.get(foundAgentType.id);
        if (userConfigRes.config?.config_values) {
          // Merge user config over defaults
          setConfigValues({ ...defaultValues, ...userConfigRes.config.config_values });
        } else {
          setConfigValues(defaultValues);
        }
      } catch {
        // No saved config, use defaults
        setConfigValues(defaultValues);
      }

      // Extract credential profiles for this agent type
      const agentCredentials = credentialsRes.items?.find(
        (item: { agent_type_id: number }) => item.agent_type_id === foundAgentType.id
      );
      const profiles = agentCredentials?.profiles || [];
      setCredentialProfiles(profiles);

      // Check if RunnerHost is default (no custom profile is default)
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

  // Handle config field change
  const handleConfigChange = useCallback((fieldName: string, value: unknown) => {
    setConfigValues((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  }, []);

  // Save runtime config
  const handleSaveConfig = useCallback(async () => {
    if (!agentType) return;

    try {
      setSavingConfig(true);
      setError(null);

      // Filter out undefined values, but keep empty strings (e.g., "Follow Runner" model option)
      // and false for booleans
      const cleanedConfig: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(configValues)) {
        if (value !== undefined) {
          cleanedConfig[key] = value;
        }
      }

      await userAgentConfigApi.set(agentType.id, cleanedConfig);
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
  }, [agentType, configValues, t]);

  // Set RunnerHost as default
  const handleSetRunnerHostDefault = useCallback(async () => {
    try {
      setError(null);
      const currentDefault = credentialProfiles.find((p) => p.is_default);
      if (currentDefault) {
        await userAgentCredentialApi.update(currentDefault.id, { is_default: false });
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

  // Set custom profile as default
  const handleSetDefault = useCallback(async (profileId: number) => {
    try {
      setError(null);
      await userAgentCredentialApi.setDefault(profileId);
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

  // Delete credential profile (no confirmation - caller should handle confirmation dialog)
  const handleDeleteProfile = useCallback(async (profileId: number) => {
    try {
      setError(null);
      await userAgentCredentialApi.delete(profileId);
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

  // Save credential profile (create or update)
  // Sends only the active credential method's value; the other is excluded.
  // Backend replaces the entire credentials object, clearing stale fields.
  const handleSaveProfile = useCallback(async (
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => {
    if (!agentType) return;

    const credentials: Record<string, string> = {};

    // base_url is shared across both methods
    if (data.baseUrl) credentials.base_url = data.baseUrl;

    // Only include the active credential method
    if (data.credentialMethod === "api_key") {
      if (data.apiKey) credentials.api_key = data.apiKey;
    } else {
      if (data.authToken) credentials.auth_token = data.authToken;
    }

    if (editingProfile) {
      // When updating, always send credentials object to ensure stale fields
      // from the other method are cleared (backend replaces entire object)
      await userAgentCredentialApi.update(editingProfile.id, {
        name: data.name,
        description: data.description || undefined,
        is_runner_host: false,
        credentials: Object.keys(credentials).length > 0 ? credentials : undefined,
      });
      setSuccess(t("settings.agentCredentials.profileUpdated"));
    } else {
      await userAgentCredentialApi.create(agentType.id, {
        name: data.name,
        description: data.description || undefined,
        is_runner_host: false,
        credentials: credentials,
      });
      setSuccess(t("settings.agentCredentials.profileCreated"));
    }

    await loadData();
    setTimeout(() => setSuccess(null), 3000);
  }, [agentType, loadData, t]);

  return {
    // State
    loading,
    savingConfig,
    agentType,
    configFields,
    configValues,
    credentialProfiles,
    isRunnerHostDefault,
    error,
    success,

    // Actions
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
