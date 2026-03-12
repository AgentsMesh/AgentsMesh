import { useAuthStore } from "@/stores/auth";
import { getApiBaseUrl } from "@/lib/env";

const API_BASE_URL = getApiBaseUrl();

type RequestMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

export interface RequestOptions {
  method?: RequestMethod;
  body?: unknown;
  headers?: Record<string, string>;
  skipAuthRefresh?: boolean; // Skip token refresh for auth endpoints
  signal?: AbortSignal; // AbortController signal to cancel in-flight requests
}

export interface ApiErrorData {
  error?: string;
  code?: string;
  [key: string]: unknown;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: unknown
  ) {
    super(`API Error: ${status} ${statusText}`);
    this.name = "ApiError";
  }

  get code(): string | undefined {
    const d = this.data as ApiErrorData | null | undefined;
    return d?.code;
  }

  get serverMessage(): string | undefined {
    const d = this.data as ApiErrorData | null | undefined;
    return d?.error;
  }

  hasCode(code: string): boolean {
    return this.code === code;
  }
}

// Track if a token refresh is in progress to avoid multiple refreshes
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();

  if (!refreshToken) {
    return false;
  }

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      // Refresh token is invalid or expired
      logout();
      return false;
    }

    const data = await response.json();
    setTokens(data.token, data.refresh_token);
    return true;
  } catch {
    logout();
    return false;
  }
}

export async function handleTokenRefresh(): Promise<boolean> {
  // If already refreshing, wait for that to complete
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = refreshAccessToken().finally(() => {
    isRefreshing = false;
    refreshPromise = null;
  });

  return refreshPromise;
}

export async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { method = "GET", body, headers = {}, skipAuthRefresh = false, signal } = options;
  const { token } = useAuthStore.getState();

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...headers,
  };

  if (token) {
    requestHeaders["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method,
    headers: requestHeaders,
    body: body ? JSON.stringify(body) : undefined,
    signal,
  });

  // Handle 401 Unauthorized - try to refresh token
  if (response.status === 401 && !skipAuthRefresh) {
    const refreshed = await handleTokenRefresh();

    if (refreshed) {
      // Retry the original request with new token
      const { token: newToken } = useAuthStore.getState();
      requestHeaders["Authorization"] = `Bearer ${newToken}`;

      const retryResponse = await fetch(`${API_BASE_URL}${endpoint}`, {
        method,
        headers: requestHeaders,
        body: body ? JSON.stringify(body) : undefined,
        signal,
      });

      if (!retryResponse.ok) {
        const data = await retryResponse.json().catch(() => null);
        throw new ApiError(retryResponse.status, retryResponse.statusText, data);
      }

      const text = await retryResponse.text();
      if (!text) {
        return {} as T;
      }
      return JSON.parse(text);
    } else {
      // Refresh failed, redirect to login
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
      throw new ApiError(401, "Unauthorized", { message: "Session expired" });
    }
  }

  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new ApiError(response.status, response.statusText, data);
  }

  // Handle empty responses
  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text);
}

// Helper function to get org-scoped API path
// Path format: /api/v1/orgs/:slug/*
export function orgPath(path: string): string {
  const { currentOrg } = useAuthStore.getState();
  if (!currentOrg) {
    throw new Error("No organization selected");
  }
  return `/api/v1/orgs/${currentOrg.slug}${path}`;
}

// Public API request (no auth required)
export async function publicRequest<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: "GET",
    headers: { "Content-Type": "application/json" },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new ApiError(response.status, response.statusText, data);
  }

  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text);
}

export async function publicPost<T>(endpoint: string, body: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new ApiError(response.status, response.statusText, data);
  }

  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text);
}
