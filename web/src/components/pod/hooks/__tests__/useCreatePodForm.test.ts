import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock external dependencies before importing the hook
const mockCreate = vi.fn();
const mockListForAgent = vi.fn();

vi.mock("@/lib/api", () => ({
  podApi: { create: (...args: unknown[]) => mockCreate(...args) },
  userAgentCredentialApi: { listForAgent: (...args: unknown[]) => mockListForAgent(...args) },
}));

vi.mock("@/stores/podCreation", () => ({
  usePodCreationStore: () => ({
    lastAgentSlug: null,
    lastRepositoryId: null,
    lastCredentialProfileId: null,
    lastBranchName: null,
    setLastChoices: vi.fn(),
    clearLastChoices: vi.fn(),
    _hasHydrated: true,
    setHasHydrated: vi.fn(),
  }),
}));

import { useCreatePodForm, RUNNER_HOST_PROFILE_ID } from "../useCreatePodForm";

const mockAgents = [
  { name: "Claude Code", slug: "claude-code", is_builtin: true, is_active: true },
];

describe("useCreatePodForm - credential via agentfile_layer (SSOT)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListForAgent.mockResolvedValue({ profiles: [], runner_host: { available: true } });
  });

  it("should omit CREDENTIAL from agentfile_layer when RunnerHost is selected", async () => {
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    act(() => {
      result.current.setSelectedAgent("claude-code");
    });

    // Verify default is RunnerHost (0)
    expect(result.current.selectedCredentialProfile).toBe(RUNNER_HOST_PROFILE_ID);
    expect(result.current.selectedCredentialProfile).toBe(0);

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    expect(mockCreate).toHaveBeenCalledTimes(1);
    const createArg = mockCreate.mock.calls[0][0];
    // credential_profile_id no longer sent; credential goes through agentfile_layer
    expect(createArg).not.toHaveProperty("credential_profile_id");
    // RunnerHost means no CREDENTIAL declaration in AgentFile Layer
    const layer = createArg.agentfile_layer ?? "";
    expect(layer).not.toContain("CREDENTIAL");
  });

  it("should include CREDENTIAL in agentfile_layer when custom profile selected", async () => {
    const customProfile = { id: 42, name: "My API Key", is_default: false, is_active: true };
    mockListForAgent.mockResolvedValue({
      profiles: [customProfile],
      runner_host: { available: true },
    });
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    // Select agent — triggers credential loading
    act(() => {
      result.current.setSelectedAgent("claude-code");
    });
    // Wait for credential profiles to load
    await act(async () => {});

    act(() => {
      result.current.setSelectedCredentialProfile(42);
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    expect(mockCreate).toHaveBeenCalledTimes(1);
    const createArg = mockCreate.mock.calls[0][0];
    // credential_profile_id no longer sent as separate field
    expect(createArg).not.toHaveProperty("credential_profile_id");
    // Custom profile name should appear in agentfile_layer
    expect(createArg.agentfile_layer).toContain('CREDENTIAL "My API Key"');
  });

  it("should always send agentfile_layer via API (SSOT)", async () => {
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    act(() => {
      result.current.setSelectedAgent("claude-code");
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    const createArg = mockCreate.mock.calls[0][0];
    // The SSOT field: agent_slug and agentfile_layer are the core parameters
    expect(createArg).toHaveProperty("agent_slug", "claude-code");
    // Old scattered fields should not be present
    expect(createArg).not.toHaveProperty("credential_profile_id");
    expect(createArg).not.toHaveProperty("repository_id");
    expect(createArg).not.toHaveProperty("interaction_mode");
    expect(createArg).not.toHaveProperty("branch_name");
    expect(createArg).not.toHaveProperty("prompt");
    expect(createArg).not.toHaveProperty("config_overrides");
  });
});
