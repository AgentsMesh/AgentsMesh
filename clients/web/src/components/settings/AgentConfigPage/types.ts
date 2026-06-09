import type { ConfigField, AgentData } from "@/lib/api";
import type { CredentialProfileViewModel } from "../_shared/credentialViewModel";
import type { CredentialFormData } from "../AgentCredentialsSettings/credentialForms/types";

// Re-export: CredentialFormData is defined once in credentialForms/types.
export type { CredentialFormData };

/**
 * Props for AgentConfigPage component
 */
export interface AgentConfigPageProps {
  agentSlug: string;
}

/**
 * State returned by useAgentConfig hook
 */
export interface AgentConfigState {
  // Loading states
  loading: boolean;
  savingConfig: boolean;

  // Data
  agent: AgentData | null;
  configFields: ConfigField[];
  configValues: Record<string, unknown>;
  credentialProfiles: CredentialProfileViewModel[];
  noPrimaryBundle: boolean;
  runtimeBundles: RuntimeBundleViewModel[];

  // UI feedback
  error: string | null;
  success: string | null;
}

/**
 * Actions returned by useAgentConfig hook
 */
export interface AgentConfigActions {
  // Config actions
  handleConfigChange: (fieldName: string, value: unknown) => void;
  handleSaveConfig: () => Promise<void>;

  // Credential actions
  handleClearPrimaryBundle: () => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDeleteProfile: (profileId: number) => Promise<void>;
  handleSaveProfile: (data: CredentialFormData, editingProfile: CredentialProfileViewModel | null) => Promise<void>;

  // Runtime bundle actions
  handleSetRuntimePrimary: (id: number) => Promise<void>;
  handleClearRuntimePrimary: () => Promise<void>;
  handleDeleteRuntimeBundle: (id: number) => Promise<void>;
  handleSaveRuntimeBundle: (data: RuntimeBundleFormData, editingBundle: RuntimeBundleViewModel | null) => Promise<void>;

  // UI actions
  setError: (error: string | null) => void;
  setSuccess: (success: string | null) => void;
  loadData: () => Promise<void>;
}

/**
 * Runtime-kind EnvBundle as the per-agent settings page sees it. Plaintext
 * values round-trip via `configured_values` (the backend doesn't strip them
 * the way it does for credential kind).
 */
export interface RuntimeBundleViewModel {
  id: number;
  agent_slug: string;
  name: string;
  description?: string;
  is_default: boolean;
  is_active: boolean;
  configured_fields?: string[];
  configured_values?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

/**
 * Payload emitted by the runtime bundle dialog. The dialog builds `data`
 * from the KV editor; useAgentConfig passes it straight to envBundleService
 * with kind="runtime".
 */
export interface RuntimeBundleFormData {
  name: string;
  description: string;
  data: Record<string, string>;
}

/**
 * Props for CredentialsSection component
 */
export interface CredentialsSectionProps {
  agentSlug: string;
  noPrimaryBundle: boolean;
  credentialProfiles: CredentialProfileViewModel[];
  onClearPrimary: () => Promise<void>;
  onSetDefault: (profileId: number) => Promise<void>;
  onEdit: (profile: CredentialProfileViewModel) => void;
  onDelete: (profileId: number) => Promise<void>;
  onAdd: () => void;
  t: (key: string) => string;
}

/**
 * Props for RuntimeConfigSection component
 */
export interface RuntimeConfigSectionProps {
  configFields: ConfigField[];
  configValues: Record<string, unknown>;
  agentSlug: string;
  saving: boolean;
  onChange: (fieldName: string, value: unknown) => void;
  onSave: () => Promise<void>;
  t: (key: string) => string;
}
