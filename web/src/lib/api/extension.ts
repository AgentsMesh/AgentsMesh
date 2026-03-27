import { request, orgPath } from "./base";
import {
  uploadRequest,
} from "./extensionTypes";
import type {
  SkillRegistryAuthType,
  SkillRegistry,
  SkillRegistryOverride,
  SkillMarketItem,
  McpMarketItem,
  InstalledSkill,
  InstalledMcpServer,
} from "./extensionTypes";

// Re-export types for consumers
export type { SkillRegistryAuthType } from "./extensionTypes";
export type {
  SkillRegistry,
  SkillMarketItem,
  McpMarketItem,
  McpHeaderSchemaEntry,
  EnvVarSchemaEntry,
  SkillRegistryOverride,
  InstalledSkill,
  InstalledMcpServer,
} from "./extensionTypes";

export const extensionApi = {
  // Skill Registries (org admin)
  listSkillRegistries: () =>
    request<{ skill_registries: SkillRegistry[] }>(orgPath("/skill-registries")),

  createSkillRegistry: (data: {
    repository_url: string;
    branch?: string;
    source_type?: string;
    compatible_agents?: string[];
    auth_type?: SkillRegistryAuthType;
    auth_credential?: string;
  }) =>
    request<{ registry: SkillRegistry }>(orgPath("/skill-registries"), { method: "POST", body: data }),

  syncSkillRegistry: (id: number) =>
    request<{ registry: SkillRegistry }>(orgPath(`/skill-registries/${id}/sync`), { method: "POST" }),

  deleteSkillRegistry: (id: number) =>
    request(orgPath(`/skill-registries/${id}`), { method: "DELETE" }),

  // Skill Registry Overrides
  togglePlatformRegistry: (registryId: number, disabled: boolean) =>
    request<{ overrides: SkillRegistryOverride[] }>(orgPath(`/skill-registries/${registryId}/toggle`), {
      method: "PUT",
      body: { disabled },
    }),

  listSkillRegistryOverrides: () =>
    request<{ overrides: SkillRegistryOverride[] }>(orgPath("/skill-registry-overrides")),

  // Marketplace
  listMarketSkills: (query?: string, category?: string) => {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (category) params.set("category", category);
    const qs = params.toString();
    return request<{ skills: SkillMarketItem[] }>(orgPath(`/market/skills${qs ? `?${qs}` : ""}`));
  },

  listMarketMcpServers: (query?: string, category?: string, limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (category) params.set("category", category);
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    const qs = params.toString();
    return request<{ mcp_servers: McpMarketItem[]; total: number; limit: number; offset: number }>(orgPath(`/market/mcp-servers${qs ? `?${qs}` : ""}`));
  },

  // Repo Skills
  listRepoSkills: (repoId: number, scope?: string) => {
    const qs = scope ? `?scope=${scope}` : "";
    return request<{ skills: InstalledSkill[] }>(orgPath(`/repositories/${repoId}/skills${qs}`));
  },

  installSkillFromMarket: (repoId: number, data: { market_item_id: number; scope: string }) =>
    request<{ skill: InstalledSkill }>(orgPath(`/repositories/${repoId}/skills/install-from-market`), {
      method: "POST",
      body: data,
    }),

  installSkillFromGitHub: (repoId: number, data: { url: string; branch?: string; path?: string; scope: string }) =>
    request<{ skill: InstalledSkill }>(orgPath(`/repositories/${repoId}/skills/install-from-github`), {
      method: "POST",
      body: data,
    }),

  installSkillFromUpload: (repoId: number, file: File, scope: string) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("scope", scope);
    return uploadRequest<{ skill: InstalledSkill }>(orgPath(`/repositories/${repoId}/skills/install-from-upload`), formData);
  },

  updateSkill: (repoId: number, installId: number, data: { is_enabled?: boolean; pinned_version?: number | null }) =>
    request<{ skill: InstalledSkill }>(orgPath(`/repositories/${repoId}/skills/${installId}`), {
      method: "PUT",
      body: data,
    }),

  uninstallSkill: (repoId: number, installId: number) =>
    request(orgPath(`/repositories/${repoId}/skills/${installId}`), { method: "DELETE" }),

  // Repo MCP Servers
  listRepoMcpServers: (repoId: number, scope?: string) => {
    const qs = scope ? `?scope=${scope}` : "";
    return request<{ mcp_servers: InstalledMcpServer[] }>(orgPath(`/repositories/${repoId}/mcp-servers${qs}`));
  },

  installMcpFromMarket: (
    repoId: number,
    data: { market_item_id: number; env_vars?: Record<string, string>; scope: string }
  ) =>
    request<{ mcp_server: InstalledMcpServer }>(orgPath(`/repositories/${repoId}/mcp-servers/install-from-market`), {
      method: "POST",
      body: data,
    }),

  installCustomMcpServer: (
    repoId: number,
    data: {
      name: string;
      slug: string;
      transport_type: string;
      command?: string;
      args?: string[];
      http_url?: string;
      http_headers?: Record<string, string>;
      env_vars?: Record<string, string>;
      scope: string;
    }
  ) =>
    request<{ mcp_server: InstalledMcpServer }>(orgPath(`/repositories/${repoId}/mcp-servers/install-custom`), {
      method: "POST",
      body: data,
    }),

  updateMcpServer: (
    repoId: number,
    installId: number,
    data: { is_enabled?: boolean; env_vars?: Record<string, string> }
  ) =>
    request<{ mcp_server: InstalledMcpServer }>(orgPath(`/repositories/${repoId}/mcp-servers/${installId}`), {
      method: "PUT",
      body: data,
    }),

  uninstallMcpServer: (repoId: number, installId: number) =>
    request(orgPath(`/repositories/${repoId}/mcp-servers/${installId}`), { method: "DELETE" }),
};
