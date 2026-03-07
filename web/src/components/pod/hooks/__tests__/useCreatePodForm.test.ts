import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock external dependencies before importing the hook
const mockCreate = vi.fn();
const mockListForAgentType = vi.fn();

vi.mock("@/lib/api", () => ({
  podApi: { create: (...args: unknown[]) => mockCreate(...args) },
  userAgentCredentialApi: { listForAgentType: (...args: unknown[]) => mockListForAgentType(...args) },
}));

vi.mock("@/stores/podCreation", () => ({
  usePodCreationStore: () => ({
    lastAgentTypeId: null,
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

const mockAgentTypes = [
  { id: 1, name: "Claude Code", slug: "claude-code", is_builtin: true, is_active: true },
];

describe("useCreatePodForm - credential_profile_id submission", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListForAgentType.mockResolvedValue({ profiles: [], runner_host: { available: true } });
  });

  it("should send credential_profile_id=0 when RunnerHost is selected", async () => {
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgentTypes, []));

    // Select agent to pass validation
    act(() => {
      result.current.setSelectedAgent(1);
    });

    // Verify default is RunnerHost (0)
    expect(result.current.selectedCredentialProfile).toBe(RUNNER_HOST_PROFILE_ID);
    expect(result.current.selectedCredentialProfile).toBe(0);

    // Submit form
    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    // Verify podApi.create was called with credential_profile_id: 0 (not undefined)
    expect(mockCreate).toHaveBeenCalledTimes(1);
    const createArg = mockCreate.mock.calls[0][0];
    expect(createArg).toHaveProperty("credential_profile_id", 0);
    // Explicitly verify it's NOT undefined
    expect(createArg.credential_profile_id).not.toBeUndefined();
  });

  it("should send credential_profile_id with positive ID when custom profile selected", async () => {
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgentTypes, []));

    act(() => {
      result.current.setSelectedAgent(1);
      result.current.setSelectedCredentialProfile(42);
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    expect(mockCreate).toHaveBeenCalledTimes(1);
    const createArg = mockCreate.mock.calls[0][0];
    expect(createArg).toHaveProperty("credential_profile_id", 42);
  });

  it("should always include credential_profile_id in API call regardless of value", async () => {
    mockCreate.mockResolvedValue({ pod: { pod_key: "test-pod", id: 1, status: "initializing", agent_status: "idle" } });

    const { result } = renderHook(() => useCreatePodForm(mockAgentTypes, []));

    act(() => {
      result.current.setSelectedAgent(1);
    });

    await act(async () => {
      await result.current.submit(1, {}, { cols: 80, rows: 24 });
    });

    // The key assertion: credential_profile_id must be an OWN property of the call arg
    const createArg = mockCreate.mock.calls[0][0];
    expect(Object.prototype.hasOwnProperty.call(createArg, "credential_profile_id")).toBe(true);
  });
});
