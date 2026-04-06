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

describe("RepositoriesPage - import and links", () => {
  const mockRepositories = [
    {
      id: 1, organization_id: 1, provider_type: "github", provider_base_url: "https://github.com",
      http_clone_url: "https://github.com/org/repo-one.git", external_id: "12345", name: "repo-one",
      slug: "org/repo-one", default_branch: "main", ticket_prefix: "REPO",
      visibility: "organization", is_active: true,
      created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z",
    },
    {
      id: 2, organization_id: 1, provider_type: "gitlab", provider_base_url: "https://gitlab.com",
      http_clone_url: "https://gitlab.com/org/repo-two.git", external_id: "67890", name: "repo-two",
      slug: "org/repo-two", default_branch: "develop", visibility: "organization", is_active: true,
      created_at: "2024-01-02T00:00:00Z", updated_at: "2024-01-02T00:00:00Z",
    },
    {
      id: 3, organization_id: 1, provider_type: "github", provider_base_url: "https://github.com",
      http_clone_url: "https://github.com/org/inactive-repo.git", external_id: "11111", name: "inactive-repo",
      slug: "org/inactive-repo", default_branch: "main", visibility: "private", is_active: false,
      created_at: "2024-01-03T00:00:00Z", updated_at: "2024-01-03T00:00:00Z",
    },
  ];

  const mockProviders = [
    { id: 1, user_id: 1, provider_type: "github", name: "GitHub", base_url: "https://github.com", has_client_id: false, has_bot_token: false, has_identity: true, is_default: false, is_active: true, created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z" },
    { id: 2, user_id: 1, provider_type: "gitlab", name: "Company GitLab", base_url: "https://gitlab.company.com", has_client_id: false, has_bot_token: true, has_identity: false, is_default: false, is_active: true, created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z" },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockRepositoryApi.list.mockResolvedValue({ repositories: mockRepositories });
    mockUserRepositoryProviderApi.list.mockResolvedValue({ providers: mockProviders });
  });

  afterEach(() => {
    vi.resetAllMocks();
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
      await waitFor(() => { expect(screen.getByText("Import Repository")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Import Repository"));
      await waitFor(() => { expect(screen.getByText("Import Repository", { selector: "h2" })).toBeInTheDocument(); });
    });

    it("should close import modal when Cancel clicked", async () => {
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("Import Repository")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Import Repository"));
      await waitFor(() => { expect(screen.getByText("Cancel")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Cancel"));
      await waitFor(() => { expect(screen.queryByText("Import Repository", { selector: "h2" })).not.toBeInTheDocument(); });
    });

    it("should show repository providers in import modal", async () => {
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("Import Repository")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Import Repository"));
      await waitFor(() => {
        expect(screen.getByText("Your Git Connections")).toBeInTheDocument();
        expect(screen.getByText("GitHub")).toBeInTheDocument();
        expect(screen.getByText("Company GitLab")).toBeInTheDocument();
      });
    });

    it("should show manual entry option", async () => {
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("Import Repository")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Import Repository"));
      await waitFor(() => { expect(screen.getByText("Enter Manually")).toBeInTheDocument(); });
    });

    it("should show message when no providers available", async () => {
      mockUserRepositoryProviderApi.list.mockResolvedValue({ providers: [] });
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("Import Repository")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Import Repository"));
      await waitFor(() => { expect(screen.getByText(/No Git connections configured/)).toBeInTheDocument(); });
    });
  });

  describe("links", () => {
    it("should link to repository detail page via name click", async () => {
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("repo-one")).toBeInTheDocument(); });
      const link = screen.getByRole("link", { name: "repo-one" });
      expect(link).toHaveAttribute("href", "/test-org/repositories/1");
    });

    it("should have links to all repositories", async () => {
      render(<RepositoriesPage />);
      await waitFor(() => { expect(screen.getByText("repo-one")).toBeInTheDocument(); });
      expect(screen.getByRole("link", { name: "repo-one" })).toHaveAttribute("href", "/test-org/repositories/1");
      expect(screen.getByRole("link", { name: "repo-two" })).toHaveAttribute("href", "/test-org/repositories/2");
      expect(screen.getByRole("link", { name: "inactive-repo" })).toHaveAttribute("href", "/test-org/repositories/3");
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
