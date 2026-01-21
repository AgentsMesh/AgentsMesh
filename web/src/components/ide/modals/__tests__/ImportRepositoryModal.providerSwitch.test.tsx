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

describe("ImportRepositoryModal - Provider Type Switching", () => {
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

  it("should change base URL when provider type is changed to gitlab", async () => {
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

    // Find and change the provider type select
    const providerSelect = document.querySelector("select") as HTMLSelectElement;
    fireEvent.change(providerSelect, { target: { value: "gitlab" } });

    await waitFor(() => {
      const baseUrlInput = screen.getByPlaceholderText("https://github.com") as HTMLInputElement;
      expect(baseUrlInput.value).toBe("https://gitlab.com");
    });
  });

  it("should change base URL when provider type is changed to gitee", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    const providerSelect = document.querySelector("select") as HTMLSelectElement;
    fireEvent.change(providerSelect, { target: { value: "gitee" } });

    await waitFor(() => {
      const baseUrlInput = screen.getByPlaceholderText("https://github.com") as HTMLInputElement;
      expect(baseUrlInput.value).toBe("https://gitee.com");
    });
  });

  it("should clear base URL when provider type is changed to generic", async () => {
    render(
      <ImportRepositoryModal open={true} onClose={mockOnClose} onImported={mockOnImported} />
    );

    await waitFor(() => {
      expect(screen.getByText("repositories.modal.enterManually")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("repositories.modal.enterManually"));

    const providerSelect = document.querySelector("select") as HTMLSelectElement;
    fireEvent.change(providerSelect, { target: { value: "generic" } });

    await waitFor(() => {
      const baseUrlInput = screen.getByPlaceholderText("https://github.com") as HTMLInputElement;
      expect(baseUrlInput.value).toBe("");
    });
  });
});
