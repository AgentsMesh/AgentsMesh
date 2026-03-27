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

// Mock Collapsible to always render children (no collapse animation in tests)
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

    it("should render agent select when agents are available", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [mockAgent],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectAgent")).toBeInTheDocument();
    });

    it("should show no agents message when no agents available", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.noAgentsForRunner")).toBeInTheDocument();
    });

    it("should show runner select inside advanced options when agent is selected", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectRunner")).toBeInTheDocument();
    });

    it("should show no runners message inside advanced options when no runners available", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [],
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.noRunnersAvailable")).toBeInTheDocument();
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
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
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
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectRunner"), { target: { value: "" } });
      expect(mockSetSelectedRunnerId).toHaveBeenCalledWith(null);
    });

    it("should show runner validation error", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        selectedAgent: "claude-code",
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
        availableAgents: [mockAgent],
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectAgent"), { target: { value: "claude-code" } });
      expect(mockSetSelectedAgent).toHaveBeenCalledWith("claude-code");
    });

    it("should show agent validation error", () => {
      vi.mocked(usePodCreationData).mockReturnValue({
        ...defaultPodCreationData,
        runners: [mockRunner],
        availableAgents: [mockAgent],
      });
      vi.mocked(useCreatePodForm).mockReturnValue({
        ...defaultFormState,
        validationErrors: { agent: "Agent is required" },
      });

      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("Agent is required")).toBeInTheDocument();
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

});
