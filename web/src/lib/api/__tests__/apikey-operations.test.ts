import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { getApiBaseUrl } from "@/lib/env";

const EXPECTED_API_URL = getApiBaseUrl();

const mockGetState = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => mockGetState(),
  },
}));

const mockFetch = vi.fn();
global.fetch = mockFetch;

import { apiKeyApi } from "../apikey";

describe("apiKeyApi - delete, revoke, cross-cutting", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetState.mockReturnValue({
      token: "test-token",
      currentOrg: { slug: "test-org" },
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("delete", () => {
    it("should make a DELETE request to the correct URL", async () => {
      const mockResponse = { message: "API key deleted successfully" };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.delete(7);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys/7`,
        {
          method: "DELETE",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: undefined,
        }
      );
      expect(result.message).toBe("API key deleted successfully");
    });

    it("should handle server error on delete", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: () => Promise.reject(new Error("Invalid JSON")),
      });

      await expect(apiKeyApi.delete(7)).rejects.toThrow("API Error: 500");
    });
  });

  describe("revoke", () => {
    it("should make a POST request to the revoke endpoint", async () => {
      const mockResponse = { message: "API key revoked successfully" };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.revoke(3);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys/3/revoke`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: undefined,
        }
      );
      expect(result.message).toBe("API key revoked successfully");
    });

    it("should handle already-revoked key (409)", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 409,
        statusText: "Conflict",
        json: () =>
          Promise.resolve({ error: "API key is already revoked" }),
      });

      await expect(apiKeyApi.revoke(3)).rejects.toThrow("API Error: 409");
    });

    it("should handle unauthorized access (403)", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () =>
          Promise.resolve({ error: "Insufficient permissions" }),
      });

      await expect(apiKeyApi.revoke(3)).rejects.toThrow("API Error: 403");
    });
  });

  describe("cross-cutting concerns", () => {
    it("should use correct org slug from auth store", async () => {
      mockGetState.mockReturnValue({
        token: "test-token",
        currentOrg: { slug: "another-org" },
      });
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ api_keys: [], total: 0 })),
      });

      await apiKeyApi.list();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/another-org/api-keys`,
        expect.anything()
      );
    });

    it("should include authorization token from auth store", async () => {
      mockGetState.mockReturnValue({
        token: "my-special-token",
        currentOrg: { slug: "test-org" },
      });
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ api_keys: [], total: 0 })),
      });

      await apiKeyApi.list();

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: "Bearer my-special-token",
          }),
        })
      );
    });
  });
});
