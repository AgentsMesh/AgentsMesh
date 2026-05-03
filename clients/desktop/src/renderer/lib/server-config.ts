/**
 * Server-list configuration for the Electron renderer.
 *
 * The desktop app talks to one AgentsMesh backend at a time (REST + WS).
 * Users (especially self-hosters) need to switch between dev / staging /
 * production / on-prem deployments without rebuilding. This module
 * persists the list + the active selection in localStorage and exposes
 * `getActiveUrl()` for env.ts to consult on every fetch resolution.
 *
 * Persistence shape stays small and forward-compatible: an array of
 * `Server` records plus a single `selectedId`. Built-in entries carry
 * `readonly: true` so the UI can show them un-deletable.
 */

const STORAGE_KEY = "agentsmesh.server_config_v1";

export interface Server {
  id: string;
  label: string;
  url: string;
  readonly?: boolean;
}

interface ServerConfig {
  selectedId: string;
  servers: Server[];
}

const BUILTIN_SERVERS: Server[] = [
  {
    id: "builtin-default",
    label: "AgentsMesh Cloud",
    url: "https://app.agentsmesh.ai",
    readonly: true,
  },
  {
    id: "builtin-localhost",
    // Match deploy/dev/.env's HTTP_PORT default. Power users with
    // worktree-offset ports edit this entry by adding a second one
    // pointing at their actual port — the built-in stays as a sane
    // first-time default.
    label: "Localhost (dev)",
    url: "http://localhost:10000",
    readonly: true,
  },
];

function readRaw(): ServerConfig | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as ServerConfig;
    if (!parsed || !Array.isArray(parsed.servers) || typeof parsed.selectedId !== "string") {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function writeRaw(cfg: ServerConfig): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
}

function ensureConfig(): ServerConfig {
  const existing = readRaw();
  if (existing) {
    // Re-merge built-ins so a code-side rename / addition is reflected
    // without nuking the user's custom entries. Match by id.
    const userOnly = existing.servers.filter((s) => !BUILTIN_SERVERS.some((b) => b.id === s.id));
    const merged: ServerConfig = {
      selectedId: existing.selectedId,
      servers: [...BUILTIN_SERVERS, ...userOnly],
    };
    if (!merged.servers.some((s) => s.id === merged.selectedId)) {
      merged.selectedId = BUILTIN_SERVERS[0].id;
    }
    return merged;
  }
  return { selectedId: BUILTIN_SERVERS[0].id, servers: [...BUILTIN_SERVERS] };
}

export function listServers(): Server[] {
  return ensureConfig().servers;
}

export function getSelectedId(): string {
  return ensureConfig().selectedId;
}

export function getActiveUrl(): string | null {
  const cfg = ensureConfig();
  const active = cfg.servers.find((s) => s.id === cfg.selectedId);
  return active ? active.url : null;
}

export function selectServer(id: string): void {
  const cfg = ensureConfig();
  if (!cfg.servers.some((s) => s.id === id)) return;
  cfg.selectedId = id;
  writeRaw(cfg);
}

export function addServer(input: { label: string; url: string }): Server {
  const cfg = ensureConfig();
  const id = `user-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const server: Server = { id, label: input.label.trim(), url: input.url.trim() };
  cfg.servers.push(server);
  cfg.selectedId = id;
  writeRaw(cfg);
  return server;
}

export function updateServer(id: string, patch: Partial<Pick<Server, "label" | "url">>): void {
  const cfg = ensureConfig();
  const target = cfg.servers.find((s) => s.id === id);
  if (!target || target.readonly) return;
  if (patch.label !== undefined) target.label = patch.label.trim();
  if (patch.url !== undefined) target.url = patch.url.trim();
  writeRaw(cfg);
}

export function removeServer(id: string): void {
  const cfg = ensureConfig();
  const target = cfg.servers.find((s) => s.id === id);
  if (!target || target.readonly) return;
  cfg.servers = cfg.servers.filter((s) => s.id !== id);
  if (cfg.selectedId === id) {
    cfg.selectedId = cfg.servers[0]?.id ?? BUILTIN_SERVERS[0].id;
  }
  writeRaw(cfg);
}

export function isValidServerUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}
