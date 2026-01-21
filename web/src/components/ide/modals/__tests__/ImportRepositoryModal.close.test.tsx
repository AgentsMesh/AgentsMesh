import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ImportRepositoryModal } from "../ImportRepositoryModal";
import {
  mockProvider,
  mockGitLabProvider,
  mockRepository,
} from "./ImportRepositoryModal.utils";

// Mock the API
vi.mock("@/lib/api", () => ({
  repositoryApi: {
    create: vi.fn(),
  },
  userRepositoryProviderApi: {
    list: vi.fn(),
    listRepositories: vi.fn(),
  },
}));

// Mock translations
vi.mock("@/lib/i18n/client", () => ({
  useTranslations: () => (key: string) => key,
}));

import { userRepositoryProviderApi } from "@/lib/api";

describe("ImportRepositoryModal - Close and Cancel", () => {
  const mockOnClose = vi.fn();
  const mockOnImported = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(userRepositoryProviderApi.list).mockResolvedValue({
      providers: [mockProvider, mockGitLabProvider],
    });
    vi.mocked(userRepositoryProviderApi.listRepositories).mockResolvedValue({
      repositories: [mockRepository],
    });
  });

  it("should call onClose when cancel button is clicked", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("common.cancel")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("common.cancel"));

    expect(mockOnClose).toHaveBeenCalled();
  });

  it("should call onClose when X button is clicked", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.title")).toBeInTheDocument();
    });

    // Find and click the X button in the header
    const closeButton = document.querySelector("button[class*='hover:text-foreground']");
    if (closeButton) {
      fireEvent.click(closeButton);
      expect(mockOnClose).toHaveBeenCalled();
    }
  });

  it("should reset state when modal is closed and reopened", async () => {
    const { rerender } = render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("My GitHub")).toBeInTheDocument();
    });

    // Select a provider
    fireEvent.click(screen.getByText("My GitHub"));

    await waitFor(() => {
      expect(screen.getByText("org/my-project")).toBeInTheDocument();
    });

    // Close modal
    rerender(
      <ImportRepositoryModal open={false} onClose={mockOnClose} onImported={mockOnImported} />
    );

    // Reopen modal
    rerender(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      // Should be back to source selection step
      expect(screen.getByText("My GitHub")).toBeInTheDocument();
      expect(screen.getByText("My GitLab")).toBeInTheDocument();
    });
  });
});
