import type { CredentialProfileData, AgentData, CredentialProfilesByAgent } from "@/lib/api";

/**
 * State returned by useAgentCredentials hook
 */
export interface AgentCredentialsState {
  loading: boolean;
  error: string | null;
  success: string | null;
  profilesByAgent: CredentialProfilesByAgent[];
  agents: AgentData[];
  expandedAgents: Set<string>;
  runnerHostDefaults: Set<string>;
}

/**
 * Actions returned by useAgentCredentials hook
 */
export interface AgentCredentialsActions {
  toggleAgent: (agentSlug: string) => void;
  handleSetRunnerHostDefault: (agentSlug: string) => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDelete: (profileId: number) => Promise<void>;
  handleSaveProfile: (
    agentSlug: string,
    data: CredentialFormData,
    editingProfile: CredentialProfileData | null
  ) => Promise<void>;
  getProfilesForAgent: (agentSlug: string) => CredentialProfileData[];
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
 * Props for AgentItem component
 */
export interface AgentItemProps {
  agent: AgentData;
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
