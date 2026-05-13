import { vi } from "vitest";
import type { RepositoryProviderData, ProviderRepositoryData } from "@/lib/api/userRepositoryProviderTypes";
import type { RepositoryData } from "@/lib/api/repositoryTypes";

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

export const mockRepository: ProviderRepositoryData = {
  id: "repo-1",
  name: "my-project",
  slug: "org/my-project",
  description: "A test project",
  default_branch: "main",
  visibility: "private",
  http_clone_url: "https://github.com/org/my-project.git",
  ssh_clone_url: "git@github.com:org/my-project.git",
  web_url: "https://github.com/org/my-project",
};

export const mockCreatedRepository: RepositoryData = {
  id: 1,
  organization_id: 1,
  name: "my-project",
  slug: "org/my-project",
  provider_type: "github",
  provider_base_url: "https://github.com",
  http_clone_url: "https://github.com/org/my-project.git",
  external_id: "repo-1",
  default_branch: "main",
  visibility: "organization",
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

export const createMockOnClose = () => vi.fn();
export const createMockOnImported = () => vi.fn();

// Connect-RPC adapter accessors. Tests must vi.mock the modules themselves
// (vi.mock is hoisted per test file). These accessors retrieve the mocked
// instances at setup time. Use stableCredSvc.* to assert against calls made
// by useImportWizard.
import * as userRepositoryProvider from "@/lib/api/userRepositoryProvider";
import * as repositoryConnect from "@/lib/api/repositoryConnect";

export const stableCredSvc = {
  list_repo_providers: vi.mocked(userRepositoryProvider.listRepositoryProviders),
  list_provider_repositories: vi.mocked(userRepositoryProvider.listProviderRepositories),
};

export const stableRepoSvc = {
  create: vi.mocked(repositoryConnect.createRepository),
};

export function setupProviderMocks(providers: RepositoryProviderData[] = [mockProvider, mockGitLabProvider]) {
  vi.mocked(userRepositoryProvider.listRepositoryProviders).mockResolvedValue({
    items: providers,
    total: providers.length,
  });
  vi.mocked(userRepositoryProvider.listProviderRepositories).mockResolvedValue({
    items: [mockRepository],
    total: 1,
  });
}

export function mockRepositoryCreate(response?: RepositoryData) {
  vi.mocked(repositoryConnect.createRepository).mockResolvedValue(response ?? mockCreatedRepository);
  return stableRepoSvc;
}
