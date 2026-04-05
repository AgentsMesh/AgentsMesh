import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

// Mock hooks and dependencies
const mockHandleSaveProfile = vi.fn();
const mockSetError = vi.fn();
const mockSetSuccess = vi.fn();

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}));

vi.mock("../useAgentCredentials", () => ({
  useAgentCredentials: () => ({
    loading: false,
    error: null,
    success: null,
    agents: [{ name: "Claude Code", slug: "claude-code", is_builtin: true, is_active: true }],
    expandedAgents: new Set(["claude-code"]),
    runnerHostDefaults: new Set(["claude-code"]),
    credentialFieldsByAgent: new Map([["claude-code", [
      { name: "ANTHROPIC_API_KEY", type: "secret", optional: true },
    ]]]),
    toggleAgent: vi.fn(),
    handleSetRunnerHostDefault: vi.fn(),
    handleSetDefault: vi.fn(),
    handleDelete: vi.fn(),
    handleSaveProfile: mockHandleSaveProfile,
    getProfilesForAgent: () => [],
    setError: mockSetError,
    setSuccess: mockSetSuccess,
  }),
}));

vi.mock("../AgentItem", () => ({
  AgentItem: ({ onAdd }: { onAdd: () => void }) => (
    <div data-testid="agent-item">
      <button data-testid="add-profile" onClick={onAdd}>Add</button>
    </div>
  ),
}));

vi.mock("../CredentialProfileDialog", () => ({
  CredentialProfileDialog: ({
    open,
    onSubmit,
    onOpenChange,
  }: {
    open: boolean;
    onSubmit: (data: unknown) => Promise<void>;
    onOpenChange: (open: boolean) => void;
  }) => open ? (
    <div data-testid="dialog">
      <button
        data-testid="dialog-submit"
        onClick={async () => {
          try {
            await onSubmit({
              name: "Test",
              description: "",
              credentials: { ANTHROPIC_API_KEY: "sk-test" },
            });
          } catch {
            // Simulate CredentialProfileDialog's catch behavior
          }
        }}
      >
        Submit
      </button>
      <button data-testid="dialog-cancel" onClick={() => onOpenChange(false)}>Cancel</button>
    </div>
  ) : null,
}));

vi.mock("@/components/ui/confirm-dialog", () => ({
  ConfirmDialog: () => null,
  useConfirmDialog: () => ({
    dialogProps: {},
    confirm: vi.fn(),
  }),
}));

import { AgentCredentialsSettings } from "../AgentCredentialsSettings";

describe("AgentCredentialsSettings - handleDialogSubmit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockHandleSaveProfile.mockResolvedValue(undefined);
  });

  it("should call handleSaveProfile with correct agentSlug when dialog is submitted", async () => {
    render(<AgentCredentialsSettings />);

    // Open add dialog (sets selectedAgentSlug = "claude-code")
    fireEvent.click(screen.getByTestId("add-profile"));

    // Dialog should appear
    expect(screen.getByTestId("dialog")).toBeInTheDocument();

    // Submit the dialog
    fireEvent.click(screen.getByTestId("dialog-submit"));

    await waitFor(() => {
      expect(mockHandleSaveProfile).toHaveBeenCalledWith(
        "claude-code",
        expect.objectContaining({ name: "Test" }),
        null
      );
    });
  });

  it("should propagate errors from handleSaveProfile to dialog's catch", async () => {
    // Simulate API failure
    const apiError = new Error("API failure");
    mockHandleSaveProfile.mockRejectedValue(apiError);

    render(<AgentCredentialsSettings />);

    // Open add dialog
    fireEvent.click(screen.getByTestId("add-profile"));

    // Submit - the error should propagate up to CredentialProfileDialog's handleSubmit catch
    fireEvent.click(screen.getByTestId("dialog-submit"));

    // The dialog's onSubmit rejects, CredentialProfileDialog's try-catch handles it
    // handleSaveProfile should have been called and failed
    await waitFor(() => {
      expect(mockHandleSaveProfile).toHaveBeenCalledTimes(1);
    });
  });
});
