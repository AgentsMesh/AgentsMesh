import { getAuthToken, useAuthStore } from "@/stores/auth";

/**
 * Get API URL from environment variables
 * Supports both unified domain configuration and legacy configuration
 */
function getApiUrl(): string {
  // Legacy configuration takes priority
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // Browser: always use current page origin to inherit correct protocol (http/https)
  // This avoids Next.js build-time constant folding issues with USE_HTTPS env var,
  // and works naturally with Traefik routing (admin.agentsmesh.ai/api → backend)
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // Server-side: derive from PRIMARY_DOMAIN (for SSR fetch calls)
  const primaryDomain = process.env.NEXT_PUBLIC_PRIMARY_DOMAIN;
  if (primaryDomain && !primaryDomain.startsWith("__")) {
    // Runtime check: avoid build-time constant folding by comparing at runtime
    const useHttpsVal = process.env.NEXT_PUBLIC_USE_HTTPS;
    const protocol = useHttpsVal && useHttpsVal === "true" ? "https" : "http";
    return `${protocol}://${primaryDomain}`;
  }

  // Server-side fallback
  return "http://localhost:10000";
}

const API_URL = getApiUrl();

export interface ApiError {
  error: string;
  status: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = getAuthToken();
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      ...options.headers,
    };

    if (token) {
      (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    // Handle 401 - logout user
    if (response.status === 401) {
      useAuthStore.getState().logout();
      throw { error: "Session expired. Please login again.", status: 401 };
    }

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw {
        error: errorData.error || `HTTP ${response.status}`,
        status: response.status,
      } as ApiError;
    }

    return response.json();
  }

  async get<T>(endpoint: string, params?: Record<string, string | number | undefined>): Promise<T> {
    let queryString = "";
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.append(key, String(value));
        }
      });
      const qs = searchParams.toString();
      if (qs) {
        queryString = `?${qs}`;
      }
    }
    return this.request<T>(`${endpoint}${queryString}`);
  }

  async post<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: "POST",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: "PUT",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: "DELETE",
      body: data ? JSON.stringify(data) : undefined,
    });
  }
}

export const apiClient = new ApiClient(`${API_URL}/api/v1/admin`);
