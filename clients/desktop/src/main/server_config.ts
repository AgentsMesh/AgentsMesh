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

// SSOT for which AgentsMesh server this process talks to. Owned by main —
// renderer reads a snapshot via sync IPC at preload time, writes via the
// `serverConfig:set` IPC handler which triggers AppState rebuild + window
// reload. Persisted at `userData/server.json` so the choice survives
// restarts.
//
// Two reasons this lives in main, not renderer:
//   1. Rust core's ApiClient binds base_url at construction. If renderer
//      held the SSOT, main wouldn't see switches → me/refresh requests would
//      hit the previous host (the bug we found 2026-05-10).
//   2. preload + main both need it BEFORE the renderer boots, so renderer
//      localStorage is too late.
//
// This module is **pure** (modulo `app.getPath("userData")` for file
// location): no env reads, no global mutation, no dialog calls. The
// cold-start AGENTSMESH_API_URL escape hatch + UX error reporting both
// live in main/index.ts where they belong.

export type { ServerConfig, ServerKind };
export { PRESETS, DEFAULT_SERVER_CONFIG as DEFAULT };

const FILE_NAME = "server.json";

function configPath(): string {
  return path.join(app.getPath("userData"), FILE_NAME);
}

function normaliseKind(raw: unknown): ServerKind {
  if (raw === "cn" || raw === "custom") return raw;
  return "global"; // legacy "cloud" alias + safe default for anything unexpected
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

// Persist `cfg` to disk and return the **normalised** view that was
// actually written. Callers should use the return value for any in-memory
// state (e.g. main's `currentCfg`) so disk and memory stay byte-identical
// — passing through the raw `cfg` would let untrimmed fields linger in
// memory until the next reload.
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

// Resolve to the actual URL the ApiClient should hit. Throws on invalid
// custom config — caller should surface the error in a dialog rather than
// silently falling back to DEFAULT (which would hide misconfiguration).
export function activeUrl(cfg: ServerConfig): string {
  if (cfg.kind === "global") return PRESETS.global.url;
  if (cfg.kind === "cn") return PRESETS.cn.url;
  if (isValidServerUrl(cfg.customUrl)) return cfg.customUrl;
  throw new Error(`Invalid custom server URL: "${cfg.customUrl}"`);
}
