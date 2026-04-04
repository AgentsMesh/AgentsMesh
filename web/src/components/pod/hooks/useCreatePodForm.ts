import { useState, useCallback, useMemo, useEffect } from "react";
import { PodData, AgentData, RepositoryData } from "@/lib/api";
import { usePodCreationStore } from "@/stores/podCreation";
import { buildAgentfileLayer } from "@/lib/agentfile-layer";
import { POD_MODE_PTY } from "@/lib/pod-modes";
import type { PodMode } from "@/lib/pod-modes";
import { submitCreatePod } from "./useCreatePodFormSubmit";
import { usePrefsAutoFill, useCredentialProfiles } from "./useCreatePodFormEffects";
import type { CreatePodFormState, FormValidationErrors } from "./useCreatePodFormTypes";
import { RUNNER_HOST_PROFILE_ID } from "./useCreatePodFormTypes";

// Re-export types for consumers
export { RUNNER_HOST_PROFILE_ID } from "./useCreatePodFormTypes";
export type { CreatePodFormState, FormValidationErrors } from "./useCreatePodFormTypes";

/**
 * Hook to manage Create Pod form state and submission
 * Note: Runner selection is managed by usePodCreationData
 * This hook manages agent selection and other form fields
 */
export function useCreatePodForm(
  availableAgents: AgentData[],
  repositories: RepositoryData[],
  onSuccess?: (pod: PodData) => void,
  configValues?: Record<string, unknown>
): CreatePodFormState {
  const { setLastChoices } = usePodCreationStore();

  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<number | null>(null);
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [interactionMode, setInteractionMode] = useState<PodMode>(POD_MODE_PTY);
  const [prompt, setPrompt] = useState<string>("");
  const [alias, setAlias] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<FormValidationErrors>({});

  // AgentFile Layer state
  const [rawLayerMode, setRawLayerModeState] = useState(false);
  const [rawLayerText, setRawLayerText] = useState("");

  // Credential profiles (extracted hook)
  const creds = useCredentialProfiles(selectedAgent);

  // Auto-fill from saved preferences
  const prefsInitializedRef = usePrefsAutoFill(
    availableAgents, repositories, setSelectedAgent, setSelectedRepository, setSelectedBranch,
  );

  // Compute agent slug from selected agent
  const selectedAgentSlug = useMemo(() => {
    if (!selectedAgent) return "";
    return availableAgents.find((a) => a.slug === selectedAgent)?.slug || "";
  }, [selectedAgent, availableAgents]);

  // Parse supported modes from selected agent type
  const supportedModes = useMemo(() => {
    if (!selectedAgent) return [POD_MODE_PTY];
    const agent = availableAgents.find((a) => a.slug === selectedAgent);
    const modes = agent?.supported_modes?.split(",").map((m) => m.trim()).filter(Boolean) || [];
    return modes.length > 0 ? modes : [POD_MODE_PTY];
  }, [selectedAgent, availableAgents]);

  const isValid = useMemo(() => selectedAgent !== null && selectedAgent !== "", [selectedAgent]);

  // Reset agent selection when available agents change
  useEffect(() => {
    if (selectedAgent && !availableAgents.find(a => a.slug === selectedAgent)) {
      setSelectedAgent(null);
      creds.setCredentialProfiles([]);
      creds.setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
      setInteractionMode(POD_MODE_PTY);
    }
  }, [availableAgents, selectedAgent, creds]);

  // Auto-set interaction mode when agent changes based on supported modes
  useEffect(() => {
    if (!selectedAgent) { setInteractionMode(POD_MODE_PTY); return; }
    if (!supportedModes.includes(interactionMode)) {
      setInteractionMode(supportedModes[0] as PodMode);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedAgent, supportedModes]);

  // Auto-select default branch when repository is selected
  useEffect(() => {
    if (!selectedRepository) { setSelectedBranch(""); return; }
    const repo = repositories.find((r) => r.id === selectedRepository);
    if (repo?.default_branch) setSelectedBranch(repo.default_branch);
  }, [selectedRepository, repositories]);

  // Clear validation error when field changes
  useEffect(() => {
    if (selectedAgent && validationErrors.agent) {
      setValidationErrors((prev) => ({ ...prev, agent: undefined }));
    }
  }, [selectedAgent, validationErrors.agent]);

  const validate = useCallback((): boolean => {
    const errors: FormValidationErrors = {};
    if (!selectedAgent) errors.agent = "Please select an agent";
    if (selectedRepository && !selectedBranch.trim()) {
      errors.branch = "Branch name is recommended when using a repository";
    }
    if (selectedBranch.trim() && !/^[a-zA-Z0-9._/-]+$/.test(selectedBranch)) {
      errors.branch = "Branch name contains invalid characters";
    }
    setValidationErrors(errors);
    return Object.keys(errors).filter(k => errors[k as keyof FormValidationErrors]).length === 0;
  }, [selectedAgent, selectedRepository, selectedBranch]);

  const reset = useCallback(() => {
    setSelectedAgent(null);
    setSelectedRepository(null);
    setSelectedBranch("");
    creds.setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
    creds.setCredentialProfiles([]);
    setInteractionMode(POD_MODE_PTY);
    setPrompt("");
    setAlias("");
    setError(null);
    setValidationErrors({});
    setRawLayerModeState(false);
    setRawLayerText("");
    prefsInitializedRef.current = false;
  }, [creds, prefsInitializedRef]);

  // AgentFile Layer: compute from form fields
  const generatedLayer = useMemo(() => {
    const repoSlug = selectedRepository
      ? repositories.find((r) => r.id === selectedRepository)?.slug
      : undefined;
    const credProfileName = creds.selectedCredentialProfile === RUNNER_HOST_PROFILE_ID
      ? undefined
      : creds.credentialProfiles.find(
          (p) => p.id === creds.selectedCredentialProfile
        )?.name;
    return buildAgentfileLayer({
      configValues: configValues ?? {},
      repositorySlug: repoSlug,
      branchName: selectedBranch || undefined,
      interactionMode,
      credentialProfileName: credProfileName,
      prompt: prompt || undefined,
    });
  }, [configValues, selectedRepository, repositories, selectedBranch, creds.selectedCredentialProfile, creds.credentialProfiles, interactionMode, prompt]);

  const agentfileLayer = rawLayerMode ? rawLayerText : generatedLayer;

  const setRawLayerMode = useCallback((enabled: boolean) => {
    if (enabled && !rawLayerText) {
      setRawLayerText(generatedLayer);
    }
    setRawLayerModeState(enabled);
  }, [generatedLayer, rawLayerText]);

  const submit = useCallback(
    async (
      selectedRunnerId: number | null | undefined,
      pluginConfig: Record<string, unknown>,
      options?: { ticketSlug?: string; cols?: number; rows?: number }
    ): Promise<PodData | null> => {
      if (!validate()) return null;
      if (!selectedAgent) { setError("Please select an agent"); return null; }
      setLoading(true);
      setError(null);
      try {
        const pod = await submitCreatePod({
          selectedAgent, alias, selectedRunnerId,
          agentfileLayer: agentfileLayer || undefined, options,
        });
        if (pod) {
          setLastChoices({
            lastAgentSlug: selectedAgent, lastRepositoryId: selectedRepository,
            lastCredentialProfileId: creds.selectedCredentialProfile > 0 ? creds.selectedCredentialProfile : null,
            lastBranchName: selectedBranch || null,
          });
          onSuccess?.(pod);
        }
        return pod;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to create pod";
        setError(message);
        console.error("Failed to create pod:", err);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [selectedAgent, selectedAgentSlug, selectedRepository, selectedBranch, creds.selectedCredentialProfile, interactionMode, prompt, alias, agentfileLayer, onSuccess, validate, setLastChoices]
  );

  return {
    selectedAgent, selectedRepository, selectedBranch,
    selectedCredentialProfile: creds.selectedCredentialProfile,
    interactionMode, prompt, alias,
    credentialProfiles: creds.credentialProfiles, loadingCredentials: creds.loadingCredentials,
    setSelectedAgent, setSelectedRepository, setSelectedBranch,
    setSelectedCredentialProfile: creds.setSelectedCredentialProfile,
    setInteractionMode, setPrompt, setAlias, selectedAgentSlug, supportedModes,
    loading, error, validationErrors, isValid, reset, validate, submit,
    // AgentFile Layer
    rawLayerMode, rawLayerText, agentfileLayer, setRawLayerMode, setRawLayerText,
  };
}
