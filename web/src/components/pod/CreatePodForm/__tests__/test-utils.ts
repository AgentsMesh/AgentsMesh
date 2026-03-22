import { vi } from "vitest";

// Mock functions
export const mockSetSelectedRunnerId = vi.fn();
export const mockFormReset = vi.fn();
export const mockFormSubmit = vi.fn();
export const mockSetPrompt = vi.fn();
export const mockSetAlias = vi.fn();
export const mockSetSelectedAgent = vi.fn();
export const mockResetPluginConfig = vi.fn();

// Default mock values
export const defaultPodCreationData = {
  runners: [],
  repositories: [],
  loading: false,
  selectedRunner: null,
  setSelectedRunnerId: mockSetSelectedRunnerId,
  availableAgentTypes: [],
  agentTypes: [],
  error: null,
};

export const defaultFormState = {
  selectedAgent: null,
  selectedRepository: null,
  selectedBranch: "",
  selectedCredentialProfile: 0,
  interactionMode: "pty" as const,
  prompt: "",
  alias: "",
  credentialProfiles: [],
  loadingCredentials: false,
  setSelectedAgent: mockSetSelectedAgent,
  setSelectedRepository: vi.fn(),
  setSelectedBranch: vi.fn(),
  setSelectedCredentialProfile: vi.fn(),
  setInteractionMode: vi.fn(),
  setPrompt: mockSetPrompt,
  setAlias: mockSetAlias,
  selectedAgentSlug: "",
  supportedModes: ["pty"],
  loading: false,
  error: null,
  validationErrors: {},
  isValid: false,
  reset: mockFormReset,
  validate: vi.fn(),
  submit: mockFormSubmit,
};

export const defaultConfigOptions = {
  fields: [],
  loading: false,
  config: {},
  updateConfig: vi.fn(),
  resetConfig: mockResetPluginConfig,
};

// Common test data
export const mockRunner = {
  id: 1,
  node_id: "runner-1",
  current_pods: 0,
  max_concurrent_pods: 5,
  status: "online" as const,
  capabilities: [],
  is_enabled: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

export const mockAgentType = {
  id: 1,
  name: "Claude Code",
  slug: "claude-code",
  is_builtin: true,
  is_active: true,
};

export const mockRepository = {
  id: 1,
  organization_id: 1,
  provider_type: "github",
  provider_base_url: "https://github.com",
  clone_url: "https://github.com/org/repo1.git",
  external_id: "org-repo1",
  name: "repo1",
  full_path: "org/repo1",
  default_branch: "main",
  visibility: "organization",
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

export const mockCredentialProfile = {
  id: 1,
  user_id: 1,
  agent_type_id: 1,
  name: "My Credentials",
  is_runner_host: false,
  is_default: false,
  is_active: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

export function clearAllMocks() {
  mockSetSelectedRunnerId.mockClear();
  mockFormReset.mockClear();
  mockFormSubmit.mockClear();
  mockSetPrompt.mockClear();
  mockSetAlias.mockClear();
  mockSetSelectedAgent.mockClear();
  mockResetPluginConfig.mockClear();
}
