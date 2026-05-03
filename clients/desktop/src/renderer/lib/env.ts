/**
 * Environment helpers for the Electron renderer. Distinct from the web
 * build's env.ts because:
 *   - `window.location.origin` points at the Vite dev server, not the backend.
 *   - The Electron main process knows the backend URL and publishes it via
 *     the preload bridge (`window.electronAPI.apiUrl`).
 *
 * Resolution priority (top wins):
 *   1. User-selected server from server-config.ts (set via the Server
 *      Settings dialog on the login screen — switching reloads the
 *      window so all in-flight callers re-resolve through this path).
 *   2. Preload bridge — the launch-time AGENTSMESH_API_URL env, used
 *      until the user picks a server explicitly.
 *   3. Build-time Vite env override (CI / dev).
 *   4. localhost dev port baked into BUILTIN_SERVERS.
 */

import { getActiveUrl } from "./server-config";

const DEFAULT_API_URL = "http://localhost:25350";

function getConfiguredUrl(): string {
  const fromUser = getActiveUrl();
  if (fromUser) return fromUser;
  if (typeof window !== "undefined" && window.electronAPI?.apiUrl) {
    return window.electronAPI.apiUrl;
  }
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  return DEFAULT_API_URL;
}

export function getApiBaseUrl(): string { return getConfiguredUrl(); }
export function getOAuthBaseUrl(): string { return getConfiguredUrl(); }

export function getWsBaseUrl(): string {
  return getConfiguredUrl().replace(/^http/, "ws");
}

export function getServerUrl(): string { return getConfiguredUrl(); }
export function getServerUrlSSR(): string { return getConfiguredUrl(); }
