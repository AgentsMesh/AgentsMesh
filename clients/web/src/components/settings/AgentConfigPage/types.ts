import type { ConfigField, AgentData, CredentialProfileData, CredentialField } from "@/lib/api";

export interface AgentConfigPageProps {
  agentSlug: string;
}

export interface AgentConfigState {
  loading: boolean;
  savingConfig: boolean;

  agent: AgentData | null;
  configFields: ConfigField[];
  configValues: Record<string, unknown>;
  credentialFields: CredentialField[];
  credentialProfiles: CredentialProfileData[];
  isRunnerHostDefault: boolean;

  error: string | null;
  success: string | null;
}

export interface AgentConfigActions {
  handleConfigChange: (fieldName: string, value: unknown) => void;
  handleSaveConfig: () => Promise<void>;

  handleSetRunnerHostDefault: () => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDeleteProfile: (profileId: number) => Promise<void>;
  handleSaveProfile: (data: CredentialFormData, editingProfile: CredentialProfileData | null) => Promise<void>;

  setError: (error: string | null) => void;
  setSuccess: (success: string | null) => void;
  loadData: () => Promise<void>;
}

export interface CredentialFormData {
  name: string;
  description: string;
  credentials: Record<string, string>;
}

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

export interface RuntimeConfigSectionProps {
  configFields: ConfigField[];
  configValues: Record<string, unknown>;
  agentSlug: string;
  saving: boolean;
  onChange: (fieldName: string, value: unknown) => void;
  onSave: () => Promise<void>;
  t: (key: string) => string;
}

export interface CredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credentialFields: CredentialField[];
  editingProfile: CredentialProfileData | null;
  onSubmit: (data: CredentialFormData, editingProfile: CredentialProfileData | null) => Promise<void>;
  t: (key: string) => string;
}
