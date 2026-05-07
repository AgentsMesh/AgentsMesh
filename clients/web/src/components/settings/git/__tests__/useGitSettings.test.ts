import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getUserCredentialService } from "@/lib/wasm-core";

import { useGitSettings } from "../useGitSettings";

const mockListRepoProviders = vi.fn();
const mockListGitCredentials = vi.fn();
const mockSetDefaultGitCredential = vi.fn();
const mockClearDefaultGitCredential = vi.fn();

const t = (key: string) => key;

describe("useGitSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockListRepoProviders.mockResolvedValue(JSON.stringify({ providers: [] }));
    mockListGitCredentials.mockResolvedValue(
      JSON.stringify({
        credentials: [
          {
            id: 7,
            name: "GitHub PAT",
            credential_type: "pat",
            is_default: false,
          },
        ],
        runner_local: {
          id: "runner_local",
          name: "Runner Local",
          credential_type: "runner_local",
          is_default: true,
        },
      })
    );
    mockSetDefaultGitCredential.mockResolvedValue(undefined);
    mockClearDefaultGitCredential.mockResolvedValue(undefined);

    vi.mocked(getUserCredentialService).mockReturnValue({
      list_repo_providers: mockListRepoProviders,
      list_git_credentials: mockListGitCredentials,
      set_default_git_credential: mockSetDefaultGitCredential,
      clear_default_git_credential: mockClearDefaultGitCredential,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
  });

  it("clears the default credential when selecting runner local", async () => {
    const { result } = renderHook(() => useGitSettings(t));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.handleSetDefault(null);
    });

    expect(mockClearDefaultGitCredential).toHaveBeenCalledTimes(1);
    expect(mockSetDefaultGitCredential).not.toHaveBeenCalled();
    expect(result.current.data?.defaultCredentialId).toBe("runner_local");
  });

  it("sets a concrete git credential as default", async () => {
    const { result } = renderHook(() => useGitSettings(t));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.handleSetDefault(7);
    });

    expect(mockSetDefaultGitCredential).toHaveBeenCalledWith(
      JSON.stringify({ credential_id: 7 })
    );
    expect(mockClearDefaultGitCredential).not.toHaveBeenCalled();
    expect(result.current.data?.defaultCredentialId).toBe(7);
  });
});
