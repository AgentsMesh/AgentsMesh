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

describe("ImportRepositoryModal - Confirmation Step", () => {
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

  it("should navigate to confirm step after selecting repository", async () => {
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
      expect(screen.getByText("repositories.modal.confirmImport")).toBeInTheDocument();
    });
  });

  it("should show repository details in confirm step", async () => {
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
      expect(screen.getByText("my-project")).toBeInTheDocument();
      expect(screen.getByText("repositories.modal.ticketPrefixOptional")).toBeInTheDocument();
    });
  });

  it("should show visibility options in confirm step", async () => {
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
      expect(screen.getByText("repositories.modal.visibility")).toBeInTheDocument();
      expect(screen.getByText("repositories.modal.organization")).toBeInTheDocument();
      expect(screen.getByText("repositories.modal.privateOnly")).toBeInTheDocument();
    });
  });

  it("should allow setting ticket prefix", async () => {
    vi.mocked(repositoryApi.create).mockResolvedValue(mockCreatedRepository);

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
      expect(screen.getByPlaceholderText("PROJ")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByPlaceholderText("PROJ"), {
      target: { value: "TEST" },
    });

    fireEvent.click(screen.getByText("repositories.modal.importRepository"));

    await waitFor(() => {
      expect(repositoryApi.create).toHaveBeenCalledWith(
        expect.objectContaining({
          ticket_prefix: "TEST",
        })
      );
    });
  });
});
