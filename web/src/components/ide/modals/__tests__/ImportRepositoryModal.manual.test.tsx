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

describe("ImportRepositoryModal - Manual Entry", () => {
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

  it("should navigate to manual entry step", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    await waitFor(() => {
      // The label includes " *" for required field marker
      expect(screen.getByText(/repositories\.modal\.cloneUrl/)).toBeInTheDocument();
    });
  });

  it("should show provider type dropdown in manual entry", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.providerType")).toBeInTheDocument();
    });
  });

  it("should update base URL when provider type changes", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    await waitFor(() => {
      const baseUrlInput = screen.getByPlaceholderText("https://github.com") as HTMLInputElement;
      expect(baseUrlInput.value).toBe("https://github.com");
    });
  });

  it("should allow filling manual entry fields", async () => {
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
  });

  it("should show error when continue is clicked without required fields", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.continue")).toBeInTheDocument();
    });

    // Click continue without filling required fields
    fireEvent.click(screen.getByText("repositories.modal.continue"));

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.fillRequiredFields")).toBeInTheDocument();
    });
  });

  it("should navigate to confirm step after filling manual entry", async () => {
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
      expect(screen.getByText("repositories.modal.confirmImport")).toBeInTheDocument();
    });
  });
});
