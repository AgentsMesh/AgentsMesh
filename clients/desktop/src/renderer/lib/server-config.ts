import {
  PRESETS,
  isValidServerUrl,
  type ServerConfig,
  type ServerKind,
} from "../../shared/server-config-types";

// SSOT is main process. saveConfig round-trips through main which persists
// userData/server.json + reloads the window — DO NOT add localStorage back;
// the renderer-persisted copy bug left main's AppState bound to a stale URL.

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

// Main reloads window after save IPC resolves — callers cannot depend on work scheduled after `await saveConfig`.
export async function saveConfig(next: ServerConfig): Promise<void> {
  await window.electronAPI.serverConfig.set({
    kind: next.kind,
    customLabel: next.customLabel.trim(),
    customUrl: next.customUrl.trim(),
  });
}
