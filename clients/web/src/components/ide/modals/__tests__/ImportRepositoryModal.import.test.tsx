import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";
import * as repositoryConnect from "@/lib/api/repositoryConnect";
import {
  setupProviderMocks,
  mockRepositoryCreate,
  mockCreatedRepository,
} from "./ImportRepositoryModal.utils";

const stable = vi.hoisted(() => ({
  org: { id: 1, name: "TestOrg", slug: "test-org" },
  user: { id: 1, email: "u@e.com", username: "u" },
}));

vi.mock("@/lib/api/repositoryConnect", () => ({
  createRepository: vi.fn(),
  fromProtoRepository: vi.fn(),
}));

vi.mock("@/stores/auth", () => ({
  useCurrentOrg: () => stable.org,
  useCurrentUser: () => stable.user,
  useAuthOrganizations: () => [],
  useAuthStore: () => ({ currentOrg: stable.org }),
  useIsAuthenticated: () => true,
  readCurrentUser: () => stable.user,
  readCurrentOrg: () => stable.org,
  readOrganizations: () => [],
}));

import { ImportRepositoryModal } from "../ImportRepositoryModal";

describe("ImportRepositoryModal - Import Actions", () => {
  const mockOnClose = vi.fn();
  const mockOnImported = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    setupProviderMocks();
    vi.mocked(repositoryConnect.createRepository).mockResolvedValue(mockCreatedRepository);
  });

  it("should call createRepository (Connect) when import is clicked", async () => {
    mockRepositoryCreate();

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
      expect(screen.getByRole("button", { name: "Import Repository" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Import Repository" }));

    // Production now calls createRepository (Connect adapter) which encodes
    // the request via protobuf .toBinary() and dispatches over the wasm bridge.
    // We assert the adapter mock received the orgSlug + structured payload,
    // matching the dual-track pattern (see lib/api/__tests__/repositoryConnect.test.ts).
    await waitFor(() => {
      expect(vi.mocked(repositoryConnect.createRepository)).toHaveBeenCalledWith(
        "test-org",
        expect.objectContaining({ provider_type: "github" }),
      );
    });
  });

  it("should call onImported and onClose after successful import", async () => {
    mockRepositoryCreate();

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
      expect(screen.getByRole("button", { name: "Import Repository" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Import Repository" }));

    await waitFor(() => {
      expect(mockOnImported).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  it("should handle import error", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    vi.mocked(repositoryConnect.createRepository).mockRejectedValue(new Error("Import failed"));

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
      expect(screen.getByRole("button", { name: "Import Repository" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Import Repository" }));

    await waitFor(() => {
      expect(screen.getByText(/Failed to import repository/)).toBeInTheDocument();
    });

    consoleSpy.mockRestore();
  });
});
