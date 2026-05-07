import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

const storeState = {
  lastAgentSlug: null as string | null,
  lastRepositoryId: null as number | null,
  lastCredentialProfileId: null as number | null,
  lastBranchName: null as string | null,
};

vi.mock("@/stores/podCreation", () => ({
  usePodCreationStore: () => storeState,
}));

import { usePrefsAutoFill } from "../useCreatePodFormEffects";

const mockAgent = { name: "Claude Code", slug: "claude-code", is_builtin: true, is_active: true };
const mockRepoA = {
  id: 11,
  organization_id: 1,
  provider_type: "github",
  provider_base_url: "https://github.com",
  http_clone_url: "https://github.com/x/a.git",
  external_id: "x-a",
  name: "a",
  slug: "x/a",
  default_branch: "main",
  visibility: "organization",
  is_active: true,
  created_at: "",
  updated_at: "",
};
const mockRepoB = { ...mockRepoA, id: 22, name: "b", slug: "x/b", external_id: "x-b" };

beforeEach(() => {
  storeState.lastAgentSlug = null;
  storeState.lastRepositoryId = null;
  storeState.lastBranchName = null;
});

describe("usePrefsAutoFill", () => {
  it("applies lastRepositoryId when no override is provided", () => {
    storeState.lastRepositoryId = 11;
    const setRepo = vi.fn();
    renderHook(() =>
      usePrefsAutoFill([mockAgent], [mockRepoA, mockRepoB], vi.fn(), setRepo, vi.fn()),
    );
    expect(setRepo).toHaveBeenCalledWith(11);
  });

  it("skips lastRepositoryId when overrides.repositoryId is set", () => {
    storeState.lastRepositoryId = 11;
    const setRepo = vi.fn();
    renderHook(() =>
      usePrefsAutoFill(
        [mockAgent],
        [mockRepoA, mockRepoB],
        vi.fn(),
        setRepo,
        vi.fn(),
        { repositoryId: 22 },
      ),
    );
    expect(setRepo).not.toHaveBeenCalled();
  });

  it("still applies lastAgentSlug and lastBranchName when override is set", () => {
    storeState.lastAgentSlug = "claude-code";
    storeState.lastRepositoryId = 11;
    storeState.lastBranchName = "feat/x";
    const setAgent = vi.fn();
    const setRepo = vi.fn();
    const setBranch = vi.fn();
    renderHook(() =>
      usePrefsAutoFill(
        [mockAgent],
        [mockRepoA, mockRepoB],
        setAgent,
        setRepo,
        setBranch,
        { repositoryId: 22 },
      ),
    );
    expect(setAgent).toHaveBeenCalledWith("claude-code");
    expect(setBranch).toHaveBeenCalledWith("feat/x");
    expect(setRepo).not.toHaveBeenCalled();
  });

  it("does not run when no agents are loaded yet", () => {
    storeState.lastRepositoryId = 11;
    const setRepo = vi.fn();
    renderHook(() => usePrefsAutoFill([], [mockRepoA], vi.fn(), setRepo, vi.fn()));
    expect(setRepo).not.toHaveBeenCalled();
  });
});
