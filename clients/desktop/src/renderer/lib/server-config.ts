import {
  PRESETS,
  isValidServerUrl,
  type ServerConfig,
  type ServerKind,
} from "../../shared/server-config-types";

/**
 * Renderer-side facade over the main-process server-config SSOT.
 *
 * Read path: `getConfig()` returns the snapshot preload sync-injected at
 * boot; AuthShell can render the label without async ceremony.
 * Write path: `saveConfig()` round-trips through main, which persists
 * `userData/server.json` + reloads the window so preload re-snapshots and
 * Rust core's ApiClient picks up the new base_url. There is **no**
 * localStorage anymore — keeping a renderer-side persisted copy was the
 * bug that left main's AppState bound to a stale URL while renderer
 * happily showed a switched server.
 */

export type { ServerConfig, ServerKind };
export { isValidServerUrl };

export function getConfig(): ServerConfig {
  return window.electronAPI.serverConfig.snapshot;
}

export function getPresets(): { kind: "global" | "cn"; label: string; url: string }[] {
  return [
    { kind: "global", ...PRESETS.global },
    { kind: "cn", ...PRESETS.cn },
  ];
}

export function getActiveLabel(cfg: ServerConfig): string {
  if (cfg.kind === "custom" && cfg.customLabel.trim()) return cfg.customLabel;
  if (cfg.kind === "cn") return PRESETS.cn.label;
  return PRESETS.global.label;
}

// Write-side. Main reloads the window after save (the IPC resolves first,
// then renderer is torn down) — callers shouldn't depend on work scheduled
// after `await saveConfig`.
export async function saveConfig(next: ServerConfig): Promise<void> {
  await window.electronAPI.serverConfig.set({
    kind: next.kind,
    customLabel: next.customLabel.trim(),
    customUrl: next.customUrl.trim(),
  });
}
