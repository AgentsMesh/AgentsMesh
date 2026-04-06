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
import { mockFetch, EXPECTED_API_URL, basePath, setupRepositoryTests } from "./testSetup";

describe("repositoryApi - CRUD Operations", () => {
  beforeEach(() => {
    setupRepositoryTests();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("list", () => {
    it("should fetch all repositories", async () => {
      const mockRepositories: RepositoryData[] = [
        {
          id: 1,
          organization_id: 1,
          provider_type: "gitlab",
          provider_base_url: "https://gitlab.com",
          http_clone_url: "https://gitlab.com/org/repo1.git",
          external_id: "123",
          name: "repo1",
          slug: "org/repo1",
          default_branch: "main",
          visibility: "organization",
          is_active: true,
          created_at: "2025-01-01T00:00:00Z",
          updated_at: "2025-01-01T00:00:00Z",
        },
      ];

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ repositories: mockRepositories })),
      });

      const result = await repositoryApi.list();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.repositories).toHaveLength(1);
      expect(result.repositories[0].name).toBe("repo1");
    });
  });

  describe("get", () => {
    it("should fetch a repository by id", async () => {
      const mockRepository: RepositoryData = {
        id: 1,
        organization_id: 1,
        provider_type: "github",
        provider_base_url: "https://github.com",
        http_clone_url: "https://github.com/org/repo.git",
        external_id: "456",
        name: "test-repo",
        slug: "org/test-repo",
        default_branch: "main",
        visibility: "organization",
        is_active: true,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-01T00:00:00Z",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ repository: mockRepository })),
      });

      const result = await repositoryApi.get(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.repository.id).toBe(1);
      expect(result.repository.name).toBe("test-repo");
    });
  });

  describe("create", () => {
    it("should create a new repository", async () => {
      const createRequest = {
        provider_type: "gitlab",
        provider_base_url: "https://gitlab.com",
        external_id: "789",
        name: "new-repo",
        slug: "org/new-repo",
      };

      const mockRepository: RepositoryData = {
        id: 2,
        organization_id: 1,
        ...createRequest,
        http_clone_url: "https://gitlab.com/org/new-repo.git",
        default_branch: "main",
        visibility: "organization",
        is_active: true,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-01T00:00:00Z",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ repository: mockRepository })),
      });

      const result = await repositoryApi.create(createRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(createRequest),
        })
      );
      expect(result.repository.name).toBe("new-repo");
    });
  });

  describe("update", () => {
    it("should update a repository", async () => {
      const updateRequest = {
        name: "updated-repo",
        default_branch: "develop",
      };

      const mockRepository: RepositoryData = {
        id: 1,
        organization_id: 1,
        provider_type: "gitlab",
        provider_base_url: "https://gitlab.com",
        http_clone_url: "https://gitlab.com/org/updated-repo.git",
        external_id: "123",
        name: "updated-repo",
        slug: "org/updated-repo",
        default_branch: "develop",
        visibility: "organization",
        is_active: true,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ repository: mockRepository })),
      });

      const result = await repositoryApi.update(1, updateRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1`,
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify(updateRequest),
        })
      );
      expect(result.repository.name).toBe("updated-repo");
      expect(result.repository.default_branch).toBe("develop");
    });
  });

  describe("delete", () => {
    it("should delete a repository", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ message: "Repository deleted" })),
      });

      const result = await repositoryApi.delete(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1`,
        expect.objectContaining({ method: "DELETE" })
      );
      expect(result.message).toBe("Repository deleted");
    });
  });
});
