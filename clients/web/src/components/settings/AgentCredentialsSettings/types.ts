import type { AgentData } from "@/lib/api";
import type { CredentialProfileViewModel, CredentialProfilesByAgent } from "../_shared/credentialViewModel";
import type { CredentialFormData } from "./credentialForms/types";

// Re-export: CredentialFormData is defined once in credentialForms/types.
export type { CredentialFormData };

export interface AgentCredentialsState {
  loading: boolean;
  error: string | null;
  success: string | null;
  profilesByAgent: CredentialProfilesByAgent[];
  agents: AgentData[];
  expandedAgents: Set<string>;
  agentsWithoutPrimaryBundle: Set<string>;
}

export interface AgentCredentialsActions {
  toggleAgent: (agentSlug: string) => void;
  handleClearPrimaryBundle: (agentSlug: string) => Promise<void>;
  handleSetDefault: (profileId: number) => Promise<void>;
  handleDelete: (profileId: number) => Promise<void>;
  handleSaveProfile: (
    agentSlug: string,
    data: CredentialFormData,
    editingProfile: CredentialProfileViewModel | null
  ) => Promise<void>;
  getProfilesForAgent: (agentSlug: string) => CredentialProfileViewModel[];
  setError: (error: string | null) => void;
  setSuccess: (success: string | null) => void;
}

export interface AgentItemProps {
  agent: AgentData;
  profiles: CredentialProfileViewModel[];
  isExpanded: boolean;
  noPrimaryBundle: boolean;
  onToggle: () => void;
  onClearPrimary: () => Promise<void>;
  onSetDefault: (profileId: number) => Promise<void>;
  onEdit: (profile: CredentialProfileViewModel) => void;
  onDelete: (profileId: number) => Promise<void>;
  onAdd: () => void;
  t: (key: string) => string;
}
