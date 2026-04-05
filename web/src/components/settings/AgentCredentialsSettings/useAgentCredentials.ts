"use client";

import { useState, useEffect, useCallback } from "react";
import {
  userAgentCredentialApi,
  agentApi,
  type CredentialField,
  type CredentialProfileData,
  type CredentialProfilesByAgent,
  type AgentData,
} from "@/lib/api";
import type { AgentCredentialsState, AgentCredentialsActions, CredentialFormData } from "./types";

/**
 * Custom hook for managing agent credentials state and actions.
 * Fetches credential field definitions from config-schema API (AgentFile SSOT).
 */
export function useAgentCredentials(
  t: (key: string) => string
): AgentCredentialsState & AgentCredentialsActions {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Data state
  const [profilesByAgent, setProfilesByAgent] = useState<CredentialProfilesByAgent[]>([]);
  const [agents, setAgents] = useState<AgentData[]>([]);
  const [expandedAgents, setExpandedAgents] = useState<Set<string>>(new Set());
  const [runnerHostDefaults, setRunnerHostDefaults] = useState<Set<string>>(new Set());
  const [credentialFieldsByAgent, setCredentialFieldsByAgent] = useState<Map<string, CredentialField[]>>(new Map());

  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [profilesRes, agentsRes] = await Promise.all([
        userAgentCredentialApi.list(),
        agentApi.list(),
      ]);

      setProfilesByAgent(profilesRes.items || []);
      const agentList = agentsRes.agents || [];
      setAgents(agentList);

      // Fetch credential fields for all agents in parallel
      const fieldsMap = new Map<string, CredentialField[]>();
      const schemaResults = await Promise.allSettled(
        agentList.map((a: AgentData) => agentApi.getConfigSchema(a.slug))
      );
      agentList.forEach((a: AgentData, i: number) => {
        const result = schemaResults[i];
        if (result.status === "fulfilled") {
          fieldsMap.set(a.slug, result.value.schema?.credential_fields || []);
        }
      });
      setCredentialFieldsByAgent(fieldsMap);

      // Determine which agents have RunnerHost as default
      const runnerHostDefaultSet = new Set<string>();
      const agentSlugs = new Set(agentList.map((a: AgentData) => a.slug));
      agentSlugs.forEach((slug: string) => runnerHostDefaultSet.add(slug));
      profilesRes.items?.forEach((item) => {
        if (item.profiles.some((p) => p.is_default)) {
          runnerHostDefaultSet.delete(item.agent_slug);
        }
      });
      setRunnerHostDefaults(runnerHostDefaultSet);

      // Auto-expand first agent or those with profiles
      const expandedIds = new Set<string>();
      if (agentList.length > 0) {
        expandedIds.add(agentList[0].slug);
      }
      profilesRes.items?.forEach((item) => {
        if (item.profiles.length > 0) expandedIds.add(item.agent_slug);
      });
      setExpandedAgents(expandedIds);
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

  const toggleAgent = useCallback((agentSlug: string) => {
    setExpandedAgents((prev) => {
      const next = new Set(prev);
      if (next.has(agentSlug)) next.delete(agentSlug);
      else next.add(agentSlug);
      return next;
    });
  }, []);

  const handleSetRunnerHostDefault = useCallback(async (agentSlug: string) => {
    try {
      setError(null);
      const group = profilesByAgent.find((g) => g.agent_slug === agentSlug);
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
  }, [profilesByAgent, loadData, t]);

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

  // Save credential profile — credentials keys are full ENV names from AgentFile.
  const handleSaveProfile = useCallback(async (
    agentSlug: string,
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => {
    const credentials = Object.keys(data.credentials).length > 0 ? data.credentials : undefined;

    try {
      if (editingProfile) {
        await userAgentCredentialApi.update(editingProfile.id, {
          name: data.name,
          description: data.description || undefined,
          is_runner_host: false,
          credentials,
        });
        setSuccess(t("settings.agentCredentials.profileUpdated"));
      } else {
        await userAgentCredentialApi.create(agentSlug, {
          name: data.name,
          description: data.description || undefined,
          is_runner_host: false,
          credentials: data.credentials,
        });
        setSuccess(t("settings.agentCredentials.profileCreated"));
      }

      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to save credential profile:", err);
      throw err;
    }
  }, [loadData, t]);

  const getProfilesForAgent = useCallback((agentSlug: string): CredentialProfileData[] => {
    const group = profilesByAgent.find((g) => g.agent_slug === agentSlug);
    return group?.profiles || [];
  }, [profilesByAgent]);

  return {
    loading, error, success,
    profilesByAgent, agents, expandedAgents, runnerHostDefaults,
    credentialFieldsByAgent,
    toggleAgent, handleSetRunnerHostDefault, handleSetDefault,
    handleDelete, handleSaveProfile, getProfilesForAgent,
    setError, setSuccess,
  };
}
