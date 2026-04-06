import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CreatePodForm } from "../index";
import {
  mockSetPrompt,
  mockSetAlias,
  defaultPodCreationData,
  defaultFormState,
  defaultConfigOptions,
  mockRunner,
  mockAgent,
  mockRepository,
  mockCredentialProfile,
  clearAllMocks,
} from "./test-utils";

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
import { useConfigOptions } from "@/components/ide/hooks";

describe("CreatePodForm - Agent Configuration", () => {
  beforeEach(() => {
    clearAllMocks();
    vi.clearAllMocks();
  });

  const setupAgentSelectedState = (overrides = {}) => {
    const mockSetSelectedRepository = vi.fn();
    const mockSetSelectedBranch = vi.fn();
    const mockSetSelectedCredentialProfile = vi.fn();

    vi.mocked(usePodCreationData).mockReturnValue({
      ...defaultPodCreationData,
      runners: [mockRunner],
      repositories: [mockRepository, { ...mockRepository, id: 2, slug: "org/repo2" }],
      selectedRunner: mockRunner,
      availableAgents: [mockAgent],
    });

    vi.mocked(useCreatePodForm).mockReturnValue({
      ...defaultFormState,
      selectedAgent: "claude-code",
      credentialProfiles: [
        { ...mockCredentialProfile, id: 1, name: "My Credentials", is_default: false },
        { ...mockCredentialProfile, id: 2, name: "Default Creds", is_default: true },
      ],
      setSelectedRepository: mockSetSelectedRepository,
      setSelectedBranch: mockSetSelectedBranch,
      setSelectedCredentialProfile: mockSetSelectedCredentialProfile,
      selectedAgentSlug: "claude-code",
      isValid: true,
      ...overrides,
    });

    return { mockSetSelectedRepository, mockSetSelectedBranch, mockSetSelectedCredentialProfile };
  };

  describe("credential selection", () => {
    it("should render credential select when agent is selected", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectCredential")).toBeInTheDocument();
    });

    it("should render credential profiles in select", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("My Credentials")).toBeInTheDocument();
      expect(screen.getByText("Default Creds (settings.agentCredentials.default)")).toBeInTheDocument();
    });

    it("should call setSelectedCredentialProfile when changed", () => {
      const { mockSetSelectedCredentialProfile } = setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectCredential"), { target: { value: "1" } });
      expect(mockSetSelectedCredentialProfile).toHaveBeenCalledWith(1);
    });

    it("should show runner host hint when runner host profile is selected", () => {
      setupAgentSelectedState({ selectedCredentialProfile: 0 });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.runnerHostHint")).toBeInTheDocument();
    });

    it("should show custom credential hint when custom profile is selected", () => {
      setupAgentSelectedState({ selectedCredentialProfile: 1 });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.customCredentialHint")).toBeInTheDocument();
    });

    it("should show loading state for credentials", () => {
      setupAgentSelectedState({ loadingCredentials: true });
      const { container } = render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("common.loading")).toBeInTheDocument();
      expect(container.querySelectorAll(".animate-spin").length).toBeGreaterThan(0);
    });
  });

  describe("repository selection", () => {
    it("should render repository select when agent is selected", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.selectRepository")).toBeInTheDocument();
    });

    it("should render repositories in select", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("org/repo1")).toBeInTheDocument();
      expect(screen.getByText("org/repo2")).toBeInTheDocument();
    });

    it("should call setSelectedRepository when changed", () => {
      const { mockSetSelectedRepository } = setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectRepository"), { target: { value: "1" } });
      expect(mockSetSelectedRepository).toHaveBeenCalledWith(1);
    });

    it("should call setSelectedRepository with null when deselected", () => {
      const { mockSetSelectedRepository } = setupAgentSelectedState({ selectedRepository: 1 });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.selectRepository"), { target: { value: "" } });
      expect(mockSetSelectedRepository).toHaveBeenCalledWith(null);
    });
  });

  describe("branch input", () => {
    it("should render branch input when repository is selected", () => {
      setupAgentSelectedState({ selectedRepository: 1 });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.branch")).toBeInTheDocument();
    });

    it("should not render branch input when no repository is selected", () => {
      setupAgentSelectedState({ selectedRepository: null });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.queryByLabelText("ide.createPod.branch")).not.toBeInTheDocument();
    });

    it("should call setSelectedBranch when changed", () => {
      const { mockSetSelectedBranch } = setupAgentSelectedState({ selectedRepository: 1 });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.branch"), { target: { value: "feature/test" } });
      expect(mockSetSelectedBranch).toHaveBeenCalledWith("feature/test");
    });

    it("should show branch validation error", () => {
      setupAgentSelectedState({
        selectedRepository: 1,
        validationErrors: { branch: "Branch is required" },
      });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("Branch is required")).toBeInTheDocument();
    });
  });

  describe("prompt textarea", () => {
    it("should render prompt textarea when agent is selected", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.prompt")).toBeInTheDocument();
    });

    it("should use custom placeholder when provided", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace", promptPlaceholder: "Custom placeholder" }} />);
      expect(screen.getByPlaceholderText("Custom placeholder")).toBeInTheDocument();
    });

    it("should call setPrompt when changed", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.prompt"), { target: { value: "New prompt" } });
      expect(mockSetPrompt).toHaveBeenCalledWith("New prompt");
    });
  });

  describe("alias input", () => {
    it("should render alias input when agent is selected", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.alias")).toBeInTheDocument();
    });

    it("should call setAlias when changed", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      fireEvent.change(screen.getByLabelText("ide.createPod.alias"), { target: { value: "my-pod" } });
      expect(mockSetAlias).toHaveBeenCalledWith("my-pod");
    });

    it("should show alias value from form state", () => {
      setupAgentSelectedState({ alias: "existing-alias" });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.alias")).toHaveValue("existing-alias");
    });

    it("should have maxLength of 100", () => {
      setupAgentSelectedState();
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByLabelText("ide.createPod.alias")).toHaveAttribute("maxLength", "100");
    });

    it("should not render alias input when no agent is selected", () => {
      vi.mocked(useCreatePodForm).mockReturnValue(defaultFormState);
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.queryByLabelText("ide.createPod.alias")).not.toBeInTheDocument();
    });
  });

  describe("config options", () => {
    it("should show loading state for config", () => {
      setupAgentSelectedState();
      vi.mocked(useConfigOptions).mockReturnValue({ ...defaultConfigOptions, loading: true });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.loadingPlugins")).toBeInTheDocument();
    });

    it("should render config form when config fields are available", () => {
      setupAgentSelectedState();
      vi.mocked(useConfigOptions).mockReturnValue({
        ...defaultConfigOptions,
        fields: [{ name: "model", type: "select" }],
      });
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.getByText("ide.createPod.pluginConfig")).toBeInTheDocument();
      expect(screen.getByTestId("config-form")).toBeInTheDocument();
    });

    it("should not render config form when no config fields available", () => {
      setupAgentSelectedState();
      vi.mocked(useConfigOptions).mockReturnValue(defaultConfigOptions);
      render(<CreatePodForm config={{ scenario: "workspace" }} />);
      expect(screen.queryByText("ide.createPod.pluginConfig")).not.toBeInTheDocument();
    });
  });
});
