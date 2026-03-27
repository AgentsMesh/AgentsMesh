import { request } from "./base";

// AgentPod types
export type AIProviderType = "claude" | "openai" | "gemini" | "codex";

export interface UserAgentPodSettings {
  id: number;
  user_id: number;
  preparation_script?: string;
  preparation_timeout: number;
  default_agent_slug?: string;
  default_model?: string;
  default_perm_mode?: string;
  terminal_font_size?: number;
  terminal_theme?: string;
  created_at: string;
  updated_at: string;
}

export interface UserAIProvider {
  id: number;
  user_id: number;
  provider_type: AIProviderType;
  name: string;
  is_default: boolean;
  is_enabled: boolean;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateSettingsRequest {
  preparation_script?: string;
  preparation_timeout?: number;
  default_model?: string;
  default_perm_mode?: string;
  terminal_font_size?: number;
  terminal_theme?: string;
}

export interface CreateProviderRequest {
  provider_type: AIProviderType;
  name: string;
  credentials: Record<string, string>;
  is_default?: boolean;
}

export interface UpdateProviderRequest {
  name?: string;
  credentials?: Record<string, string>;
  is_enabled?: boolean;
  is_default?: boolean;
}

// AgentPod API
export const agentpodApi = {
  // Settings
  getSettings: () =>
    request<{ settings: UserAgentPodSettings }>("/api/v1/users/me/agentpod/settings"),

  updateSettings: (data: UpdateSettingsRequest) =>
    request<{ settings: UserAgentPodSettings }>("/api/v1/users/me/agentpod/settings", {
      method: "PUT",
      body: data,
    }),

  // AI Providers
  listProviders: () =>
    request<{ providers: UserAIProvider[] }>("/api/v1/users/me/agentpod/providers"),

  createProvider: (data: CreateProviderRequest) =>
    request<{ provider: UserAIProvider }>("/api/v1/users/me/agentpod/providers", {
      method: "POST",
      body: data,
    }),

  updateProvider: (id: number, data: UpdateProviderRequest) =>
    request<{ provider: UserAIProvider }>(`/api/v1/users/me/agentpod/providers/${id}`, {
      method: "PUT",
      body: data,
    }),

  deleteProvider: (id: number) =>
    request<{ message: string }>(`/api/v1/users/me/agentpod/providers/${id}`, {
      method: "DELETE",
    }),

  setDefaultProvider: (id: number) =>
    request<{ provider: UserAIProvider }>(`/api/v1/users/me/agentpod/providers/${id}/default`, {
      method: "POST",
    }),
};
