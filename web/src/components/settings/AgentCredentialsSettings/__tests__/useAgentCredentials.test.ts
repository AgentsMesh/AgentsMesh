import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock API modules
const mockList = vi.fn();
const mockListAgents = vi.fn();
const mockGetConfigSchema = vi.fn();
const mockApiCreate = vi.fn();
const mockApiUpdate = vi.fn();

vi.mock("@/lib/api", () => ({
  userAgentCredentialApi: {
    list: (...args: unknown[]) => mockList(...args),
    create: (...args: unknown[]) => mockApiCreate(...args),
    update: (...args: unknown[]) => mockApiUpdate(...args),
    delete: vi.fn(),
    setDefault: vi.fn(),
    listForAgent: vi.fn(),
  },
  agentApi: {
    list: (...args: unknown[]) => mockListAgents(...args),
    getConfigSchema: (...args: unknown[]) => mockGetConfigSchema(...args),
  },
}));

import { useAgentCredentials } from "../useAgentCredentials";
import type { CredentialFormData } from "../types";

const mockTranslate = (key: string) => key;

describe("useAgentCredentials - handleSaveProfile error handling", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockList.mockResolvedValue({ items: [] });
    mockListAgents.mockResolvedValue({ agents: [{ name: "Claude", slug: "claude-code" }] });
    mockGetConfigSchema.mockResolvedValue({
      schema: {
        fields: [],
        credential_fields: [
          { name: "ANTHROPIC_API_KEY", type: "secret", optional: true },
          { name: "ANTHROPIC_AUTH_TOKEN", type: "secret", optional: true },
          { name: "ANTHROPIC_BASE_URL", type: "text", optional: true },
        ],
      },
    });
  });

  it("should propagate API errors from create to caller", async () => {
    const apiError = new Error("Network error");
    mockApiCreate.mockRejectedValue(apiError);

    const { result } = renderHook(() => useAgentCredentials(mockTranslate));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const formData: CredentialFormData = {
      name: "Test Profile",
      description: "",
      credentials: { ANTHROPIC_API_KEY: "sk-test" },
    };

    await expect(
      act(async () => {
        await result.current.handleSaveProfile("claude-code", formData, null);
      })
    ).rejects.toThrow("Network error");
  });

  it("should propagate API errors from update to caller", async () => {
    const apiError = new Error("Unauthorized");
    mockApiUpdate.mockRejectedValue(apiError);

    const { result } = renderHook(() => useAgentCredentials(mockTranslate));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const editingProfile = {
      id: 5,
      user_id: 1,
      agent_slug: "claude-code",
      name: "Existing",
      is_runner_host: false,
      is_default: false,
      is_active: true,
      created_at: "2024-01-01",
      updated_at: "2024-01-01",
    };

    const formData: CredentialFormData = {
      name: "Updated",
      description: "",
      credentials: { ANTHROPIC_API_KEY: "sk-new" },
    };

    await expect(
      act(async () => {
        await result.current.handleSaveProfile("claude-code", formData, editingProfile);
      })
    ).rejects.toThrow("Unauthorized");
  });

  it("should set success message and call loadData on successful create", async () => {
    mockApiCreate.mockResolvedValue({ profile: { id: 1 } });

    const { result } = renderHook(() => useAgentCredentials(mockTranslate));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const formData: CredentialFormData = {
      name: "New Profile",
      description: "",
      credentials: { ANTHROPIC_API_KEY: "sk-test" },
    };

    await act(async () => {
      await result.current.handleSaveProfile("claude-code", formData, null);
    });

    expect(mockApiCreate).toHaveBeenCalledTimes(1);
    expect(result.current.success).toBe("settings.agentCredentials.profileCreated");
    // loadData should have been called again (mockList called 2 times: initial + after save)
    expect(mockList).toHaveBeenCalledTimes(2);
  });
});
