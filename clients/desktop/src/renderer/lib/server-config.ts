/**
 * Server selection for the Electron desktop renderer.
 *
 * Two-mode model:
 *   - "cloud": built-in pointer at app.agentsmesh.ai (fixed URL).
 *   - "custom": user-supplied URL + label, editable in-place.
 *
 * Persisted as a single object in localStorage. Schema is intentionally
 * forward-compatible: bump STORAGE_KEY to invalidate all clients when
 * the shape changes (the v1 list-based shape was discarded after the
 * UX shift; old keys are left orphaned in localStorage rather than
 * migrated, since they only held built-in pointers).
 */

const STORAGE_KEY = "agentsmesh.server_config_v2";

const CLOUD_URL = "https://app.agentsmesh.ai";
const CLOUD_LABEL = "AgentsMesh Cloud";

export type ServerKind = "cloud" | "custom";

export interface ServerConfig {
  kind: ServerKind;
  customLabel: string;
  customUrl: string;
}

const DEFAULT_CONFIG: ServerConfig = {
  kind: "cloud",
  customLabel: "",
  customUrl: "",
};

function readRaw(): ServerConfig | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<ServerConfig>;
    return {
      kind: parsed.kind === "custom" ? "custom" : "cloud",
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

export function getCloudInfo(): { label: string; url: string } {
  return { label: CLOUD_LABEL, url: CLOUD_URL };
}

/**
 * Resolves the URL the renderer should hit. Returns null when the
 * user hasn't made an explicit choice yet — env.ts then falls back
 * to the preload-bridge default (AGENTSMESH_API_URL the main process
 * was launched with), which is what dev / e2e / packaged users want
 * out of the box. Only returns a URL once the user has actively
 * picked one through the Server Settings dialog (saveConfig writes
 * the localStorage entry).
 *
 * In custom mode the URL is also nullable when the saved value is
 * malformed — env.ts again falls back rather than 404'ing every
 * request against an invalid origin.
 */
export function getActiveUrl(): string | null {
  const cfg = readRaw();
  if (!cfg) return null;
  if (cfg.kind === "cloud") return CLOUD_URL;
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
