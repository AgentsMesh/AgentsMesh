import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as userRepositoryProvider from "@/lib/api/userRepositoryProvider";
import * as userGitCredential from "@/lib/api/userGitCredential";

import { useGitSettings } from "../useGitSettings";

vi.mock("@/lib/api/userRepositoryProvider", () => ({
  listRepositoryProviders: vi.fn(),
  deleteRepositoryProvider: vi.fn(),
  testRepositoryProviderConnection: vi.fn(),
}));

vi.mock("@/lib/api/userGitCredential", () => ({
  listGitCredentials: vi.fn(),
  deleteGitCredential: vi.fn(),
  setDefaultGitCredential: vi.fn(),
}));

const t = (key: string) => key;

const mockListRepoProviders = vi.mocked(userRepositoryProvider.listRepositoryProviders);
const mockListGitCredentials = vi.mocked(userGitCredential.listGitCredentials);
const mockSetDefaultGitCredential = vi.mocked(userGitCredential.setDefaultGitCredential);

describe("useGitSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockListRepoProviders.mockResolvedValue({ items: [], total: 0 });
    mockListGitCredentials.mockResolvedValue({
      items: [
        {
          id: 7,
          name: "GitHub PAT",
          credential_type: "pat",
          is_default: false,
          created_at: "",
          updated_at: "",
        },
      ],
      total: 1,
      runner_local_is_default: true,
    });
    mockSetDefaultGitCredential.mockResolvedValue({ is_runner_local: true });
  });

  it("selects runner local by sending undefined credential id", async () => {
    const { result } = renderHook(() => useGitSettings(t));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.handleSetDefault(null);
    });

    expect(mockSetDefaultGitCredential).toHaveBeenCalledWith(undefined);
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

    expect(mockSetDefaultGitCredential).toHaveBeenCalledWith(7);
    expect(result.current.data?.defaultCredentialId).toBe(7);
  });
});
