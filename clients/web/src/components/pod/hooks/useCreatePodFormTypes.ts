import { PodData, CredentialProfileData } from "@/lib/api";
import type { PodMode } from "@/lib/pod-modes";

export interface FormValidationErrors {
  runner?: string;
  agent?: string;
  repository?: string;
  branch?: string;
  prompt?: string;
}

export const RUNNER_HOST_PROFILE_ID = 0;

export interface CreatePodFormState {
  selectedAgent: string | null;
  selectedRepository: number | null;
  selectedBranch: string;
  selectedCredentialProfile: number; // 0 = RunnerHost, >0 = custom profile ID
  interactionMode: PodMode;
  prompt: string;
  alias: string;
  perpetual: boolean;

  credentialProfiles: CredentialProfileData[];
  loadingCredentials: boolean;

  setSelectedAgent: (slug: string | null) => void;
  setSelectedRepository: (id: number | null) => void;
  setSelectedBranch: (branch: string) => void;
  setSelectedCredentialProfile: (id: number) => void;
  setInteractionMode: (mode: PodMode) => void;
  setPrompt: (prompt: string) => void;
  setAlias: (alias: string) => void;
  setPerpetual: (perpetual: boolean) => void;

  rawLayerMode: boolean;
  rawLayerText: string;
  agentfileLayer: string;
  setRawLayerMode: (enabled: boolean) => void;
  setRawLayerText: (text: string) => void;

  selectedAgentSlug: string;
  supportedModes: string[]; // parsed from agent type's supported_modes

  loading: boolean;
  error: string | null;
  validationErrors: FormValidationErrors;
  isValid: boolean;

  reset: () => void;
  validate: () => boolean;
  submit: (
    selectedRunnerId: number | null | undefined,
    pluginConfig: Record<string, unknown>,
    options?: { ticketSlug?: string; cols?: number; rows?: number }
  ) => Promise<PodData | null>;
}
