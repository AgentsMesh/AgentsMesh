// Shared between main / preload / renderer. Pure data + pure helpers,
// NO i/o, NO process-specific imports — so it compiles cleanly in all
// three Electron contexts (node main, sandboxed preload, browser
// renderer). Edit a preset URL or the validation rule here and all
// three boundaries update together.

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

// HTTP/HTTPS URL validator. Lives here because every trust boundary that
// accepts a server URL — env override (main), persisted custom URL
// (main), dialog input (renderer) — must agree on what counts as valid.
// Splitting the rule across processes was how `app.agentsmesh.ai` slipped
// through pre-PR-#336.
export function isValidServerUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}
