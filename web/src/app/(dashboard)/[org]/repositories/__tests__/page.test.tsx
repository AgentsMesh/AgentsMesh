import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";
import RepositoriesPage from "../page";

// Mock next/link
vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

// Mock next/navigation
vi.mock("next/navigation", () => ({
  useParams: () => ({ org: "test-org" }),
  useRouter: () => ({ push: vi.fn() }),
}));

// Mock the API modules
vi.mock("@/lib/api", () => ({
  repositoryApi: {
    list: vi.fn(),
    delete: vi.fn(),
    create: vi.fn(),
  },
  userRepositoryProviderApi: {
    list: vi.fn(),
    listRepositories: vi.fn(),
  },
}));

import { repositoryApi, userRepositoryProviderApi } from "@/lib/api";
const mockRepositoryApi = vi.mocked(repositoryApi);
const mockUserRepositoryProviderApi = vi.mocked(userRepositoryProviderApi);

describe("RepositoriesPage", () => {
  const mockRepositories = [
    {
      id: 1,
      organization_id: 1,
      provider_type: "github",
      provider_base_url: "https://github.com",
      clone_url: "https://github.com/org/repo-one.git",
      external_id: "12345",
      name: "repo-one",
      slug: "org/repo-one",
      default_branch: "main",
      ticket_prefix: "REPO",
      visibility: "organization",
      is_active: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
    {
      id: 2,
      organization_id: 1,
      provider_type: "gitlab",
      provider_base_url: "https://gitlab.com",
      clone_url: "https://gitlab.com/org/repo-two.git",
      external_id: "67890",
      name: "repo-two",
      slug: "org/repo-two",
      default_branch: "develop",
      visibility: "organization",
      is_active: true,
      created_at: "2024-01-02T00:00:00Z",
      updated_at: "2024-01-02T00:00:00Z",
    },
    {
      id: 3,
      organization_id: 1,
      provider_type: "github",
      provider_base_url: "https://github.com",
      clone_url: "https://github.com/org/inactive-repo.git",
      external_id: "11111",
      name: "inactive-repo",
      slug: "org/inactive-repo",
      default_branch: "main",
      visibility: "private",
      is_active: false,
      created_at: "2024-01-03T00:00:00Z",
      updated_at: "2024-01-03T00:00:00Z",
    },
  ];

  const mockProviders = [
    {
      id: 1,
      user_id: 1,
      provider_type: "github",
      name: "GitHub",
      base_url: "https://github.com",
      has_client_id: false,
      has_bot_token: false,
      has_identity: true,
      is_default: false,
      is_active: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
    {
      id: 2,
      user_id: 1,
      provider_type: "gitlab",
      name: "Company GitLab",
      base_url: "https://gitlab.company.com",
      has_client_id: false,
      has_bot_token: true,
      has_identity: false,
      is_default: false,
      is_active: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockRepositoryApi.list.mockResolvedValue({ repositories: mockRepositories });
    mockUserRepositoryProviderApi.list.mockResolvedValue({ providers: mockProviders });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("loading state", () => {
    it("should show loading spinner initially", () => {
      // Keep promise pending
      mockRepositoryApi.list.mockImplementation(() => new Promise(() => {}));

      const { container } = render(<RepositoriesPage />);

      // Check for spinner via class
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });
  });

  describe("rendering", () => {
    it("should render page title", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Repositories")).toBeInTheDocument();
      });
    });

    it("should render page description", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Manage your Git repositories for AgentPod")).toBeInTheDocument();
      });
    });

    it("should render Import Repository button", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });
    });

    it("should render stats cards", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Total Repositories")).toBeInTheDocument();
        // "Active" appears in stats card label and in status badges for repositories
        expect(screen.getAllByText("Active").length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText("Providers")).toBeInTheDocument();
      });
    });

    it("should show correct stats values", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        // Total: 3, Active: 2, Providers: 2
        const stats = screen.getAllByText("2");
        expect(stats.length).toBeGreaterThanOrEqual(2);
      });
    });
  });

  describe("repository list", () => {
    it("should render all repositories", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
        expect(screen.getByText("repo-two")).toBeInTheDocument();
        expect(screen.getByText("inactive-repo")).toBeInTheDocument();
      });
    });

    it("should show full path for each repository", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("org/repo-one")).toBeInTheDocument();
        expect(screen.getByText("org/repo-two")).toBeInTheDocument();
      });
    });

    it("should show inactive badge for inactive repositories", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Inactive")).toBeInTheDocument();
      });
    });

    it("should show active/inactive status badges", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        // Check for active and inactive status badges
        const activeBadges = screen.getAllByText("Active");
        expect(activeBadges.length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText("Inactive")).toBeInTheDocument();
      });
    });

    it("should show default branch for each repository", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        // There can be multiple "main" branches from different repos
        expect(screen.getAllByText("main").length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText("develop")).toBeInTheDocument();
      });
    });

    it("should show provider type for each repository", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Provider types are shown with capitalize CSS class but inner text is lowercase
      // They also appear in the filter dropdown, so use getAllByText
      expect(screen.getAllByText("github").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("gitlab").length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("filtering", () => {
    it("should filter repositories by search text", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      fireEvent.change(screen.getByPlaceholderText("Search repositories..."), {
        target: { value: "repo-one" },
      });

      expect(screen.getByText("repo-one")).toBeInTheDocument();
      expect(screen.queryByText("repo-two")).not.toBeInTheDocument();
    });

    it("should filter repositories by provider type", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Find the provider filter select
      const providerSelect = screen.getByRole("combobox");
      fireEvent.change(providerSelect, { target: { value: "gitlab" } });

      await waitFor(() => {
        expect(screen.queryByText("repo-one")).not.toBeInTheDocument();
        expect(screen.getByText("repo-two")).toBeInTheDocument();
      });
    });

    it("should show empty state when no matches", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      fireEvent.change(screen.getByPlaceholderText("Search repositories..."), {
        target: { value: "nonexistent" },
      });

      expect(screen.getByText("No repositories match your search")).toBeInTheDocument();
    });
  });

  describe("delete functionality", () => {
    it("should show confirm dialog when delete button clicked", async () => {
      mockRepositoryApi.delete.mockResolvedValue({ message: "Deleted" });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Find and click the first delete button
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // ConfirmDialog should be shown
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });
    });

    it("should call delete API when confirmed in dialog", async () => {
      mockRepositoryApi.delete.mockResolvedValue({ message: "Deleted" });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Click delete button to open dialog
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // Wait for dialog and click confirm
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });

      // Click the confirm button in the dialog (there should be a "Delete" button in dialog footer)
      // Note: The first "Delete" button opens the dialog, then there are two "Delete" buttons visible
      const confirmButtons = screen.getAllByRole("button", { name: "Delete" });
      // The last "Delete" button is the confirm button in the dialog
      fireEvent.click(confirmButtons[confirmButtons.length - 1]);

      await waitFor(() => {
        expect(mockRepositoryApi.delete).toHaveBeenCalledWith(1);
      });
    });

    it("should not call delete API when cancelled in dialog", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Click delete button to open dialog
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      // Wait for dialog
      await waitFor(() => {
        expect(screen.getByText("Delete Repository")).toBeInTheDocument();
      });

      // Click cancel button
      const cancelButton = screen.getByRole("button", { name: "Cancel" });
      fireEvent.click(cancelButton);

      // Dialog should close and delete should not be called
      await waitFor(() => {
        expect(screen.queryByText("Delete Repository")).not.toBeInTheDocument();
      });
      expect(mockRepositoryApi.delete).not.toHaveBeenCalled();
    });
  });

});
