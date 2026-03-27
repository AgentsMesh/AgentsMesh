import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { getApiBaseUrl } from "@/lib/env";

// Get the expected API URL - must match getApiBaseUrl() logic used by base.ts
const EXPECTED_API_URL = getApiBaseUrl();

// Mock useAuthStore
const mockGetState = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => mockGetState(),
  },
}));

// Mock global fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Import after mocks are set up
import { apiKeyApi } from "../apikey";

describe("apiKeyApi", () => {
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

  describe("list", () => {
    it("should make a GET request to the correct org-scoped URL", async () => {
      const mockResponse = {
        api_keys: [
          {
            id: 1,
            name: "Test Key",
            key_prefix: "am_test",
            scopes: ["pods:read"],
            is_enabled: true,
          },
        ],
        total: 1,
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.list();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys`,
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: undefined,
        }
      );
      expect(result).toEqual(mockResponse);
      expect(result.api_keys).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it("should return empty list when no keys exist", async () => {
      const mockResponse = { api_keys: [], total: 0 };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.list();

      expect(result.api_keys).toEqual([]);
      expect(result.total).toBe(0);
    });

    it("should throw synchronously when no organization selected", () => {
      mockGetState.mockReturnValue({
        token: "test-token",
        currentOrg: null,
      });

      // orgPath() throws synchronously before request() is called
      expect(() => apiKeyApi.list()).toThrow("No organization selected");
    });
  });

  describe("get", () => {
    it("should make a GET request for a specific key by ID", async () => {
      const mockResponse = {
        api_key: {
          id: 42,
          name: "My Key",
          key_prefix: "am_abc",
          scopes: ["pods:read", "tickets:write"],
          is_enabled: true,
          organization_id: 1,
          created_by: 1,
          created_at: "2024-01-01T00:00:00Z",
          updated_at: "2024-01-01T00:00:00Z",
        },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.get(42);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys/42`,
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: undefined,
        }
      );
      expect(result.api_key.id).toBe(42);
      expect(result.api_key.name).toBe("My Key");
    });

    it("should handle non-existent key (404)", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        statusText: "Not Found",
        json: () => Promise.resolve({ error: "API key not found" }),
      });

      await expect(apiKeyApi.get(999)).rejects.toThrow("API Error: 404");
    });
  });

  describe("create", () => {
    it("should make a POST request with the correct body", async () => {
      const createData = {
        name: "CI/CD Key",
        description: "For CI pipeline",
        scopes: ["pods:read", "pods:write"],
        expires_in: 2592000,
      };
      const mockResponse = {
        api_key: {
          id: 10,
          name: "CI/CD Key",
          description: "For CI pipeline",
          key_prefix: "am_ci1",
          scopes: ["pods:read", "pods:write"],
          is_enabled: true,
          organization_id: 1,
          created_by: 1,
          created_at: "2024-06-01T00:00:00Z",
          updated_at: "2024-06-01T00:00:00Z",
        },
        raw_key: "am_ci1_abcdef1234567890",
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.create(createData);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: JSON.stringify(createData),
        }
      );
      expect(result.api_key.id).toBe(10);
      expect(result.raw_key).toBe("am_ci1_abcdef1234567890");
    });

    it("should create a key without optional fields", async () => {
      const createData = {
        name: "Basic Key",
        scopes: ["pods:read"],
      };
      const mockResponse = {
        api_key: {
          id: 11,
          name: "Basic Key",
          key_prefix: "am_bas",
          scopes: ["pods:read"],
          is_enabled: true,
          organization_id: 1,
          created_by: 1,
          created_at: "2024-06-01T00:00:00Z",
          updated_at: "2024-06-01T00:00:00Z",
        },
        raw_key: "am_bas_xyz789",
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.create(createData);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(createData),
        })
      );
      expect(result.raw_key).toBe("am_bas_xyz789");
    });

    it("should handle validation error (400)", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        statusText: "Bad Request",
        json: () =>
          Promise.resolve({ error: "Name is required" }),
      });

      await expect(
        apiKeyApi.create({ name: "", scopes: [] })
      ).rejects.toThrow("API Error: 400");
    });
  });

  describe("update", () => {
    it("should make a PUT request with the correct URL and body", async () => {
      const updateData = {
        name: "Updated Key Name",
        scopes: ["pods:read", "tickets:read"],
        is_enabled: false,
      };
      const mockResponse = {
        api_key: {
          id: 5,
          name: "Updated Key Name",
          key_prefix: "am_upd",
          scopes: ["pods:read", "tickets:read"],
          is_enabled: false,
          organization_id: 1,
          created_by: 1,
          created_at: "2024-01-01T00:00:00Z",
          updated_at: "2024-06-15T00:00:00Z",
        },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.update(5, updateData);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys/5`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer test-token",
          },
          body: JSON.stringify(updateData),
        }
      );
      expect(result.api_key.name).toBe("Updated Key Name");
      expect(result.api_key.is_enabled).toBe(false);
    });

    it("should update only the name", async () => {
      const updateData = { name: "New Name Only" };
      const mockResponse = {
        api_key: {
          id: 5,
          name: "New Name Only",
          key_prefix: "am_upd",
          scopes: ["pods:read"],
          is_enabled: true,
          organization_id: 1,
          created_by: 1,
          created_at: "2024-01-01T00:00:00Z",
          updated_at: "2024-06-15T00:00:00Z",
        },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await apiKeyApi.update(5, updateData);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/api-keys/5`,
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify(updateData),
        })
      );
      expect(result.api_key.name).toBe("New Name Only");
    });

    it("should handle not found (404) on update", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        statusText: "Not Found",
        json: () => Promise.resolve({ error: "API key not found" }),
      });

      await expect(
        apiKeyApi.update(999, { name: "Nope" })
      ).rejects.toThrow("API Error: 404");
    });
  });

});
