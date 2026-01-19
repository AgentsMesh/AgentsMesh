import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Get the expected API URL - should match what base.ts uses
const EXPECTED_API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

import { request, ApiError, orgPath } from "../base";

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

describe("request", () => {
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

  it("should make a GET request with authorization headers", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ data: "test" })),
    });

    const result = await request("/api/v1/test");

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-token",
        },
        body: undefined,
      }
    );
    expect(result).toEqual({ data: "test" });
  });

  it("should make a POST request with body", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ id: 1 })),
    });

    const body = { name: "test" };
    const result = await request("/api/v1/test", {
      method: "POST",
      body,
    });

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-token",
        },
        body: JSON.stringify(body),
      }
    );
    expect(result).toEqual({ id: 1 });
  });

  it("should make a PUT request", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ updated: true })),
    });

    const body = { name: "updated" };
    await request("/api/v1/test/1", { method: "PUT", body });

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test/1`,
      expect.objectContaining({ method: "PUT" })
    );
  });

  it("should make a DELETE request", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ deleted: true })),
    });

    await request("/api/v1/test/1", { method: "DELETE" });

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test/1`,
      expect.objectContaining({ method: "DELETE" })
    );
  });

  it("should not include Authorization header when no token", async () => {
    mockGetState.mockReturnValue({
      token: null,
      currentOrg: { slug: "test-org" },
    });
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({})),
    });

    await request("/api/v1/test");

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        body: undefined,
      }
    );
  });

  it("should handle empty response", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(""),
    });

    const result = await request("/api/v1/test");

    expect(result).toEqual({});
  });

  it("should throw ApiError for non-ok response", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: () => Promise.resolve({ error: "Not found" }),
    });

    await expect(request("/api/v1/test")).rejects.toThrow(ApiError);

    try {
      await request("/api/v1/test");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      const apiError = error as ApiError;
      expect(apiError.status).toBe(404);
      expect(apiError.statusText).toBe("Not Found");
      expect(apiError.data).toEqual({ error: "Not found" });
    }
  });

  it("should handle error response with no JSON body", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: () => Promise.reject(new Error("Invalid JSON")),
    });

    try {
      await request("/api/v1/test");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      const apiError = error as ApiError;
      expect(apiError.status).toBe(500);
      expect(apiError.data).toBeNull();
    }
  });

  it("should merge custom headers", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({})),
    });

    await request("/api/v1/test", {
      headers: { "X-Custom-Header": "custom-value" },
    });

    expect(mockFetch).toHaveBeenCalledWith(
      `${EXPECTED_API_URL}/api/v1/test`,
      expect.objectContaining({
        headers: expect.objectContaining({
          "X-Custom-Header": "custom-value",
        }),
      })
    );
  });
});

describe("ApiError", () => {
  it("should create an error with correct properties", () => {
    const error = new ApiError(404, "Not Found", { message: "Resource not found" });

    expect(error.status).toBe(404);
    expect(error.statusText).toBe("Not Found");
    expect(error.data).toEqual({ message: "Resource not found" });
    expect(error.message).toBe("API Error: 404 Not Found");
    expect(error.name).toBe("ApiError");
  });

  it("should work without data", () => {
    const error = new ApiError(500, "Internal Server Error");

    expect(error.status).toBe(500);
    expect(error.statusText).toBe("Internal Server Error");
    expect(error.data).toBeUndefined();
  });
});

describe("orgPath", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should return org-scoped path", () => {
    mockGetState.mockReturnValue({
      token: "test-token",
      currentOrg: { slug: "my-org" },
    });

    const path = orgPath("/pods");

    expect(path).toBe("/api/v1/orgs/my-org/pods");
  });

  it("should throw error when no organization selected", () => {
    mockGetState.mockReturnValue({
      token: "test-token",
      currentOrg: null,
    });

    expect(() => orgPath("/pods")).toThrow("No organization selected");
  });
});
