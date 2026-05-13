import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import * as podConnect from "@/lib/api/podConnect";
import * as userAgentCredential from "@/lib/api/userAgentCredential";

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

vi.mock("@/lib/api/podConnect", () => ({
  createPod: vi.fn(),
}));

vi.mock("@/lib/api/userAgentCredential", () => ({
  listAgentCredentialProfilesForAgent: vi.fn(),
}));

import { useCreatePodForm, RUNNER_HOST_PROFILE_ID } from "../useCreatePodForm";

const mockAgents = [
  { name: "Claude Code", slug: "claude-code", is_builtin: true, is_active: true },
];

const mockCreatePod = vi.mocked(podConnect.createPod);
const mockListCredentials = vi.mocked(
  userAgentCredential.listAgentCredentialProfilesForAgent
);

describe("useCreatePodForm - credential via agentfile_layer (SSOT)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListCredentials.mockResolvedValue({
      items: [],
      total: 0,
      runner_host: { available: true, description: "" },
    });
  });

  it("should omit CREDENTIAL from agentfile_layer when RunnerHost is selected", async () => {
    mockCreatePod.mockResolvedValue({
      pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" },
    });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    act(() => {
      result.current.setSelectedAgent("claude-code");
    });

    expect(result.current.selectedCredentialProfile).toBe(RUNNER_HOST_PROFILE_ID);
    expect(result.current.selectedCredentialProfile).toBe(0);

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    expect(mockCreatePod).toHaveBeenCalledTimes(1);
    const [, createArg] = mockCreatePod.mock.calls[0];
    expect(createArg).not.toHaveProperty("credential_profile_id");
    const layer = createArg.agentfile_layer ?? "";
    expect(layer).not.toContain("CREDENTIAL");
  });

  it("should include CREDENTIAL in agentfile_layer when custom profile selected", async () => {
    const customProfile = {
      id: 42,
      user_id: 1,
      agent_slug: "claude-code",
      name: "My API Key",
      is_runner_host: false,
      is_default: false,
      is_active: true,
      configured_fields: [],
      configured_values: {},
      created_at: "",
      updated_at: "",
    };
    mockListCredentials.mockResolvedValue({
      items: [customProfile],
      total: 1,
      runner_host: { available: true, description: "" },
    });
    mockCreatePod.mockResolvedValue({
      pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" },
    });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    act(() => {
      result.current.setSelectedAgent("claude-code");
    });
    await act(async () => {});

    act(() => {
      result.current.setSelectedCredentialProfile(42);
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    expect(mockCreatePod).toHaveBeenCalledTimes(1);
    const [, createArg] = mockCreatePod.mock.calls[0];
    expect(createArg).not.toHaveProperty("credential_profile_id");
    expect(createArg.agentfile_layer).toContain('CREDENTIAL "My API Key"');
  });

  it("should always send agentfile_layer via API (SSOT)", async () => {
    mockCreatePod.mockResolvedValue({
      pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" },
    });

    const { result } = renderHook(() => useCreatePodForm(mockAgents, []));

    act(() => {
      result.current.setSelectedAgent("claude-code");
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    const [, createArg] = mockCreatePod.mock.calls[0];
    expect(createArg).toHaveProperty("agent_slug", "claude-code");
    expect(createArg).not.toHaveProperty("credential_profile_id");
    expect(createArg).not.toHaveProperty("repository_id");
    expect(createArg).not.toHaveProperty("interaction_mode");
    expect(createArg).not.toHaveProperty("branch_name");
    expect(createArg).not.toHaveProperty("prompt");
    expect(createArg).not.toHaveProperty("config_overrides");
  });
});
