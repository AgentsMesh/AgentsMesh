import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";
import RepositoryDetailPage from "../page";

// Mock next/navigation
const mockPush = vi.fn();
vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "1" }),
  useRouter: () => ({ push: mockPush }),
}));

// Mock next/link
vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

// Mock the API modules
vi.mock("@/lib/api", () => ({
  repositoryApi: {
    get: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    registerWebhook: vi.fn(),
    getWebhookStatus: vi.fn(),
    getWebhookSecret: vi.fn(),
    deleteWebhook: vi.fn(),
    markWebhookConfigured: vi.fn(),
  },
}));

import { repositoryApi } from "@/lib/api";
const mockRepositoryApi = vi.mocked(repositoryApi);

describe("RepositoryDetailPage", () => {
  // New self-contained repository model (no git_provider_id)
  const mockRepository = {
    id: 1,
    organization_id: 1,
    provider_type: "github",
    provider_base_url: "https://github.com",
    clone_url: "https://github.com/org/my-repo.git",
    external_id: "12345",
    name: "my-repo",
    full_path: "org/my-repo",
    default_branch: "main",
    ticket_prefix: "PROJ",
    visibility: "organization",
    is_active: true,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockRepositoryApi.get.mockResolvedValue({ repository: mockRepository });
    mockRepositoryApi.registerWebhook.mockResolvedValue({
      result: {
        repo_id: 123,
        registered: true,
        webhook_id: "wh_123",
        needs_manual_setup: false,
      },
    });
    mockRepositoryApi.getWebhookStatus.mockResolvedValue({
      webhook_status: {
        registered: false,
        is_active: false,
        needs_manual_setup: false,
      },
    });
    mockRepositoryApi.delete.mockResolvedValue({ message: "Deleted" });
    mockRepositoryApi.update.mockResolvedValue({ repository: mockRepository });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("loading state", () => {
    it("should show loading spinner initially", () => {
      mockRepositoryApi.get.mockImplementation(() => new Promise(() => {}));

      render(<RepositoryDetailPage />);

      expect(document.querySelector(".animate-spin")).toBeTruthy();
    });
  });

  describe("not found state", () => {
    let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      // Suppress expected console.error for not found tests
      consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
      consoleErrorSpy.mockRestore();
    });

    it("should show not found message when repository not found", async () => {
      mockRepositoryApi.get.mockRejectedValue(new Error("Not found"));

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Repository not found")).toBeInTheDocument();
      });
    });

    it("should show back button when not found", async () => {
      mockRepositoryApi.get.mockRejectedValue(new Error("Not found"));

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Back to Repositories")).toBeInTheDocument();
      });
    });
  });

  describe("rendering", () => {
    it("should render repository name", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        // Multiple instances of name appear (header, breadcrumb, details)
        expect(screen.getAllByText("my-repo").length).toBeGreaterThanOrEqual(1);
      });
    });

    it("should render repository full path", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        // Multiple instances of path appear (header, details)
        expect(screen.getAllByText("org/my-repo").length).toBeGreaterThanOrEqual(1);
      });
    });

    it("should render Edit and Delete buttons", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Edit")).toBeInTheDocument();
        expect(screen.getByText("Delete")).toBeInTheDocument();
      });
    });

    it("should render breadcrumb", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByRole("link", { name: "Repositories" })).toBeInTheDocument();
      });
    });

    it("should render tabs", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Information")).toBeInTheDocument();
        expect(screen.getByText("Extensions")).toBeInTheDocument();
      });
    });
  });

  describe("information tab", () => {
    it("should show repository details section", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Repository Details")).toBeInTheDocument();
      });
    });

    it("should show default branch", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Default Branch")).toBeInTheDocument();
        expect(screen.getByText("main")).toBeInTheDocument();
      });
    });

    it("should show clone URL", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Clone URL")).toBeInTheDocument();
        expect(screen.getByText("https://github.com/org/my-repo.git")).toBeInTheDocument();
      });
    });

    it("should show ticket prefix", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Ticket Prefix")).toBeInTheDocument();
        expect(screen.getByText("PROJ")).toBeInTheDocument();
      });
    });

    it("should show active status", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Status")).toBeInTheDocument();
        expect(screen.getByText("Active")).toBeInTheDocument();
      });
    });

    it("should show git provider info from self-contained fields", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Git Provider")).toBeInTheDocument();
        expect(screen.getByText("github")).toBeInTheDocument();
        expect(screen.getByText("https://github.com")).toBeInTheDocument();
      });
    });

    it("should show visibility", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Visibility")).toBeInTheDocument();
        expect(screen.getByText("organization")).toBeInTheDocument();
      });
    });

    it("should show webhook settings section", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Webhook Settings")).toBeInTheDocument();
        expect(screen.getByText("Register Webhook")).toBeInTheDocument();
      });
    });
  });

  describe("extensions tab", () => {
    it("should show extensions tab button", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Extensions")).toBeInTheDocument();
      });
    });
  });

  describe("delete functionality", () => {
    it("should show confirm dialog when Delete clicked", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Delete")).toBeInTheDocument();
      });

      // Click the delete button in the header
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // Confirm dialog should appear
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });
    });

    it("should call delete API and navigate when confirmed", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Delete")).toBeInTheDocument();
      });

      // Click the delete button in the header
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // Wait for confirm dialog
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });

      // Click the confirm button in the dialog
      const confirmButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(confirmButtons[confirmButtons.length - 1]);

      await waitFor(() => {
        expect(mockRepositoryApi.delete).toHaveBeenCalledWith(1);
      });

      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledWith("../repositories");
      });
    });

    it("should not delete when cancelled", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Delete")).toBeInTheDocument();
      });

      // Click the delete button in the header
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // Wait for confirm dialog
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });

      // Click the cancel button
      fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

      // Dialog should close and delete should not be called
      await waitFor(() => {
        expect(screen.queryByText("Delete Repository")).not.toBeInTheDocument();
      });
      expect(mockRepositoryApi.delete).not.toHaveBeenCalled();
    });
  });

  describe("webhook setup", () => {
    it("should call registerWebhook API when button clicked", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Register Webhook")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Register Webhook"));

      await waitFor(() => {
        expect(mockRepositoryApi.registerWebhook).toHaveBeenCalledWith(1);
      });
    });

    it("should refresh webhook status after successful registration", async () => {
      // Initial status: not registered
      mockRepositoryApi.getWebhookStatus.mockResolvedValueOnce({
        webhook_status: {
          registered: false,
          is_active: false,
          needs_manual_setup: false,
        },
      });

      // After registration: registered and active
      mockRepositoryApi.getWebhookStatus.mockResolvedValueOnce({
        webhook_status: {
          registered: true,
          is_active: true,
          needs_manual_setup: false,
          webhook_id: "wh_123",
        },
      });

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Register Webhook")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Register Webhook"));

      // Should refresh status after registration
      await waitFor(() => {
        // getWebhookStatus called twice: once on load, once after registration
        expect(mockRepositoryApi.getWebhookStatus).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe("edit modal", () => {
    it("should open edit modal when Edit clicked", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Edit")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Edit"));

      expect(screen.getByText("Edit Repository")).toBeInTheDocument();
    });

    it("should close edit modal when Cancel clicked", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Edit")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Edit"));
      fireEvent.click(screen.getByText("Cancel"));

      await waitFor(() => {
        expect(screen.queryByText("Edit Repository")).not.toBeInTheDocument();
      });
    });

    it("should call update API when save clicked", async () => {
      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Edit")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Edit"));

      // Change the name
      const nameInput = screen.getByDisplayValue("my-repo");
      fireEvent.change(nameInput, { target: { value: "updated-repo" } });

      fireEvent.click(screen.getByText("Save Changes"));

      await waitFor(() => {
        expect(mockRepositoryApi.update).toHaveBeenCalledWith(1, expect.objectContaining({
          name: "updated-repo",
        }));
      });
    });
  });

  describe("inactive repository", () => {
    it("should show Inactive badge for inactive repository", async () => {
      mockRepositoryApi.get.mockResolvedValue({
        repository: { ...mockRepository, is_active: false },
      });

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        // Multiple "Inactive" elements: header badge and status section
        expect(screen.getAllByText("Inactive").length).toBeGreaterThanOrEqual(1);
      });
    });
  });

  describe("private visibility repository", () => {
    it("should show Private badge for private visibility repository", async () => {
      mockRepositoryApi.get.mockResolvedValue({
        repository: { ...mockRepository, visibility: "private" },
      });

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("Private")).toBeInTheDocument();
      });
    });
  });

  describe("different providers", () => {
    it("should show GitLab provider type", async () => {
      mockRepositoryApi.get.mockResolvedValue({
        repository: {
          ...mockRepository,
          provider_type: "gitlab",
          provider_base_url: "https://gitlab.com",
        },
      });

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("gitlab")).toBeInTheDocument();
        expect(screen.getByText("https://gitlab.com")).toBeInTheDocument();
      });
    });

    it("should show Gitee provider type", async () => {
      mockRepositoryApi.get.mockResolvedValue({
        repository: {
          ...mockRepository,
          provider_type: "gitee",
          provider_base_url: "https://gitee.com",
        },
      });

      render(<RepositoryDetailPage />);

      await waitFor(() => {
        expect(screen.getByText("gitee")).toBeInTheDocument();
        expect(screen.getByText("https://gitee.com")).toBeInTheDocument();
      });
    });
  });
});
