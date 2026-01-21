import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ImportRepositoryModal } from "../ImportRepositoryModal";
import {
  mockProvider,
  mockGitLabProvider,
  mockRepository,
  mockCreatedRepository,
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

import { repositoryApi, userRepositoryProviderApi } from "@/lib/api";

describe("ImportRepositoryModal - Navigation Flow", () => {
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

  it("should complete manual import flow successfully", async () => {
    vi.mocked(repositoryApi.create).mockResolvedValue({
      ...mockCreatedRepository,
      name: "test-repo",
      full_path: "test/repo",
      clone_url: "https://github.com/test/repo.git",
    });

    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("https://github.com/org/repo.git")).toBeInTheDocument();
    });

    // Fill in required fields
    fireEvent.change(screen.getByPlaceholderText("https://github.com/org/repo.git"), {
      target: { value: "https://github.com/test/repo.git" },
    });
    fireEvent.change(screen.getByPlaceholderText("my-project"), {
      target: { value: "test-repo" },
    });
    fireEvent.change(screen.getByPlaceholderText("org/my-project"), {
      target: { value: "test/repo" },
    });

    fireEvent.click(screen.getByText("repositories.modal.continue"));

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.importRepository")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.importRepository"));

    await waitFor(() => {
      expect(repositoryApi.create).toHaveBeenCalledWith(
        expect.objectContaining({
          provider_type: "github",
          clone_url: "https://github.com/test/repo.git",
          name: "test-repo",
          full_path: "test/repo",
        })
      );
      expect(mockOnImported).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  it("should allow changing visibility in confirm step", async () => {
    vi.mocked(repositoryApi.create).mockResolvedValue({
      ...mockCreatedRepository,
      visibility: "private",
    });

    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("My GitHub")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("My GitHub"));

    await waitFor(() => {
      expect(screen.getByText("org/my-project")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("org/my-project"));

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.privateOnly")).toBeInTheDocument();
    });

    // Find and click the private radio button
    const privateRadio = screen.getByText("repositories.modal.privateOnly").previousElementSibling;
    if (privateRadio) {
      fireEvent.click(privateRadio);
    }

    fireEvent.click(screen.getByText("repositories.modal.importRepository"));

    await waitFor(() => {
      expect(repositoryApi.create).toHaveBeenCalledWith(
        expect.objectContaining({
          visibility: "private",
        })
      );
    });
  });
});
