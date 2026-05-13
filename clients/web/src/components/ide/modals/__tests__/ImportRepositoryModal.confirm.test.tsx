import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";
import * as repositoryConnect from "@/lib/api/repositoryConnect";
import {
  setupProviderMocks,
  mockRepositoryCreate,
  mockCreatedRepository,
} from "./ImportRepositoryModal.utils";

// Stable references so React's useCallback([currentOrg]) doesn't churn.
// vi.hoisted survives vi.mock's factory hoisting.
const stable = vi.hoisted(() => ({
  org: { id: 1, name: "TestOrg", slug: "test-org" },
  user: { id: 1, email: "u@e.com", username: "u" },
}));

// Production calls createRepository (Connect adapter) which encodes proto
// via .toBinary(); mock the adapter so tests assert structured input
// instead of stringified JSON (legacy stableRepoSvc.create lane is gone).
vi.mock("@/lib/api/repositoryConnect", () => ({
  createRepository: vi.fn(),
  fromProtoRepository: vi.fn(),
}));

// useImportWizard.handleImport bails on !currentOrg; provide a non-null org.
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

describe("ImportRepositoryModal - Confirmation Step", () => {
  const mockOnClose = vi.fn();
  const mockOnImported = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    setupProviderMocks();
    vi.mocked(repositoryConnect.createRepository).mockResolvedValue(mockCreatedRepository);
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
      expect(screen.getByText("Confirm Import")).toBeInTheDocument();
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
      expect(screen.getByText(/Ticket Prefix/)).toBeInTheDocument();
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
      expect(screen.getByText("Visibility")).toBeInTheDocument();
      expect(screen.getByText("Organization")).toBeInTheDocument();
      expect(screen.getByText("Private (only you)")).toBeInTheDocument();
    });
  });

  it("should allow setting ticket prefix", async () => {
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
      expect(screen.getByPlaceholderText("PROJ")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByPlaceholderText("PROJ"), {
      target: { value: "TEST" },
    });

    fireEvent.click(screen.getByRole("button", { name: "Import Repository" }));

    await waitFor(() => {
      expect(vi.mocked(repositoryConnect.createRepository)).toHaveBeenCalledWith(
        "test-org",
        expect.objectContaining({ ticket_prefix: "TEST" }),
      );
    });
  });
});
