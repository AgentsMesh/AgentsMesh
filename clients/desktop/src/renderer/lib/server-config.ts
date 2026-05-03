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

function readRaw(): ServerConfig {
  if (typeof window === "undefined") return { ...DEFAULT_CONFIG };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...DEFAULT_CONFIG };
    const parsed = JSON.parse(raw) as Partial<ServerConfig>;
    return {
      kind: parsed.kind === "custom" ? "custom" : "cloud",
      customLabel: typeof parsed.customLabel === "string" ? parsed.customLabel : "",
      customUrl: typeof parsed.customUrl === "string" ? parsed.customUrl : "",
    };
  } catch {
    return { ...DEFAULT_CONFIG };
  }
}

function writeRaw(cfg: ServerConfig): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
}

export function getConfig(): ServerConfig {
  return readRaw();
}

export function getCloudInfo(): { label: string; url: string } {
  return { label: CLOUD_LABEL, url: CLOUD_URL };
}

/**
 * Resolves the URL the renderer should hit. Returns null when the
 * user is in custom mode but hasn't entered a valid URL yet — env.ts
 * falls back to its own default in that case so the app still loads
 * (the user fixes their config from the Server Settings dialog).
 */
export function getActiveUrl(): string | null {
  const cfg = readRaw();
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
