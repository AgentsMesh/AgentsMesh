function getPrimaryDomain(): string | undefined {
  const domain = process.env.NEXT_PUBLIC_PRIMARY_DOMAIN;
  if (domain && domain.startsWith("__")) return undefined;
  return domain;
}

function isHttpsEnabled(): boolean {
  const val = process.env.NEXT_PUBLIC_USE_HTTPS;
  if (!val || val.startsWith("__")) return false;
  return val === "true";
}

function deriveHttpUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "https" : "http";
  return `${protocol}://${domain}`;
}

function deriveWsUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "wss" : "ws";
  return `${protocol}://${domain}`;
}

export function getApiBaseUrl(): string {
  if (process.env.NEXT_PUBLIC_API_URL === "") {
    return "";
  }

  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

export function getOAuthBaseUrl(): string {
  if (process.env.NEXT_PUBLIC_OAUTH_URL) {
    return process.env.NEXT_PUBLIC_OAUTH_URL;
  }
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

/**
 * WebSocket cannot be proxied via Next.js rewrites — must use a full URL.
 */
export function getWsBaseUrl(): string {
  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }

  const apiUrl = process.env.NEXT_PUBLIC_API_URL;
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws");
  }

  if (apiUrl === "") {
    const derived = deriveWsUrl();
    if (derived) return derived;
  }

  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    return `${protocol}//${host}`;
  }

  const derived = deriveWsUrl();
  if (derived) return derived;

  return "ws://localhost:10000";
}

const DEFAULT_SERVER_URL = "https://agentsmesh.ai";

export function getServerUrlSSR(): string {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  return DEFAULT_SERVER_URL;
}

// Use getServerUrlSSR in SSR components — calling getServerUrl during SSR causes hydration mismatch.
export function getServerUrl(): string {
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  return getServerUrlSSR();
}
