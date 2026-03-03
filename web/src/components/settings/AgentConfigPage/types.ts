import type { ConfigField, AgentTypeData, CredentialProfileData } from "@/lib/api";

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
  agentType: AgentTypeData | null;
  configFields: ConfigField[];
  configValues: Record<string, unknown>;
  credentialProfiles: CredentialProfileData[];
  isRunnerHostDefault: boolean;

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
  handleSetRunnerHostDefault: () => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDeleteProfile: (profileId: number) => Promise<void>;
  handleSaveProfile: (data: CredentialFormData, editingProfile: CredentialProfileData | null) => Promise<void>;

  // UI actions
  setError: (error: string | null) => void;
  setSuccess: (success: string | null) => void;
  loadData: () => Promise<void>;
}

/**
 * Credential method type - api_key and auth_token are mutually exclusive
 */
export type CredentialMethod = "api_key" | "auth_token";

/**
 * Credential form data for add/edit dialog
 */
export interface CredentialFormData {
  name: string;
  description: string;
  baseUrl: string;
  apiKey: string;
  authToken: string;
  credentialMethod: CredentialMethod;
}

/**
 * Props for CredentialsSection component
 */
export interface CredentialsSectionProps {
  isRunnerHostDefault: boolean;
  credentialProfiles: CredentialProfileData[];
  onSetRunnerHostDefault: () => Promise<void>;
  onSetDefault: (profileId: number) => Promise<void>;
  onEdit: (profile: CredentialProfileData) => void;
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

/**
 * Props for CredentialDialog component
 */
export interface CredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingProfile: CredentialProfileData | null;
  onSubmit: (data: CredentialFormData, editingProfile: CredentialProfileData | null) => Promise<void>;
  t: (key: string) => string;
}
