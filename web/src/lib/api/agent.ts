import { request, orgPath } from "./base";

// Agent type interface
export interface AgentTypeData {
  id: number;
  slug: string;
  name: string;
  description?: string;
  launch_command?: string;
  is_builtin: boolean;
  is_active: boolean;
}

// Config field option for select type (value only, label from frontend i18n)
export interface ConfigFieldOption {
  value: string;
}

// Config field definition from Backend (raw, without i18n labels)
// Frontend is responsible for i18n using: agent.{slug}.fields.{name}.label
export interface ConfigField {
  name: string;
  type: "boolean" | "string" | "select" | "number" | "secret";
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
  agent_type_id: number;
  agent_type_name?: string;
  agent_type_slug?: string;
  config_values: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// Agents API
export const agentApi = {
  listTypes: async () => {
    const response = await request<{
      builtin_types: AgentTypeData[];
      custom_types: AgentTypeData[];
    }>(orgPath("/agents/types"));
    // Combine builtin and custom types for frontend compatibility
    return {
      agent_types: [...(response.builtin_types || []), ...(response.custom_types || [])],
    };
  },

  // Get config schema for an agent type (raw, frontend handles i18n)
  getConfigSchema: (agentTypeId: number) => {
    return request<{ schema: ConfigSchema }>(`${orgPath("/agents")}/${agentTypeId}/config-schema`);
  },
};

// User Agent Config API (personal runtime configuration)
export const userAgentConfigApi = {
  // List all personal configs for the current user
  list: () =>
    request<{ configs: UserAgentConfigData[] }>("/api/v1/users/me/agent-configs"),

  // Get user's personal config for a specific agent type
  get: (agentTypeId: number) =>
    request<{ config: UserAgentConfigData }>(`/api/v1/users/me/agent-configs/${agentTypeId}`),

  // Set/update user's personal config for an agent type
  set: (agentTypeId: number, configValues: Record<string, unknown>) =>
    request<{ config: UserAgentConfigData }>(`/api/v1/users/me/agent-configs/${agentTypeId}`, {
      method: "PUT",
      body: { config_values: configValues },
    }),

  // Delete user's personal config for an agent type
  delete: (agentTypeId: number) =>
    request<{ message: string }>(`/api/v1/users/me/agent-configs/${agentTypeId}`, {
      method: "DELETE",
    }),
};
