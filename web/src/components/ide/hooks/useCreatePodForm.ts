import { useState, useCallback, useMemo, useEffect } from "react";
import { podApi, PodData, AgentTypeData, RepositoryData } from "@/lib/api/client";
import { userAgentCredentialApi, CredentialProfileData } from "@/lib/api";

/**
 * Validation errors for the form
 */
export interface FormValidationErrors {
  runner?: string;
  agent?: string;
  repository?: string;
  branch?: string;
  prompt?: string;
}

// Special value for RunnerHost (use Runner's local environment)
export const RUNNER_HOST_PROFILE_ID = 0;

export interface CreatePodFormState {
  // Selection state (order: Runner -> Agent -> Others)
  selectedAgent: number | null;
  selectedRepository: number | null;
  selectedBranch: string;
  selectedCredentialProfile: number; // 0 = RunnerHost, >0 = custom profile ID
  prompt: string;

  // Credential profiles for selected agent
  credentialProfiles: CredentialProfileData[];
  loadingCredentials: boolean;

  // Actions
  setSelectedAgent: (id: number | null) => void;
  setSelectedRepository: (id: number | null) => void;
  setSelectedBranch: (branch: string) => void;
  setSelectedCredentialProfile: (id: number) => void;
  setPrompt: (prompt: string) => void;

  // Computed
  selectedAgentSlug: string;

  // Form state
  loading: boolean;
  error: string | null;
  validationErrors: FormValidationErrors;
  isValid: boolean;

  // Actions
  reset: () => void;
  validate: (selectedRunnerId: number | null) => boolean;
  submit: (
    selectedRunnerId: number | null,
    pluginConfig: Record<string, unknown>,
    options?: { ticketId?: number }
  ) => Promise<PodData | null>;
}

/**
 * Hook to manage Create Pod form state and submission
 * Note: Runner selection is managed by usePodCreationData
 * This hook manages agent selection and other form fields
 */
