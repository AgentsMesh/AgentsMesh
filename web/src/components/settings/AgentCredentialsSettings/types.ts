import type { CredentialProfileData, AgentData, CredentialProfilesByAgent, CredentialField } from "@/lib/api";

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
  credentialFieldsByAgent: Map<string, CredentialField[]>;
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
 * Credential form data for add/edit dialog.
 * credentials key = full ENV name (e.g. "ANTHROPIC_API_KEY"), value = user input.
 */
export interface CredentialFormData {
  name: string;
  description: string;
  credentials: Record<string, string>;
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
 * Props for CredentialProfileDialog component
 */
export interface CredentialProfileDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credentialFields: CredentialField[];
  editingProfile: CredentialProfileData | null;
  onSubmit: (data: CredentialFormData) => Promise<void>;
  t: (key: string) => string;
}
