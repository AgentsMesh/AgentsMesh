// Shared by main / preload / renderer. Pure data + helpers, NO i/o, NO process-specific imports.
// Edit a preset URL or the validation rule here — all 3 boundaries update together.

export type ServerKind = "global" | "cn" | "custom";

export interface ServerConfig {
  kind: ServerKind;
  customLabel: string;
  customUrl: string;
}

export const PRESETS: Record<"global" | "cn", { label: string; url: string }> = {
  global: { label: "AgentsMesh Global", url: "https://agentsmesh.ai" },
  cn: { label: "AgentsMesh 中国", url: "https://agentsmesh.cn" },
};

export const DEFAULT_SERVER_CONFIG: ServerConfig = {
  kind: "global",
  customLabel: "",
  customUrl: "",
};

// Cross-boundary rule: env override (main), persisted custom URL (main), dialog input (renderer)
// must agree on what counts as valid. PR #336 fixed `app.agentsmesh.ai` slipping through a split rule.
export function isValidServerUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}
