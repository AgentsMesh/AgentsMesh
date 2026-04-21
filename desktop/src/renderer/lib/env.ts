/**
 * Environment helpers for the Electron renderer. Distinct from the web
 * build's env.ts because:
 *   - `window.location.origin` points at the Vite dev server, not the backend.
 *   - The Electron main process knows the backend URL and publishes it via
 *     the preload bridge (`window.electronAPI.apiUrl`).
 */

const DEFAULT_API_URL = "http://localhost:25350";

function getConfiguredUrl(): string {
  // 1. Runtime: preload bridge exposes the resolved API URL.
  if (typeof window !== "undefined" && window.electronAPI?.apiUrl) {
    return window.electronAPI.apiUrl;
  }
  // 2. Build-time Vite env var override.
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
