/**
 * Environment variable utility functions
 *
 * =============================================================================
 * URL Resolution Priority (all getXxxUrl functions follow this order)
 * =============================================================================
 * 1. Explicit env vars (NEXT_PUBLIC_API_URL, NEXT_PUBLIC_WS_URL, etc.)
 * 2. Derived from NEXT_PUBLIC_PRIMARY_DOMAIN + NEXT_PUBLIC_USE_HTTPS
 * 3. Client fallback: window.location.origin (supports IP access and on-premise)
 * 4. Default: localhost:10000
 *
 * =============================================================================
 * Deployment Scenarios
 * =============================================================================
 *
 * [SaaS Production]
 * - Set NEXT_PUBLIC_PRIMARY_DOMAIN=yourdomain.com
 * - Set NEXT_PUBLIC_USE_HTTPS=true
 *
 * [Local Development] (dev.sh)
 * - Set NEXT_PUBLIC_API_URL="" -> uses relative paths, proxied by Next.js rewrites
 *
 * [On-premise / IP Access]
 * - No env vars needed
 * - Client automatically uses window.location.origin
 * - Supports access via http://192.168.1.100:3000
 *
 * =============================================================================
 * Environment Variables
 * =============================================================================
 * - NEXT_PUBLIC_PRIMARY_DOMAIN -> Primary domain (e.g., "yourdomain.com")
 * - NEXT_PUBLIC_USE_HTTPS -> Whether to use HTTPS (true/false)
 * - NEXT_PUBLIC_API_URL -> Explicit API URL (overrides auto-derivation)
 * - NEXT_PUBLIC_WS_URL -> Explicit WebSocket URL
 * - NEXT_PUBLIC_OAUTH_URL -> Explicit OAuth callback URL
 */

// =============================================================================
// Unified Domain Configuration Helpers
// =============================================================================

/**
 * Get the primary domain configuration.
 * Filters out unreplaced docker-entrypoint.sh placeholders (e.g., "__PRIMARY_DOMAIN__").
 */
function getPrimaryDomain(): string | undefined {
  const domain = process.env.NEXT_PUBLIC_PRIMARY_DOMAIN;
  if (domain && domain.startsWith("__")) return undefined;
  return domain;
}

/**
 * Whether HTTPS is enabled.
 * Filters out unreplaced docker-entrypoint.sh placeholders (e.g., "__USE_HTTPS__").
 */
function isHttpsEnabled(): boolean {
  const val = process.env.NEXT_PUBLIC_USE_HTTPS;
  if (!val || val.startsWith("__")) return false;
  return val === "true";
}

/**
 * Derive HTTP(S) URL from PRIMARY_DOMAIN.
 */
function deriveHttpUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "https" : "http";
  return `${protocol}://${domain}`;
}

/**
 * Derive WS(S) URL from PRIMARY_DOMAIN.
 */
function deriveWsUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "wss" : "ws";
  return `${protocol}://${domain}`;
}

// =============================================================================
// Public API
// =============================================================================

/**
 * Get the API base URL.
 * - Local dev: returns empty string (relative paths, proxied by Next.js rewrites)
 * - Docker/production: returns full URL
 * - On-premise: automatically uses the current page origin (supports IP access)
 */
export function getApiBaseUrl(): string {
  // NEXT_PUBLIC_API_URL="" means use relative paths (local dev proxy mode)
  if (process.env.NEXT_PUBLIC_API_URL === "") {
    return "";
  }

  // Explicit API URL takes priority
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // Browser: use current page origin, automatically inheriting the correct protocol.
  // This avoids issues with Next.js build-time constant folding mis-evaluating USE_HTTPS.
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // Server-side: derive from PRIMARY_DOMAIN (used for SSR fetch calls)
  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

/**
 * Get the OAuth base URL (for browser redirects).
 * OAuth requires a full URL since the browser navigates directly to the backend.
 */
export function getOAuthBaseUrl(): string {
  // Explicit configuration takes priority
  if (process.env.NEXT_PUBLIC_OAUTH_URL) {
    return process.env.NEXT_PUBLIC_OAUTH_URL;
  }
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // Browser: use current page origin
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // Server-side: derive from PRIMARY_DOMAIN
  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

/**
 * Get the WebSocket base URL.
 * WebSocket requires a full URL since it cannot be proxied via Next.js rewrites.
 */
export function getWsBaseUrl(): string {
  // Explicit configuration takes priority
  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }

  // Derive from API URL
  const apiUrl = process.env.NEXT_PUBLIC_API_URL;
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws");
  }

  // In local dev proxy mode (NEXT_PUBLIC_API_URL=""), REST uses Next.js rewrites
  // but WebSocket can't be proxied. Derive from PRIMARY_DOMAIN instead of
  // window.location which would incorrectly point to the Next.js dev server.
  if (apiUrl === "") {
    const derived = deriveWsUrl();
    if (derived) return derived;
  }

  // Browser: derive from current page, automatically inheriting the correct protocol
  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    return `${protocol}//${host}`;
  }

  // Server-side: derive from PRIMARY_DOMAIN
  const derived = deriveWsUrl();
  if (derived) return derived;

  return "ws://localhost:10000";
}

// Default server URL for SSR and production
const DEFAULT_SERVER_URL = "http://localhost:10000";

/**
 * Get server deployment URL (SSR-safe version).
 * Returns the same value on both server and client initial render to avoid hydration mismatch.
 *
 * @returns Server URL based on environment variable configuration
 */
export function getServerUrlSSR(): string {
  // Use env var or default (consistent between server and client)
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  return DEFAULT_SERVER_URL;
}

/**
 * Get server deployment URL (for Runner registration and external access).
 * - Client: uses the current page origin
 * - Server: uses NEXT_PUBLIC_API_URL or the default
 *
 * WARNING: Using this function in SSR components will cause hydration mismatch.
 * For SSR components, use getServerUrlSSR() for the initial value,
 * then call getServerUrl() inside useEffect to update.
 *
 * @returns Full server URL (e.g., https://yourdomain.com)
 */
export function getServerUrl(): string {
  // Client: use current page origin
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // Server: use env var or default
  return getServerUrlSSR();
}
