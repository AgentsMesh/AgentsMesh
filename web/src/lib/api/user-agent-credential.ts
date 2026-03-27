import { request } from "./base";

// Agent Credential Profile types
export interface CredentialProfileData {
  id: number;
  user_id: number;
  agent_slug: string;
  name: string;
  description?: string;
  is_runner_host: boolean;
  is_default: boolean;
  is_active: boolean;
  configured_fields?: string[];
  configured_values?: Record<string, string>;
  agent_name?: string;
  created_at: string;
  updated_at: string;
}

export interface CredentialProfilesByAgent {
  agent_slug: string;
  agent_name: string;
  profiles: CredentialProfileData[];
}

export interface CreateCredentialProfileRequest {
  name: string;
  description?: string;
  is_runner_host: boolean;
  credentials?: Record<string, string>;
  is_default?: boolean;
}

export interface UpdateCredentialProfileRequest {
  name?: string;
  description?: string;
  is_runner_host?: boolean;
  credentials?: Record<string, string>;
  is_default?: boolean;
  is_active?: boolean;
}

export interface RunnerHostInfo {
  available: boolean;
  description: string;
}

// User Agent Credential Profile API
export const userAgentCredentialApi = {
  // List all credential profiles grouped by agent
  list: () =>
    request<{
      items: CredentialProfilesByAgent[];
    }>("/api/v1/users/agent-credentials"),

  // List credential profiles for a specific agent
  listForAgent: (agentSlug: string) =>
    request<{
      profiles: CredentialProfileData[];
      runner_host: RunnerHostInfo;
    }>(`/api/v1/users/agent-credentials/agents/${agentSlug}`),

  // Create a new credential profile
  create: (agentSlug: string, data: CreateCredentialProfileRequest) =>
    request<{ profile: CredentialProfileData }>(
      `/api/v1/users/agent-credentials/agents/${agentSlug}`,
      {
        method: "POST",
        body: data,
      }
    ),

  // Get a single credential profile
  get: (profileId: number) =>
    request<{ profile: CredentialProfileData }>(
      `/api/v1/users/agent-credentials/profiles/${profileId}`
    ),

  // Update a credential profile
  update: (profileId: number, data: UpdateCredentialProfileRequest) =>
    request<{ profile: CredentialProfileData }>(
      `/api/v1/users/agent-credentials/profiles/${profileId}`,
      {
        method: "PUT",
        body: data,
      }
    ),

  // Delete a credential profile
  delete: (profileId: number) =>
    request<{ message: string }>(
      `/api/v1/users/agent-credentials/profiles/${profileId}`,
      {
        method: "DELETE",
      }
    ),

  // Set a profile as default for its agent
  setDefault: (profileId: number) =>
    request<{ message: string; profile: CredentialProfileData }>(
      `/api/v1/users/agent-credentials/profiles/${profileId}/set-default`,
      {
        method: "POST",
      }
    ),
};

// Helper function to check if a profile uses RunnerHost mode
export function isRunnerHostProfile(profile: CredentialProfileData): boolean {
  return profile.is_runner_host;
}

// Helper function to get display status for a profile
export function getProfileStatusLabel(profile: CredentialProfileData): string {
  if (profile.is_runner_host) {
    return "RunnerHost";
  }
  if (profile.configured_fields && profile.configured_fields.length > 0) {
    return "Configured";
  }
  return "Not configured";
}
