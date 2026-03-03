import type { CredentialProfileData, AgentTypeData, CredentialProfilesByAgentType } from "@/lib/api";

/**
 * State returned by useAgentCredentials hook
 */
export interface AgentCredentialsState {
  loading: boolean;
  error: string | null;
  success: string | null;
  profilesByAgentType: CredentialProfilesByAgentType[];
  agentTypes: AgentTypeData[];
  expandedAgentTypes: Set<number>;
  runnerHostDefaults: Set<number>;
}

/**
 * Actions returned by useAgentCredentials hook
 */
export interface AgentCredentialsActions {
  toggleAgentType: (agentTypeId: number) => void;
  handleSetRunnerHostDefault: (agentTypeId: number) => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDelete: (profileId: number) => Promise<void>;
  handleSaveProfile: (
    agentTypeId: number,
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => Promise<void>;
  getProfilesForAgentType: (agentTypeId: number) => CredentialProfileData[];
  setError: (error: string | null) => void;
  setSuccess: (success: string | null) => void;
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
 * Props for AgentTypeItem component
 */
export interface AgentTypeItemProps {
  agentType: AgentTypeData;
  profiles: CredentialProfileData[];
  isExpanded: boolean;
  isRunnerHostDefault: boolean;
  onToggle: () => void;
  onSetRunnerHostDefault: () => Promise<void>;
  onSetDefault: (profileId: number) => Promise<void>;
  onEdit: (profile: CredentialProfileData) => void;
  onDelete: (profileId: number) => Promise<void>;
  onAdd: () => void;
  t: (key: string) => string;
}

/**
 * Props for CredentialProfileDialog component (shared)
 */
export interface CredentialProfileDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingProfile: CredentialProfileData | null;
  onSubmit: (data: CredentialFormData) => Promise<void>;
  t: (key: string) => string;
}
