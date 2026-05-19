export type SkillRegistryAuthType = "none" | "github_pat" | "gitlab_pat" | "ssh_key";

export interface SkillRegistry {
  id: number;
  organization_id: number | null;
  repository_url: string;
  branch: string;
  source_type: string;
  detected_type: string;
  compatible_agents?: string[];  // agent whitelist, e.g. ["claude-code"]
  auth_type: SkillRegistryAuthType;
  // auth_credential is never returned from the API (json:"-" in Go)
  last_synced_at: string | null;
  sync_status: string;
  sync_error: string;
  skill_count: number;
  is_active: boolean;
}

export interface SkillMarketItem {
  id: number;
  registry_id: number;
  slug: string;
  display_name: string;
  description: string;
  license: string;
  category: string;
  content_sha: string;
  version: number;
  is_active: boolean;
  registry?: SkillRegistry;
}

export interface McpMarketItem {
  id: number;
  slug: string;
  name: string;
  description: string;
  icon: string;
  transport_type: string;
  command: string;
  default_args?: string[] | null;
  default_http_url?: string;
  default_http_headers?: McpHeaderSchemaEntry[] | null;
  env_var_schema?: EnvVarSchemaEntry[] | null;
  category: string;
  source?: string;
  registry_name?: string;
  version?: string;
  repository_url?: string;
}

export interface McpHeaderSchemaEntry {
  name: string;
  description?: string;
  value?: string;
  required: boolean;
  sensitive: boolean;
}

export interface EnvVarSchemaEntry {
  name: string;
  label: string;
  required: boolean;
  sensitive: boolean;
  placeholder?: string;
}

export interface SkillRegistryOverride {
  id: number;
  organization_id: number;
  registry_id: number;
  is_disabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface InstalledSkill {
  id: number;
  organization_id: number;
  repository_id: number;
  market_item_id: number | null;
  scope: "org" | "user";
  installed_by: number;
  slug: string;
  install_source: "market" | "github" | "upload";
  source_url: string;
  content_sha: string;
  package_size: number;
  pinned_version: number | null;
  is_enabled: boolean;
  market_item?: SkillMarketItem;
}

export interface InstalledMcpServer {
  id: number;
  organization_id: number;
  repository_id: number;
  market_item_id: number | null;
  scope: "org" | "user";
  installed_by: number;
  name: string;
  slug: string;
  transport_type: string;
  command: string;
  args?: string[] | null;
  http_url: string;
  http_headers?: Record<string, string> | null;
  env_vars: Record<string, string>;
  is_enabled: boolean;
  market_item?: McpMarketItem;
}
