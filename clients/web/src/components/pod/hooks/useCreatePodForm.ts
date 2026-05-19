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

export { RUNNER_HOST_PROFILE_ID } from "./useCreatePodFormTypes";
export type { CreatePodFormState, FormValidationErrors } from "./useCreatePodFormTypes";

export function useCreatePodForm(
  availableAgents: AgentData[],
  repositories: RepositoryData[],
  onSuccess?: (pod: PodData) => void,
  configValues?: Record<string, unknown>,
  overrides?: { repositoryId?: number | null },
): CreatePodFormState {
  const { setLastChoices } = usePodCreationStore();

  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<number | null>(null);
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [interactionMode, setInteractionMode] = useState<PodMode>(POD_MODE_PTY);
  const [prompt, setPrompt] = useState<string>("");
  const [alias, setAlias] = useState<string>("");
  const [perpetual, setPerpetual] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<FormValidationErrors>({});

  const [rawLayerMode, setRawLayerModeState] = useState(false);
  const [rawLayerText, setRawLayerText] = useState("");

  const creds = useCredentialProfiles(selectedAgent);

  const prefsInitializedRef = usePrefsAutoFill(
    availableAgents, repositories, setSelectedAgent, setSelectedRepository, setSelectedBranch,
    overrides,
  );

  const selectedAgentSlug = useMemo(() => {
    if (!selectedAgent) return "";
    return availableAgents.find((a) => a.slug === selectedAgent)?.slug || "";
  }, [selectedAgent, availableAgents]);

  const supportedModes = useMemo(() => {
    if (!selectedAgent) return [POD_MODE_PTY];
    const agent = availableAgents.find((a) => a.slug === selectedAgent);
    const raw = agent?.supported_modes;
    const modes = Array.isArray(raw)
      ? raw.map((m: string) => m.trim()).filter(Boolean)
      : (typeof raw === "string" ? raw.split(",").map((m: string) => m.trim()).filter(Boolean) : []);
    return modes.length > 0 ? modes : [POD_MODE_PTY];
  }, [selectedAgent, availableAgents]);

  const isValid = useMemo(() => selectedAgent !== null && selectedAgent !== "", [selectedAgent]);

  useEffect(() => {
    if (selectedAgent && !availableAgents.find(a => a.slug === selectedAgent)) {
      setSelectedAgent(null);
      creds.setCredentialProfiles([]);
      creds.setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
      setInteractionMode(POD_MODE_PTY);
    }
  }, [availableAgents, selectedAgent, creds]);

  useEffect(() => {
    if (!selectedAgent) { setInteractionMode(POD_MODE_PTY); return; }
    if (!supportedModes.includes(interactionMode)) {
      setInteractionMode(supportedModes[0] as PodMode);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedAgent, supportedModes]);

  useEffect(() => {
    if (!selectedRepository) { setSelectedBranch(""); return; }
    const repo = repositories.find((r) => r.id === selectedRepository);
    if (repo?.default_branch) setSelectedBranch(repo.default_branch);
  }, [selectedRepository, repositories]);

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
    setPerpetual(false);
    setError(null);
    setWarning(null);
    setValidationErrors({});
    setRawLayerModeState(false);
    setRawLayerText("");
    prefsInitializedRef.current = false;
  }, [creds, prefsInitializedRef]);

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
      setWarning(null);
      try {
        const result = await submitCreatePod({
          selectedAgent, alias, perpetual, selectedRunnerId,
          agentfileLayer: agentfileLayer || undefined, options,
        });
        if (result) {
          setLastChoices({
            lastAgentSlug: selectedAgent, lastRepositoryId: selectedRepository,
            lastCredentialProfileId: creds.selectedCredentialProfile > 0 ? creds.selectedCredentialProfile : null,
            lastBranchName: selectedBranch || null,
          });
          if (result.warning) setWarning(result.warning);
          onSuccess?.(result.pod);
        }
        return result?.pod ?? null;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to create pod";
        setError(message);
        console.error("Failed to create pod:", err);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [selectedAgent, selectedRepository, selectedBranch, creds.selectedCredentialProfile, alias, perpetual, agentfileLayer, onSuccess, validate, setLastChoices]
  );

  return {
    selectedAgent, selectedRepository, selectedBranch,
    selectedCredentialProfile: creds.selectedCredentialProfile,
    interactionMode, prompt, alias, perpetual,
    credentialProfiles: creds.credentialProfiles, loadingCredentials: creds.loadingCredentials,
    setSelectedAgent, setSelectedRepository, setSelectedBranch,
    setSelectedCredentialProfile: creds.setSelectedCredentialProfile,
    setInteractionMode, setPrompt, setAlias, setPerpetual, selectedAgentSlug, supportedModes,
    loading, error, warning, validationErrors, isValid, reset, validate, submit,
    rawLayerMode, rawLayerText, agentfileLayer, setRawLayerMode, setRawLayerText,
  };
}
