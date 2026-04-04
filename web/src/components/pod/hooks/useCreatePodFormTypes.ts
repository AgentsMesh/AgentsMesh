import { PodData, CredentialProfileData } from "@/lib/api";
import type { PodMode } from "@/lib/pod-modes";

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
  selectedAgent: string | null;
  selectedRepository: number | null;
  selectedBranch: string;
  selectedCredentialProfile: number; // 0 = RunnerHost, >0 = custom profile ID
  interactionMode: PodMode;
  prompt: string;
  alias: string;

  // Credential profiles for selected agent
  credentialProfiles: CredentialProfileData[];
  loadingCredentials: boolean;

  // Actions
  setSelectedAgent: (slug: string | null) => void;
  setSelectedRepository: (id: number | null) => void;
  setSelectedBranch: (branch: string) => void;
  setSelectedCredentialProfile: (id: number) => void;
  setInteractionMode: (mode: PodMode) => void;
  setPrompt: (prompt: string) => void;
  setAlias: (alias: string) => void;

  // AgentFile Layer
  rawLayerMode: boolean;
  rawLayerText: string;
  agentfileLayer: string;
  setRawLayerMode: (enabled: boolean) => void;
  setRawLayerText: (text: string) => void;

  // Computed
  selectedAgentSlug: string;
  supportedModes: string[]; // parsed from agent type's supported_modes

  // Form state
  loading: boolean;
  error: string | null;
  validationErrors: FormValidationErrors;
  isValid: boolean;

  // Actions
  reset: () => void;
  validate: () => boolean;
  submit: (
    selectedRunnerId: number | null | undefined,
    pluginConfig: Record<string, unknown>,
    options?: { ticketSlug?: string; cols?: number; rows?: number }
  ) => Promise<PodData | null>;
}
