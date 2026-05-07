/**
 * Server selection for the Electron desktop renderer.
 *
 * Three modes:
 *   - "global": built-in pointer at agentsmesh.ai (production global).
 *   - "cn":     built-in pointer at agentsmesh.cn (production China).
 *   - "custom": user-supplied URL + label.
 *
 * Persisted as a single object in localStorage. Schema is forward-
 * compatible: bump STORAGE_KEY to invalidate clients on shape break.
 *
 * Legacy migration: v0.30.x and earlier shipped a single "cloud"
 * preset pointing at app.agentsmesh.ai (which never existed in
 * production). We silently treat any saved kind: "cloud" as
 * kind: "global" and drop the bad app. host. The localStorage
 * record is left in v2 form until the user next saves through the
 * dialog — readRaw normalises it on the way out, no eager rewrite.
 */
const STORAGE_KEY = "agentsmesh.server_config_v2";

const PRESETS: Record<"global" | "cn", { label: string; url: string }> = {
  global: { label: "AgentsMesh Global", url: "https://agentsmesh.ai" },
  cn: { label: "AgentsMesh 中国", url: "https://agentsmesh.cn" },
};

export type ServerKind = "global" | "cn" | "custom";

export interface ServerConfig {
  kind: ServerKind;
  customLabel: string;
  customUrl: string;
}

const DEFAULT_CONFIG: ServerConfig = {
  kind: "global",
  customLabel: "",
  customUrl: "",
};

function normaliseKind(raw: unknown): ServerKind {
  if (raw === "cn" || raw === "custom") return raw;
  // "cloud" is the legacy alias for "global"; anything else falls
  // through to the safe default.
  return "global";
}

function readRaw(): ServerConfig | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<ServerConfig> & { kind?: unknown };
    return {
      kind: normaliseKind(parsed.kind),
      customLabel: typeof parsed.customLabel === "string" ? parsed.customLabel : "",
      customUrl: typeof parsed.customUrl === "string" ? parsed.customUrl : "",
    };
  } catch {
    return null;
  }
}

function writeRaw(cfg: ServerConfig): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
}

export function getConfig(): ServerConfig {
  return readRaw() ?? { ...DEFAULT_CONFIG };
}

export function getPresets(): { kind: "global" | "cn"; label: string; url: string }[] {
  return [
    { kind: "global", ...PRESETS.global },
    { kind: "cn", ...PRESETS.cn },
  ];
}

/**
 * Human-readable label for the currently selected server. The footer
 * pill in AuthShell uses this to show the user what they're connected
 * to. Custom mode falls back to the preset name when the user hasn't
 * supplied a label, so we never render an empty pill.
 */
export function getActiveLabel(cfg: ServerConfig): string {
  if (cfg.kind === "custom" && cfg.customLabel.trim()) return cfg.customLabel;
  if (cfg.kind === "cn") return PRESETS.cn.label;
  return PRESETS.global.label;
}

/**
 * Resolves the URL the renderer should hit. Returns null when no
 * explicit choice has been made yet — env.ts then falls back to the
 * preload-bridge default (AGENTSMESH_API_URL the main process was
 * launched with), which is what dev / e2e / packaged users want
 * out of the box. Only returns a URL once the user has actively
 * picked one through the Server Settings dialog.
 *
 * In custom mode the URL is also nullable when the saved value is
 * malformed — env.ts again falls back rather than 404'ing every
 * request against an invalid origin.
 */
export function getActiveUrl(): string | null {
  const cfg = readRaw();
  if (!cfg) return null;
  if (cfg.kind === "global") return PRESETS.global.url;
  if (cfg.kind === "cn") return PRESETS.cn.url;
  if (cfg.kind === "custom" && isValidServerUrl(cfg.customUrl)) return cfg.customUrl;
  return null;
}

export function saveConfig(next: ServerConfig): void {
  writeRaw({
    kind: next.kind,
    customLabel: next.customLabel.trim(),
    customUrl: next.customUrl.trim(),
  });
}

export function isValidServerUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}
