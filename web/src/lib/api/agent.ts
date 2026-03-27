import { request, orgPath } from "./base";

// Agent interface (slug is the primary key, no numeric id)
export interface AgentData {
  slug: string;
  name: string;
  description?: string;
  launch_command?: string;
  is_builtin: boolean;
  is_active: boolean;
  supported_modes?: string; // comma-separated: "pty", "pty,acp"
}

// Config field option for select type (value only, label from frontend i18n)
export interface ConfigFieldOption {
  value: string;
}

// Config field definition from Backend (raw, without i18n labels)
// Frontend is responsible for i18n using: agent.{slug}.fields.{name}.label
export interface ConfigField {
  name: string;
  type: "boolean" | "string" | "select" | "number" | "secret" | "model_list";
  default?: unknown;
  options?: ConfigFieldOption[];
  required?: boolean;
  // Validation rules (optional)
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    min_length?: number;
    max_length?: number;
  };
  // Conditional display
  show_when?: {
    field: string;
    operator: string;
    value?: unknown;
  };
}

// Config schema returned by Backend (raw, without i18n labels)
export interface ConfigSchema {
  fields: ConfigField[];
}

// User agent config interface (personal runtime configuration)
export interface UserAgentConfigData {
  id: number;
  user_id: number;
  agent_slug: string;
  agent_name?: string;
  config_values: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// Agents API
export const agentApi = {
  list: async () => {
    const response = await request<{
      builtin_agents: AgentData[];
      custom_agents: AgentData[];
    }>(orgPath("/agents"));
    // Combine builtin and custom agents for frontend compatibility
    return {
      agents: [...(response.builtin_agents || []), ...(response.custom_agents || [])],
    };
  },

  // Get config schema for an agent (raw, frontend handles i18n)
  getConfigSchema: (agentSlug: string) => {
    return request<{ schema: ConfigSchema }>(`${orgPath("/agents")}/${agentSlug}/config-schema`);
  },
};

// User Agent Config API (personal runtime configuration)
export const userAgentConfigApi = {
  // List all personal configs for the current user
  list: () =>
    request<{ configs: UserAgentConfigData[] }>("/api/v1/users/me/agent-configs"),

  // Get user's personal config for a specific agent
  get: (agentSlug: string) =>
    request<{ config: UserAgentConfigData }>(`/api/v1/users/me/agent-configs/${agentSlug}`),

  // Set/update user's personal config for an agent
  set: (agentSlug: string, configValues: Record<string, unknown>) =>
    request<{ config: UserAgentConfigData }>(`/api/v1/users/me/agent-configs/${agentSlug}`, {
      method: "PUT",
      body: { config_values: configValues },
    }),

  // Delete user's personal config for an agent
  delete: (agentSlug: string) =>
    request<{ message: string }>(`/api/v1/users/me/agent-configs/${agentSlug}`, {
      method: "DELETE",
    }),
};
