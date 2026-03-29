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

describe("RepositoryDetailPage - Webhook, Edit & Variants", () => {
  const mockRepository = {
    id: 1,
    organization_id: 1,
    provider_type: "github",
    provider_base_url: "https://github.com",
    clone_url: "https://github.com/org/my-repo.git",
    external_id: "12345",
    name: "my-repo",
    slug: "org/my-repo",
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
      mockRepositoryApi.getWebhookStatus.mockResolvedValueOnce({
        webhook_status: {
          registered: false,
          is_active: false,
          needs_manual_setup: false,
        },
      });

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

      await waitFor(() => {
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
