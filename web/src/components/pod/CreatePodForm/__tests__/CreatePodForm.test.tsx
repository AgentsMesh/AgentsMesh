import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CreatePodForm } from "../index";
import { CreatePodFormConfig } from "../types";
import {
  mockSetSelectedRunnerId,
  mockFormReset,
  mockFormSubmit,
  mockSetPrompt,
  mockSetSelectedAgent,
  mockResetPluginConfig,
  defaultPodCreationData,
  defaultFormState,
  defaultConfigOptions,
  mockRunner,
  mockAgentType,
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

vi.mock("@/lib/i18n/client", () => ({
  useTranslations: () => (key: string) => key,
}));

vi.mock("@/components/ide/ConfigForm", () => ({
  ConfigForm: () => <div data-testid="config-form">Config Form</div>,
}));

import { usePodCreationData, useCreatePodForm } from "../../hooks";
import { useConfigOptions } from "@/components/ide/hooks";

describe("CreatePodForm", () => {
  beforeEach(() => {
    clearAllMocks();
    vi.clearAllMocks();
  });

  describe("rendering", () => {
    it("should render loading state when data is loading", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        loading: true,
      });

      const { container } = render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });

    it("should render runner select when data is loaded", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectRunner")).toBeInTheDocument();
    });

    it("should show no runners message when no runners available", () => {
      vi.mocked(usePodCreationData).mockReturnValue(defaultPodCreationData);

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.noRunnersAvailable")).toBeInTheDocument();
    });

    it("should show agent select after runner is selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgentTypes: [mockAgentType],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectAgent")).toBeInTheDocument();
    });

    it("should show no agents message when runner has no supported agents", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgentTypes: [],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.noAgentsForRunner")).toBeInTheDocument();
    });

    it("should apply custom className to container", () => {
      const { container } = render(
        <CreatePodForm config={{ scenario: "workspace" }} className="custom-class" />
      );
      expect(container.firstChild).toHaveClass("custom-class");
    });
  });

  describe("runner selection", () => {
    it("should call setSelectedRunnerId when runner is selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner, { ...mockRunner, id: 2, node_id: "runner-2" }],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectRunner"), { target: { value: "1" } });
      expect(mockSetSelectedRunnerId).toHaveBeenCalledWith(1);
    });

    it("should call setSelectedRunnerId with null when deselected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectRunner"), { target: { value: "" } });
      expect(mockSetSelectedRunnerId).toHaveBeenCalledWith(null);
    });

    it("should show runner validation error", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        validationErrors: { runner: "Runner is required" },
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("Runner is required")).toBeInTheDocument();
    });
  });

  describe("agent selection", () => {
    it("should call setSelectedAgent when agent is selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgentTypes: [mockAgentType],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectAgent"), { target: { value: "1" } });
      expect(mockSetSelectedAgent).toHaveBeenCalledWith(1);
    });

    it("should show agent validation error", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgentTypes: [mockAgentType],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        validationErrors: { agent: "Agent is required" },
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("Agent is required")).toBeInTheDocument();
    });
  });

  describe("form submission", () => {
    const setupSubmitState = () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        selectedRunner: mockRunner,
        availableAgentTypes: [mockAgentType],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: 1,
        prompt: "test prompt",
        selectedAgentSlug: "claude-code",
        isValid: true,
      });
    };

    it("should call submit with correct parameters", async () => {
      setupSubmitState();
      mockFormSubmit.mockResolvedValue({ pod_key: "test-pod" });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        expect(mockFormSubmit).toHaveBeenCalledWith(1, {}, { ticketId: undefined, initialPrompt: "test prompt" });
      });
    });

    it("should pass ticketId when in ticket scenario", async () => {
      setupSubmitState();
      mockFormSubmit.mockResolvedValue({ pod_key: "test-pod" });

      const config: CreatePodFormConfig = {
        scenario: "ticket",
        context: { ticket: { id: 123, identifier: "PROJ-123", title: "Test" } },
      };

      render(<CreatePodForm config={config} />);
      fireEvent.click(screen.getByText("ide.createPod.create"));

      await waitFor(() => {
        expect(mockFormSubmit).toHaveBeenCalledWith(1, {}, { ticketId: 123, initialPrompt: "test prompt" });
      });
    });

    it("should disable create button when no runner selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue(defaultPodCreationData);
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.create")).toBeDisabled();
    });

    it("should show loading state on create button when submitting", () => {
      setupSubmitState();
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: 1,
        loading: true,
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.creating")).toBeInTheDocument();
    });
  });

  describe("cancel button", () => {
    it("should render cancel button when onCancel is provided", () => {
      render(<CreatePodForm config={{ scenario: "workspace", onCancel: vi.fn() }} />);
      expect(screen.getByText("ide.createPod.cancel")).toBeInTheDocument();
    });

    it("should call onCancel when clicked", () => {
      const onCancel = vi.fn();
      render(<CreatePodForm config={{ scenario: "workspace", onCancel }} />);
      fireEvent.click(screen.getByText("ide.createPod.cancel"));
      expect(onCancel).toHaveBeenCalled();
    });

    it("should not render cancel button when onCancel is not provided", () => {
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.queryByText("ide.createPod.cancel")).not.toBeInTheDocument();
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
        availableAgentTypes: [mockAgentType],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: 1,
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
            context: { ticket: { id: 1, identifier: "PROJ-123", title: "Test ticket" } },
          }}
        />
      );

      await waitFor(() => {
        expect(mockSetPrompt).toHaveBeenCalledWith("Work on ticket PROJ-123: Test ticket");
      });
    });
  });
});
