import { getExtensionService } from "@/lib/wasm-core";
export type {
  SkillRegistryAuthType, SkillRegistry, SkillRegistryOverride,
  SkillMarketItem, McpMarketItem, McpHeaderSchemaEntry, EnvVarSchemaEntry,
  InstalledSkill, InstalledMcpServer,
} from "./extensionTypes";

export const extensionApi = {
  listRepoMcpServers: async (repoId: number, scope?: string) => {
    const json = await getExtensionService().list_repo_mcp_servers(BigInt(repoId), scope ?? null);
    return JSON.parse(json);
  },
  updateMcpServer: async (repoId: number, installId: number, data: Record<string, unknown>) => {
    const json = await getExtensionService().update_mcp_server(BigInt(repoId), BigInt(installId), JSON.stringify(data));
    return JSON.parse(json);
  },
  uninstallMcpServer: async (repoId: number, installId: number) => {
    await getExtensionService().uninstall_mcp_server(BigInt(repoId), BigInt(installId));
  },
  installSkillFromUpload: async (repoId: number, file: File, scope?: string) => {
    // Multipart upload stays REST forever — Connect-RPC doesn't handle multipart/form-data.
    const buf = new Uint8Array(await file.arrayBuffer());
    const json = await getExtensionService().install_skill_from_upload(BigInt(repoId), buf, file.name, scope ?? null);
    return JSON.parse(json);
  },
  installMcpFromMarket: async (repoId: number, data: Record<string, unknown>) => {
    const json = await getExtensionService().install_mcp_from_market(BigInt(repoId), JSON.stringify(data));
    return JSON.parse(json);
  },
  installCustomMcpServer: async (repoId: number, data: Record<string, unknown>) => {
    const json = await getExtensionService().install_custom_mcp_server(BigInt(repoId), JSON.stringify(data));
    return JSON.parse(json);
  },
};
