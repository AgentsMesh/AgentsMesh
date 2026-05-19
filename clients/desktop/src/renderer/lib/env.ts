// SSOT is main process (src/main/server_config.ts). Preload sync-injects to
// window.electronAPI.apiUrl at boot; main reloads window on server switch so the
// snapshot is never stale within a process lifetime.
// No fallback default — 2026-05-10 incident showed any local-port "friendly default"
// silently routes to whatever else (OrbStack, Docker Desktop) listens on that port.
// Multiple exports below are cross-bundle interface contract: desktop vite aliases
// `@/lib/env` here but resolves `@/hooks` to web tree, so getServerUrl/getServerUrlSSR
// must keep their names as adapter shims.

export function getApiBaseUrl(): string {
  return window.electronAPI.apiUrl;
}

export function getWsBaseUrl(): string {
  return window.electronAPI.apiUrl.replace(/^http/, "ws");
}

export function getServerUrl(): string { return window.electronAPI.apiUrl; }
export function getServerUrlSSR(): string { return window.electronAPI.apiUrl; }
