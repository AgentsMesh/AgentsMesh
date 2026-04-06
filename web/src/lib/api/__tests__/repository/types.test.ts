import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock must be at module level for Vitest hoisting
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => ({
      token: "test-token",
      currentOrg: { slug: "test-org" },
    }),
  },
}));

import { repositoryApi, RepositoryData } from "../../repository";
import { mockFetch, setupRepositoryTests } from "./testSetup";

describe("repositoryApi - Type Validation", () => {
  beforeEach(() => {
    setupRepositoryTests();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it("should handle repository with webhook_config", async () => {
    const mockRepository: RepositoryData = {
      id: 1,
      organization_id: 1,
      provider_type: "gitlab",
      provider_base_url: "https://gitlab.com",
      http_clone_url: "https://gitlab.com/org/repo.git",
      external_id: "123",
      name: "test-repo",
      slug: "org/test-repo",
      default_branch: "main",
      visibility: "organization",
      is_active: true,
      webhook_config: {
        id: "wh_123",
        url: "https://example.com/webhooks/org/gitlab/1",
        events: ["merge_request", "pipeline"],
        is_active: true,
        needs_manual_setup: false,
        created_at: "2025-01-01T00:00:00Z",
      },
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    };

    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ repository: mockRepository })),
    });

    const result = await repositoryApi.get(1);

    expect(result.repository.webhook_config).toBeDefined();
    expect(result.repository.webhook_config?.id).toBe("wh_123");
    expect(result.repository.webhook_config?.is_active).toBe(true);
  });

  it("should handle repository with optional fields", async () => {
    const mockRepository: RepositoryData = {
      id: 1,
      organization_id: 1,
      provider_type: "github",
      provider_base_url: "https://github.com",
      http_clone_url: "https://github.com/org/repo.git",
      external_id: "456",
      name: "minimal-repo",
      slug: "org/minimal-repo",
      default_branch: "main",
      visibility: "private",
      imported_by_user_id: 123,
      is_active: true,
      ticket_prefix: "PROJ",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    };

    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ repository: mockRepository })),
    });

    const result = await repositoryApi.get(1);

    expect(result.repository.ticket_prefix).toBe("PROJ");
    expect(result.repository.imported_by_user_id).toBe(123);
    expect(result.repository.visibility).toBe("private");
  });
});
