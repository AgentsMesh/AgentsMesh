import { vi } from "vitest";

// Mock provider data
export const mockProvider = {
  id: 1,
  name: "My GitHub",
  provider_type: "github",
  base_url: "https://github.com",
  is_active: true,
  has_identity: true,
  has_bot_token: false,
};

export const mockGitLabProvider = {
  id: 2,
  name: "My GitLab",
  provider_type: "gitlab",
  base_url: "https://gitlab.com",
  is_active: true,
  has_identity: false,
  has_bot_token: true,
};

// Mock repository data
export const mockRepository = {
  id: "repo-1",
  name: "my-project",
  full_path: "org/my-project",
  description: "A test project",
  default_branch: "main",
  visibility: "private",
  clone_url: "https://github.com/org/my-project.git",
  ssh_clone_url: "git@github.com:org/my-project.git",
  web_url: "https://github.com/org/my-project",
};

// Mock repository API response
export const mockCreatedRepository = {
  id: 1,
  organization_id: 1,
  name: "my-project",
  full_path: "org/my-project",
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
