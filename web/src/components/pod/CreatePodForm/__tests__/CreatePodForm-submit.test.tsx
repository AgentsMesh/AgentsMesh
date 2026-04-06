import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CreatePodForm } from "../index";
import { CreatePodFormConfig } from "../types";
import {
  mockSetSelectedRunnerId,
  mockFormReset,
  mockFormSubmit,
  mockSetPrompt,
  mockResetPluginConfig,
  defaultPodCreationData,
  defaultFormState,
  defaultConfigOptions,
  mockRunner,
  mockAgent,
  clearAllMocks,
} from "./test-utils";

// Mock hooks
vi.mock("../../hooks", () => ({
  usePodCreationData: vi.fn(() => defaultPodCreationData),
  useCreatePodForm: vi.fn(() => defaultFormState),
  RUNNER_HOST_PROFILE_ID: 0,
}));

vi.mock("@/components/ide/hooks", () => ({
  useConfigOptions: vi.fn(() => defaultConfigOptions),
}));

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}));

vi.mock("@/components/ide/ConfigForm", () => ({
  ConfigForm: () => <div data-testid="config-form">Config Form</div>,
}));

vi.mock("@/lib/terminal-size", () => ({
  estimateWorkspaceTerminalSize: () => ({ cols: 80, rows: 24 }),
}));

vi.mock("@/components/ui/collapsible", () => ({
  Collapsible: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CollapsibleTrigger: ({ children }: { children: React.ReactNode }) => <button>{children}</button>,
  CollapsibleContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import { usePodCreationData, useCreatePodForm } from "../../hooks";

describe("CreatePodForm - submission, errors & reset", () => {
  beforeEach(() => {
    clearAllMocks();
    vi.clearAllMocks();
  });

  describe("form submission", () => {
    const setupSubmitState = () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
        prompt: "test prompt",
        selectedAgentSlug: "claude-code",
        isValid: true,
      });
    };

    it("should call submit with correct parameters (runner passed when manually selected)", async () => {
      setupSubmitState();
      mockFormSubmit.mockResolvedValue({ pod_key: "test-pod" });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        // prompt is now in form state (agentfile_layer SSOT), not passed as prompt
        expect(mockFormSubmit).toHaveBeenCalledWith(1, {}, { ticketSlug: undefined, cols: 80, rows: 24 });
      });
    });

    it("should call submit with null runner when no runner manually selected", async () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: null,
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
        prompt: "test prompt",
        selectedAgentSlug: "claude-code",
        isValid: true,
      });
      mockFormSubmit.mockResolvedValue({ pod_key: "test-pod" });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        expect(mockFormSubmit).toHaveBeenCalledWith(null, {}, { ticketSlug: undefined, cols: 80, rows: 24 });
      });
    });

    it("should pass ticketSlug when in ticket scenario", async () => {
      setupSubmitState();
      mockFormSubmit.mockResolvedValue({ pod_key: "test-pod" });

      const config: CreatePodFormConfig = {
        scenario: "ticket",
        context: { ticket: { id: 123, slug: "PROJ-123", title: "Test" } },
      };

      render(<CreatePodForm config={config} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        expect(mockFormSubmit).toHaveBeenCalledWith(1, {}, { ticketSlug: "PROJ-123", cols: 80, rows: 24 });
      });
    });

    it("should disable create button when no agent selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: null,
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.create")).toBeDisabled();
    });

    it("should enable create button when agent is selected (no runner needed)", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [],
        selectedRunner: null,
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.create")).not.toBeDisabled();
    });

    it("should show loading state on create button when submitting", () => {
      setupSubmitState();
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
        loading: true,
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.creating")).toBeInTheDocument();
    });
  });

  describe("error handling", () => {
    it("should display form error when present", () => {
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        error: "Failed to create pod",
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByRole("alert")).toHaveTextContent("Failed to create pod");
    });

    it("should call onError when submission fails", async () => {
      const onError = vi.fn();
      const error = new Error("Network error");
      mockFormSubmit.mockRejectedValue(error);

      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
        isValid: true,
      });

      render(<CreatePodForm config={{ scenario: "workspace", onError }} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        expect(onError).toHaveBeenCalledWith(error);
      });
    });
  });

  describe("form reset on disable", () => {
    it("should reset form when enabled changes from true to false", () => {
      const { rerender } = render(<CreatePodForm enabled={true} config={{ scenario: "workspace" }} />);
      rerender(<CreatePodForm enabled={false} config={{ scenario: "workspace" }} />);

      expect(mockFormReset).toHaveBeenCalled();
      expect(mockResetPluginConfig).toHaveBeenCalled();
      expect(mockSetSelectedRunnerId).toHaveBeenCalledWith(null);
    });

    it("should not reset form when enabled stays true", () => {
      const { rerender } = render(<CreatePodForm enabled={true} config={{ scenario: "workspace" }} />);
      vi.clearAllMocks();
      rerender(<CreatePodForm enabled={true} config={{ scenario: "workspace" }} />);

      expect(mockFormReset).not.toHaveBeenCalled();
    });
  });

  describe("default prompt initialization", () => {
    it("should initialize prompt with default value for ticket scenario", async () => {
      vi.mocked(useCreatePodForm).mockReturnValue({ ...defaultFormState, prompt: "" });

      render(
        <CreatePodForm
          enabled={true}
          config={{
            scenario: "ticket",
            context: { ticket: { id: 1, slug: "PROJ-123", title: "Test ticket" } },
          }}
        />
      );

      await waitFor(() => {
        expect(mockSetPrompt).toHaveBeenCalledWith("Work on ticket PROJ-123: Test ticket");
      });
    });
  });
});
