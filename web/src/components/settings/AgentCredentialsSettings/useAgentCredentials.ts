"use client";

import { useState, useEffect, useCallback } from "react";
import {
  userAgentCredentialApi,
  agentApi,
  type CredentialProfileData,
  type CredentialProfilesByAgentType,
  type AgentTypeData,
} from "@/lib/api";
import type { AgentCredentialsState, AgentCredentialsActions, CredentialFormData } from "./types";

/**
 * Custom hook for managing agent credentials state and actions
 */
export function useAgentCredentials(
  t: (key: string) => string
): AgentCredentialsState & AgentCredentialsActions {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Data state
  const [profilesByAgentType, setProfilesByAgentType] = useState<CredentialProfilesByAgentType[]>([]);
  const [agentTypes, setAgentTypes] = useState<AgentTypeData[]>([]);
  const [expandedAgentTypes, setExpandedAgentTypes] = useState<Set<number>>(new Set());
  const [runnerHostDefaults, setRunnerHostDefaults] = useState<Set<number>>(new Set());

  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [profilesRes, typesRes] = await Promise.all([
        userAgentCredentialApi.list(),
        agentApi.listTypes(),
      ]);

      setProfilesByAgentType(profilesRes.items || []);
      setAgentTypes(typesRes.agent_types || []);

      // Determine which agent types have RunnerHost as default
      const runnerHostDefaultSet = new Set<number>();
      const agentTypeIds = new Set(typesRes.agent_types?.map((at: AgentTypeData) => at.id) || []);

      // Start by assuming all agent types default to RunnerHost
      agentTypeIds.forEach((id: number) => runnerHostDefaultSet.add(id));

      // Remove from set if there's a custom default profile
      profilesRes.items?.forEach((item) => {
        const hasCustomDefault = item.profiles.some((p) => p.is_default);
        if (hasCustomDefault) {
          runnerHostDefaultSet.delete(item.agent_type_id);
        }
      });

      setRunnerHostDefaults(runnerHostDefaultSet);

      // Auto-expand first agent type or those with profiles
      const expandedIds = new Set<number>();
      if (typesRes.agent_types?.length > 0) {
        expandedIds.add(typesRes.agent_types[0].id);
      }
      profilesRes.items?.forEach((item) => {
        if (item.profiles.length > 0) {
          expandedIds.add(item.agent_type_id);
        }
      });
      setExpandedAgentTypes(expandedIds);
    } catch (err) {
      console.error("Failed to load agent credentials:", err);
      setError(t("settings.agentCredentials.failedToLoad"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Toggle agent type expansion
  const toggleAgentType = useCallback((agentTypeId: number) => {
    setExpandedAgentTypes((prev) => {
      const next = new Set(prev);
      if (next.has(agentTypeId)) {
        next.delete(agentTypeId);
      } else {
        next.add(agentTypeId);
      }
      return next;
    });
  }, []);

  // Set RunnerHost as default for an agent type
  const handleSetRunnerHostDefault = useCallback(async (agentTypeId: number) => {
    try {
      setError(null);
      const group = profilesByAgentType.find((g) => g.agent_type_id === agentTypeId);
      const currentDefault = group?.profiles.find((p) => p.is_default);
      if (currentDefault) {
        await userAgentCredentialApi.update(currentDefault.id, { is_default: false });
      }
      setSuccess(t("settings.agentCredentials.defaultSet"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set RunnerHost as default:", err);
      setError(t("settings.agentCredentials.failedToSetDefault"));
    }
  }, [profilesByAgentType, loadData, t]);

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
      setError(t("settings.agentCredentials.failedToSetDefault"));
    }
  }, [loadData, t]);

  // Delete profile (no confirmation - caller should handle confirmation dialog)
  const handleDelete = useCallback(async (profileId: number) => {
    try {
      setError(null);
      await userAgentCredentialApi.delete(profileId);
      setSuccess(t("settings.agentCredentials.profileDeleted"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to delete profile:", err);
      setError(t("settings.agentCredentials.failedToDelete"));
    }
  }, [loadData, t]);

  // Save credential profile (create or update)
  // Sends only the active credential method's value; the other is excluded.
  // Backend replaces the entire credentials object, clearing stale fields.
  const handleSaveProfile = useCallback(async (
    agentTypeId: number,
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => {
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
      await userAgentCredentialApi.create(agentTypeId, {
        name: data.name,
        description: data.description || undefined,
        is_runner_host: false,
        credentials: credentials,
      });
      setSuccess(t("settings.agentCredentials.profileCreated"));
    }

    await loadData();
    setTimeout(() => setSuccess(null), 3000);
  }, [loadData, t]);

  // Get profiles for a specific agent type
  const getProfilesForAgentType = useCallback((agentTypeId: number): CredentialProfileData[] => {
    const group = profilesByAgentType.find((g) => g.agent_type_id === agentTypeId);
    return group?.profiles || [];
  }, [profilesByAgentType]);

  return {
    // State
    loading,
    error,
    success,
    profilesByAgentType,
    agentTypes,
    expandedAgentTypes,
    runnerHostDefaults,

    // Actions
    toggleAgentType,
    handleSetRunnerHostDefault,
    handleSetDefault,
    handleDelete,
    handleSaveProfile,
    getProfilesForAgentType,
    setError,
    setSuccess,
  };
}
