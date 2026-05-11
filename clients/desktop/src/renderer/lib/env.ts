/**
 * Renderer-side base_url accessor.
 *
 * SSOT lives in main (`server_config.ts`). Preload sync-injects the
 * resolved active URL onto `window.electronAPI.apiUrl` at boot, so all
 * accessors here are O(1) reads of that snapshot. When the user switches
 * servers, main reloads the window — preload re-reads, snapshot is
 * never stale within a process lifetime.
 *
 * No `?? "http://localhost:25350"` fallback. The 2026-05-10 incident
 * proved that any "friendly default" pointing at a local port is a
 * footgun the moment something else (OrbStack, Docker Desktop, ...)
 * decides to listen there.
 *
 * Multiple accessor names exist as an INTERFACE CONTRACT, not redundancy:
 * desktop's vite config aliases `@/lib/env` to THIS file but also resolves
 * `@/hooks` etc. to the web source tree. Web hooks like `useServerUrl`
 * import `getServerUrl` / `getServerUrlSSR`, so removing those names here
 * breaks the desktop bundle. They all return the same value because
 * desktop has no SSR; the names are kept as adapter shims for cross-bundle
 * imports.
 */

export function getApiBaseUrl(): string {
  return window.electronAPI.apiUrl;
}

export function getWsBaseUrl(): string {
  return window.electronAPI.apiUrl.replace(/^http/, "ws");
}

// Adapter shims for web-tree code reused via vite alias. See module doc.
export function getServerUrl(): string { return window.electronAPI.apiUrl; }
export function getServerUrlSSR(): string { return window.electronAPI.apiUrl; }
