/**
 * Agent ViewModels — UI-side projections of `proto.agent.v1.*`.
 *
 * `AgentData` mirrors `proto.Agent` but keeps snake_case + parsed `default`
 * for the existing form/registry components. `ConfigField` extends the wire
 * shape with UI-only fields (`required`, `validation`, `show_when`) that the
 * REST shape used to carry; the proto wire layer does not, so these stay
 * undefined at runtime but the renderer paths still typecheck.
 *
 * New components should consume `proto.agent.v1` types directly.
 */
export interface AgentData {
  slug: string;
  name: string;
  description?: string;
  launch_command?: string;
  is_builtin: boolean;
  is_active: boolean;
  supported_modes?: string | string[];
  // Package manager for remote upgrade ("npm"/"pip"); absent = not upgradable.
  upgrade_manager?: string;
}

export interface ConfigFieldOption {
  value: string;
}

export interface ConfigField {
  name: string;
  type: "boolean" | "string" | "select" | "number" | "secret" | "model_list";
  default?: unknown;
  options?: ConfigFieldOption[];
  required?: boolean;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    min_length?: number;
    max_length?: number;
  };
  show_when?: {
    field: string;
    operator: string;
    value?: unknown;
  };
}

export interface ConfigSchema {
  fields: ConfigField[];
  credential_fields?: CredentialField[];
}

export interface CredentialField {
  name: string;
  type: "secret" | "text";
  optional: boolean;
}

export interface UserAgentConfigData {
  id: number;
  user_id: number;
  agent_slug: string;
  agent_name?: string;
  config_values: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}
