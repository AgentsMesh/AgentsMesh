import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  act,
  cleanup,
} from "@testing-library/react";
import { APIKeysSettings } from "../APIKeysSettings";
import type { APIKeyData, UpdateAPIKeyRequest } from "@/lib/api/apikey";

// Mock the apiKeyApi module
const mockList = vi.fn();
const mockUpdate = vi.fn();
const mockRevoke = vi.fn();

vi.mock("@/lib/api/apikey", () => ({
  apiKeyApi: {
    list: (...args: unknown[]) => mockList(...args),
    create: vi.fn(),
    update: (...args: unknown[]) => mockUpdate(...args),
    delete: vi.fn(),
    revoke: (...args: unknown[]) => mockRevoke(...args),
  },
}));

const mockConfirm = vi.fn();
vi.mock("@/components/ui/confirm-dialog", () => ({
  ConfirmDialog: () => null,
  useConfirmDialog: () => ({
    dialogProps: {
      open: false,
      onOpenChange: vi.fn(),
      title: "",
      onConfirm: vi.fn(),
    },
    confirm: mockConfirm,
    isOpen: false,
  }),
}));

vi.mock("../apikeys", () => ({
  APIKeyCard: ({
    apiKey,
    onEdit,
    onRevoke,
  }: {
    apiKey: APIKeyData;
    onEdit: (key: APIKeyData) => void;
    onRevoke: (id: number) => void;
    t: unknown;
  }) => (
    <div data-testid={`api-key-card-${apiKey.id}`}>
      <span data-testid="key-name">{apiKey.name}</span>
      <button data-testid={`edit-${apiKey.id}`} onClick={() => onEdit(apiKey)}>
        Edit
      </button>
      <button
        data-testid={`revoke-${apiKey.id}`}
        onClick={() => onRevoke(apiKey.id)}
      >
        Revoke
      </button>
    </div>
  ),
  CreateAPIKeyDialog: () => null,
  APIKeySecretDialog: () => null,
  EditAPIKeyDialog: ({
    apiKey,
    open,
    onOpenChange,
    onSave,
  }: {
    apiKey: APIKeyData;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSave: (id: number, data: UpdateAPIKeyRequest) => Promise<void>;
    t: unknown;
  }) => {
    if (!open) return null;
    return (
      <div data-testid="edit-dialog">
        <span data-testid="editing-key-name">{apiKey.name}</span>
        <button
          data-testid="edit-dialog-save"
          onClick={() => onSave(apiKey.id, { name: "Updated" })}
        >
          Save
        </button>
        <button
          data-testid="edit-dialog-close"
          onClick={() => onOpenChange(false)}
        >
          Close
        </button>
      </div>
    );
  },
}));

const mockT = vi.fn((key: string) => key);

async function renderAndWaitForLoad(): Promise<ReturnType<typeof render>> {
  let result: ReturnType<typeof render>;
  await act(async () => {
    result = render(<APIKeysSettings t={mockT} />);
  });
  return result!;
}

const sampleKeys: APIKeyData[] = [
  {
    id: 1,
    organization_id: 10,
    name: "CI/CD Key",
    key_prefix: "am_ci",
    scopes: ["pods:read", "pods:write"],
    is_enabled: true,
    created_by: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
  {
    id: 2,
    organization_id: 10,
    name: "Monitoring Key",
    key_prefix: "am_mon",
    scopes: ["tickets:read"],
    is_enabled: false,
    created_by: 1,
    created_at: "2024-02-01T00:00:00Z",
    updated_at: "2024-02-01T00:00:00Z",
  },
];

describe("APIKeysSettings - edit & revoke flows", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mockList.mockResolvedValue({ api_keys: sampleKeys, total: 2 });
    mockConfirm.mockResolvedValue(true);
  });

  afterEach(() => {
    cleanup();
  });

  describe("edit flow", () => {
    it("should open edit dialog when edit button is clicked", async () => {
      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("edit-1"));
      });

      expect(screen.getByTestId("edit-dialog")).toBeInTheDocument();
      expect(screen.getByTestId("editing-key-name")).toHaveTextContent(
        "CI/CD Key"
      );
    });

    it("should close edit dialog when close is clicked", async () => {
      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("edit-1"));
      });

      expect(screen.getByTestId("edit-dialog")).toBeInTheDocument();

      await act(async () => {
        fireEvent.click(screen.getByTestId("edit-dialog-close"));
      });

      expect(
        screen.queryByTestId("edit-dialog")
      ).not.toBeInTheDocument();
    });

    it("should refresh key list after saving edit", async () => {
      mockUpdate.mockResolvedValue({
        api_key: { id: 1, name: "Updated" },
      });

      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("edit-1"));
      });

      const initialCallCount = mockList.mock.calls.length;

      await act(async () => {
        fireEvent.click(screen.getByTestId("edit-dialog-save"));
      });

      await waitFor(() => {
        expect(mockList).toHaveBeenCalledTimes(initialCallCount + 1);
      });
    });
  });

  describe("revoke flow", () => {
    it("should call confirm dialog when revoke is clicked", async () => {
      mockConfirm.mockResolvedValue(false);

      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("revoke-1"));
      });

      expect(mockConfirm).toHaveBeenCalledWith(
        expect.objectContaining({
          title: "settings.apiKeys.revokeDialog.title",
          description: "settings.apiKeys.revokeDialog.description",
          variant: "destructive",
        })
      );
    });

    it("should call revoke API when confirmed", async () => {
      mockConfirm.mockResolvedValue(true);
      mockRevoke.mockResolvedValue({ message: "Revoked" });

      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("revoke-1"));
      });

      await waitFor(() => {
        expect(mockRevoke).toHaveBeenCalledWith(1);
      });
    });

    it("should NOT call revoke API when cancelled", async () => {
      mockConfirm.mockResolvedValue(false);

      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("revoke-1"));
      });

      expect(mockRevoke).not.toHaveBeenCalled();
    });

    it("should refresh key list after successful revoke", async () => {
      mockConfirm.mockResolvedValue(true);
      mockRevoke.mockResolvedValue({ message: "Revoked" });

      await renderAndWaitForLoad();

      const callCountBefore = mockList.mock.calls.length;

      await act(async () => {
        fireEvent.click(screen.getByTestId("revoke-1"));
      });

      await waitFor(() => {
        expect(mockList).toHaveBeenCalledTimes(callCountBefore + 1);
      });
    });

    it("should handle revoke API failure gracefully", async () => {
      mockConfirm.mockResolvedValue(true);
      mockRevoke.mockRejectedValue(new Error("Revoke failed"));
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      await renderAndWaitForLoad();

      await act(async () => {
        fireEvent.click(screen.getByTestId("revoke-1"));
      });

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith(
          "Failed to revoke API key:",
          expect.any(Error)
        );
      });

      consoleSpy.mockRestore();
    });
  });
});
