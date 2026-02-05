import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";
import RepositoriesPage from "../page";

// Mock next/link
vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
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
      full_path: "org/repo-one",
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
      full_path: "org/repo-two",
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
      full_path: "org/inactive-repo",
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
    // Mock window.confirm
    vi.spyOn(window, "confirm").mockReturnValue(true);
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
    it("should call delete API when delete button clicked and confirmed", async () => {
      mockRepositoryApi.delete.mockResolvedValue({ message: "Deleted" });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Find and click the first delete button (text-based)
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      expect(window.confirm).toHaveBeenCalled();
      await waitFor(() => {
        expect(mockRepositoryApi.delete).toHaveBeenCalledWith(1);
      });
    });

    it("should not call delete API when cancelled", async () => {
      vi.spyOn(window, "confirm").mockReturnValue(false);

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      fireEvent.click(deleteButtons[0]);

      expect(window.confirm).toHaveBeenCalled();
      expect(mockRepositoryApi.delete).not.toHaveBeenCalled();
    });
  });

  describe("empty state", () => {
    it("should show empty state when no repositories", async () => {
      mockRepositoryApi.list.mockResolvedValue({ repositories: [] });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("No repositories yet")).toBeInTheDocument();
      });
    });

    it("should still show import button in empty state", async () => {
      mockRepositoryApi.list.mockResolvedValue({ repositories: [] });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });
    });
  });

  describe("import modal", () => {
    it("should open import modal when Import Repository clicked", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Import Repository"));

      // The modal header should appear
      await waitFor(() => {
        expect(screen.getByText("Import Repository", { selector: "h2" })).toBeInTheDocument();
      });
    });

    it("should close import modal when Cancel clicked", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Import Repository"));

      await waitFor(() => {
        expect(screen.getByText("Cancel")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Cancel"));

      await waitFor(() => {
        expect(screen.queryByText("Import Repository", { selector: "h2" })).not.toBeInTheDocument();
      });
    });

    it("should show repository providers in import modal", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Import Repository"));

      await waitFor(() => {
        expect(screen.getByText("Your Git Connections")).toBeInTheDocument();
        expect(screen.getByText("GitHub")).toBeInTheDocument();
        expect(screen.getByText("Company GitLab")).toBeInTheDocument();
      });
    });

    it("should show manual entry option", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Import Repository"));

      await waitFor(() => {
        expect(screen.getByText("Enter Manually")).toBeInTheDocument();
      });
    });

    it("should show message when no providers available", async () => {
      mockUserRepositoryProviderApi.list.mockResolvedValue({ providers: [] });

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("Import Repository")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("Import Repository"));

      await waitFor(() => {
        expect(screen.getByText(/No Git connections configured/)).toBeInTheDocument();
      });
    });
  });

  describe("links", () => {
    it("should link to repository detail page via name click", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      const link = screen.getByRole("link", { name: "repo-one" });
      expect(link).toHaveAttribute("href", "repositories/1");
    });

    it("should have links to all repositories", async () => {
      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(screen.getByText("repo-one")).toBeInTheDocument();
      });

      // Verify all repository names are links
      const repoOneLink = screen.getByRole("link", { name: "repo-one" });
      const repoTwoLink = screen.getByRole("link", { name: "repo-two" });
      const inactiveLink = screen.getByRole("link", { name: "inactive-repo" });

      expect(repoOneLink).toHaveAttribute("href", "repositories/1");
      expect(repoTwoLink).toHaveAttribute("href", "repositories/2");
      expect(inactiveLink).toHaveAttribute("href", "repositories/3");
    });
  });

  describe("error handling", () => {
    it("should handle API errors gracefully", async () => {
      mockRepositoryApi.list.mockRejectedValue(new Error("Network error"));
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      render(<RepositoriesPage />);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith("Failed to load data:", expect.any(Error));
      });

      consoleSpy.mockRestore();
    });
  });
});
