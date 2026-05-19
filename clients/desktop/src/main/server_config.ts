import { app } from "electron";
import fs from "fs";
import path from "path";
import {
  PRESETS,
  DEFAULT_SERVER_CONFIG,
  isValidServerUrl,
  type ServerConfig,
  type ServerKind,
} from "../shared/server-config-types";

// SSOT owned by main. Renderer reads sync IPC snapshot at preload; writes via
// `serverConfig:set` → AppState rebuild + window reload. Persisted at userData/server.json.
// Must live in main: (1) Rust ApiClient binds base_url at construction — renderer SSOT
// would mean me/refresh hit the previous host (2026-05-10 bug); (2) preload+main need it
// before renderer boots, localStorage is too late.
// Pure module: no env reads, no global mutation, no dialog calls — those live in main/index.ts.

export type { ServerConfig, ServerKind };
export { PRESETS, DEFAULT_SERVER_CONFIG as DEFAULT };

const FILE_NAME = "server.json";

function configPath(): string {
  return path.join(app.getPath("userData"), FILE_NAME);
}

function normaliseKind(raw: unknown): ServerKind {
  if (raw === "cn" || raw === "custom") return raw;
  return "global"; // legacy "cloud" alias + safe default
}

function normaliseStringField(raw: unknown): string {
  return typeof raw === "string" ? raw.trim() : "";
}

function readFromDisk(): ServerConfig | null {
  try {
    const raw = fs.readFileSync(configPath(), "utf8");
    const parsed = JSON.parse(raw) as Partial<ServerConfig> & { kind?: unknown };
    return {
      kind: normaliseKind(parsed.kind),
      customLabel: normaliseStringField(parsed.customLabel),
      customUrl: normaliseStringField(parsed.customUrl),
    };
  } catch {
    return null;
  }
}

export function load(): ServerConfig {
  return readFromDisk() ?? { ...DEFAULT_SERVER_CONFIG };
}

// Returns normalised cfg actually written — callers MUST use this for in-memory state
// to keep disk/memory byte-identical (passing raw cfg lets untrimmed fields linger).
export function save(cfg: ServerConfig): ServerConfig {
  const next: ServerConfig = {
    kind: normaliseKind(cfg.kind),
    customLabel: normaliseStringField(cfg.customLabel),
    customUrl: normaliseStringField(cfg.customUrl),
  };
  const tmp = configPath() + ".tmp";
  fs.writeFileSync(tmp, JSON.stringify(next), "utf8");
  fs.renameSync(tmp, configPath()); // atomic on same filesystem
  return next;
}

// Throws on invalid custom config — caller MUST surface error rather than fall back to DEFAULT.
export function activeUrl(cfg: ServerConfig): string {
  if (cfg.kind === "global") return PRESETS.global.url;
  if (cfg.kind === "cn") return PRESETS.cn.url;
  if (isValidServerUrl(cfg.customUrl)) return cfg.customUrl;
  throw new Error(`Invalid custom server URL: "${cfg.customUrl}"`);
}
