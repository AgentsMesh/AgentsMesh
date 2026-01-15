import { vi } from "vitest";

// Mock functions
export const mockSetSelectedRunnerId = vi.fn();
export const mockFormReset = vi.fn();
export const mockFormSubmit = vi.fn();
export const mockSetPrompt = vi.fn();
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
};

export const defaultFormState = {
  selectedAgent: null,
  selectedRepository: null,
  selectedBranch: "",
  selectedCredentialProfile: 0,
  prompt: "",
  credentialProfiles: [],
  loadingCredentials: false,
  setSelectedAgent: mockSetSelectedAgent,
  setSelectedRepository: vi.fn(),
  setSelectedBranch: vi.fn(),
  setSelectedCredentialProfile: vi.fn(),
  setPrompt: mockSetPrompt,
  selectedAgentSlug: "",
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
  status: "online",
  capabilities: [],
};

export const mockAgentType = {
  id: 1,
  name: "Claude Code",
  slug: "claude-code",
};

export const mockRepository = {
  id: 1,
  full_path: "org/repo1",
  default_branch: "main",
};

export const mockCredentialProfile = {
  id: 1,
  name: "My Credentials",
  is_default: false,
};

export function clearAllMocks() {
  mockSetSelectedRunnerId.mockClear();
  mockFormReset.mockClear();
  mockFormSubmit.mockClear();
  mockSetPrompt.mockClear();
  mockSetSelectedAgent.mockClear();
  mockResetPluginConfig.mockClear();
}
