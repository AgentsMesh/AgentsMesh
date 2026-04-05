import type { ConfigField, AgentData, CredentialProfileData, CredentialField } from "@/lib/api";

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
  credentialFields: CredentialField[];
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
 * Credential form data for add/edit dialog.
 * credentials key = full ENV name (e.g. "ANTHROPIC_API_KEY"), value = user input.
 */
export interface CredentialFormData {
  name: string;
  description: string;
  credentials: Record<string, string>;
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
  credentialFields: CredentialField[];
  editingProfile: CredentialProfileData | null;
  onSubmit: (data: CredentialFormData, editingProfile: CredentialProfileData | null) => Promise<void>;
  t: (key: string) => string;
}