export function useCreatePodForm(
  availableAgentTypes: AgentTypeData[],
  repositories: RepositoryData[],
  onSuccess?: (pod: PodData) => void
): CreatePodFormState {
  const [selectedAgent, setSelectedAgent] = useState<number | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<number | null>(null);
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [selectedCredentialProfile, setSelectedCredentialProfile] = useState<number>(RUNNER_HOST_PROFILE_ID);
  const [prompt, setPrompt] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<FormValidationErrors>({});

  // Credential profiles state
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [loadingCredentials, setLoadingCredentials] = useState(false);

  // Compute agent slug from selected agent
  const selectedAgentSlug = useMemo(() => {
    if (!selectedAgent) return "";
    const agent = availableAgentTypes.find((a) => a.id === selectedAgent);
    return agent?.slug || "";
  }, [selectedAgent, availableAgentTypes]);

  // Compute form validity (runner validation is done externally)
  const isValid = useMemo(() => {
    return selectedAgent !== null;
  }, [selectedAgent]);

  // Reset agent selection when available agent types change (e.g., when runner changes)
  useEffect(() => {
    // If current selection is not in available types, reset it
    if (selectedAgent && !availableAgentTypes.find(a => a.id === selectedAgent)) {
      setSelectedAgent(null);
      setCredentialProfiles([]);
      setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
    }
  }, [availableAgentTypes, selectedAgent]);

  // Auto-select default branch when repository is selected
  useEffect(() => {
    if (!selectedRepository) {
      setSelectedBranch("");
      return;
    }

    const repo = repositories.find((r) => r.id === selectedRepository);
    if (repo?.default_branch) {
      setSelectedBranch(repo.default_branch);
    }
  }, [selectedRepository, repositories]);

  // Load credential profiles when agent is selected
  useEffect(() => {
    if (!selectedAgent) {
      setCredentialProfiles([]);
      setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
      return;
    }

    const loadCredentials = async () => {
      setLoadingCredentials(true);
      try {
        const res = await userAgentCredentialApi.listForAgentType(selectedAgent);
        const profiles = res.profiles || [];
        setCredentialProfiles(profiles);

        // Auto-select: if there's a default profile, use it; otherwise use RunnerHost
        const defaultProfile = profiles.find((p) => p.is_default);
        if (defaultProfile) {
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
  }, [selectedAgent]);

  // Clear validation error when field changes
  useEffect(() => {
    if (selectedAgent && validationErrors.agent) {
      setValidationErrors((prev) => ({ ...prev, agent: undefined }));
    }
  }, [selectedAgent, validationErrors.agent]);

  // Validate form
  const validate = useCallback((selectedRunnerId: number | null): boolean => {
    const errors: FormValidationErrors = {};

    if (!selectedRunnerId) {
      errors.runner = "Please select a runner";
    }

    if (!selectedAgent) {
      errors.agent = "Please select an agent type";
    }

    // Branch validation: if repository is selected but branch is empty, warn
    if (selectedRepository && !selectedBranch.trim()) {
      errors.branch = "Branch name is recommended when using a repository";
    }

    // Validate branch name format (optional, only if provided)
    if (selectedBranch.trim()) {
      const branchRegex = /^[a-zA-Z0-9._/-]+$/;
      if (!branchRegex.test(selectedBranch)) {
        errors.branch = "Branch name contains invalid characters";
      }
    }

    setValidationErrors(errors);
    return Object.keys(errors).filter(k => errors[k as keyof FormValidationErrors]).length === 0;
  }, [selectedAgent, selectedRepository, selectedBranch]);

  // Reset form
  const reset = useCallback(() => {
    setSelectedAgent(null);
    setSelectedRepository(null);
    setSelectedBranch("");
    setSelectedCredentialProfile(RUNNER_HOST_PROFILE_ID);
    setCredentialProfiles([]);
    setPrompt("");
    setError(null);
    setValidationErrors({});
  }, []);

  /**
   * Submit options for pod creation
   */
  interface SubmitOptions {
    /** Ticket ID to associate with the pod */
    ticketId?: number;
  }

  // Submit form
  const submit = useCallback(
    async (
      selectedRunnerId: number | null,
      pluginConfig: Record<string, unknown>,
      options?: SubmitOptions
    ): Promise<PodData | null> => {
      // Validate before submission
      if (!validate(selectedRunnerId)) {
        return null;
      }

      if (!selectedRunnerId || !selectedAgent) {
        setError("Please select a runner and agent");
        return null;
      }

      setLoading(true);
      setError(null);

      try {
        // Build config overrides for API
        const configOverrides: Record<string, unknown> = {
          ...pluginConfig,
        };

        const response = await podApi.create({
          agent_type_id: selectedAgent,
          runner_id: selectedRunnerId,
          repository_id: selectedRepository || undefined,
          branch_name: selectedBranch || undefined,
          initial_prompt: prompt,
          config_overrides: configOverrides,
          credential_profile_id: selectedCredentialProfile > 0 ? selectedCredentialProfile : undefined,
          ticket_id: options?.ticketId,
        });

        if (response.pod) {
          onSuccess?.(response.pod);
          return response.pod;
        }
        return null;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to create pod";
        setError(message);
        console.error("Failed to create pod:", err);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [selectedAgent, selectedRepository, selectedBranch, selectedCredentialProfile, prompt, onSuccess, validate]
  );

  return {
    selectedAgent,
    selectedRepository,
    selectedBranch,
    selectedCredentialProfile,
    prompt,
    credentialProfiles,
    loadingCredentials,
    setSelectedAgent,
    setSelectedRepository,
    setSelectedBranch,
    setSelectedCredentialProfile,
    setPrompt,
    selectedAgentSlug,
    loading,
    error,
    validationErrors,
    isValid,
    reset,
    validate,
    submit,
  };
}
