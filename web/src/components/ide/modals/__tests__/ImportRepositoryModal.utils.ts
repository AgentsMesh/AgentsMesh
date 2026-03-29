import { vi } from "vitest";
import type { RepositoryProviderData, RepositoryData } from "@/lib/api/user-repository-provider";
import type { RepositoryData as OrgRepositoryData } from "@/lib/api/repository";

// Mock provider data - matches RepositoryProviderData interface
export const mockProvider: RepositoryProviderData = {
  id: 1,
  user_id: 1,
  name: "My GitHub",
  provider_type: "github",
  base_url: "https://github.com",
  has_client_id: false,
  has_bot_token: false,
  has_identity: true,
  is_default: true,
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

export const mockGitLabProvider: RepositoryProviderData = {
  id: 2,
  user_id: 1,
  name: "My GitLab",
  provider_type: "gitlab",
  base_url: "https://gitlab.com",
  has_client_id: false,
  has_bot_token: true,
  has_identity: false,
  is_default: false,
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

// Mock repository data - matches RepositoryData interface from user-repository-provider
export const mockRepository: RepositoryData = {
  id: "repo-1",
  name: "my-project",
  slug: "org/my-project",
  description: "A test project",
  default_branch: "main",
  visibility: "private",
  clone_url: "https://github.com/org/my-project.git",
  ssh_clone_url: "git@github.com:org/my-project.git",
  web_url: "https://github.com/org/my-project",
};

// Mock repository API response - matches RepositoryData from repository API
export const mockCreatedRepository: OrgRepositoryData = {
  id: 1,
  organization_id: 1,
  name: "my-project",
  slug: "org/my-project",
  provider_type: "github",
  provider_base_url: "https://github.com",
  clone_url: "https://github.com/org/my-project.git",
  external_id: "repo-1",
  default_branch: "main",
  visibility: "organization",
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

// Create mock functions
export const createMockOnClose = () => vi.fn();
export const createMockOnImported = () => vi.fn();

// Helper to create listRepositories mock response
export const createListRepositoriesResponse = (repositories: RepositoryData[] = [mockRepository]) => ({
  repositories,
  page: 1,
  per_page: 20,
});

// Helper to create repositoryApi.create mock response
export const createRepositoryResponse = (repository: OrgRepositoryData = mockCreatedRepository) => ({
  repository,
});
