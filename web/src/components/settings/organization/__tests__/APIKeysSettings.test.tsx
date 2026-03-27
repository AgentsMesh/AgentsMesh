import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  render,
  screen,
  fireEvent,
  act,
  cleanup,
} from "@testing-library/react";
import { APIKeysSettings } from "../APIKeysSettings";
import type { APIKeyData, UpdateAPIKeyRequest } from "@/lib/api/apikey";

// Mock the apiKeyApi module
const mockList = vi.fn();
const mockCreate = vi.fn();
const mockUpdate = vi.fn();
const mockRevoke = vi.fn();

vi.mock("@/lib/api/apikey", () => ({
  apiKeyApi: {
    list: (...args: unknown[]) => mockList(...args),
    create: (...args: unknown[]) => mockCreate(...args),
    update: (...args: unknown[]) => mockUpdate(...args),
    delete: vi.fn(),
    revoke: (...args: unknown[]) => mockRevoke(...args),
  },
}));

// Mock the confirm dialog hook to auto-resolve
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

// Mock sub-dialog components to simplify testing the parent.
// These mocks honor the `open` prop to control visibility, matching real behavior.
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
  CreateAPIKeyDialog: ({
    open,
    onOpenChange,
    onCreate,
  }: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onCreate: (data: {
      name: string;
      scopes: string[];
    }) => Promise<void>;
    t: unknown;
  }) => {
    if (!open) return null;
    return (
      <div data-testid="create-dialog">
        <button
          data-testid="create-dialog-submit"
          onClick={() =>
            onCreate({ name: "New Key", scopes: ["pods:read"] })
          }
        >
          Submit
        </button>
        <button
          data-testid="create-dialog-close"
          onClick={() => onOpenChange(false)}
        >
          Close
        </button>
      </div>
    );
  },
  APIKeySecretDialog: ({
    rawKey,
    open,
    onOpenChange,
  }: {
    rawKey: string;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    t: unknown;
  }) => {
    if (!open) return null;
    return (
      <div data-testid="secret-dialog">
        <span data-testid="raw-key">{rawKey}</span>
        <button
          data-testid="secret-dialog-close"
          onClick={() => onOpenChange(false)}
        >
          Done
        </button>
      </div>
    );
  },
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

const mockT = vi.fn(
  (key: string) => key
);

// Helper to render the component and wait for initial load (useEffect + fetch)
async function renderAndWaitForLoad(): Promise<ReturnType<typeof render>> {
  let result: ReturnType<typeof render>;
  await act(async () => {
    result = render(<APIKeysSettings t={mockT} />);
  });
  return result!;
}

describe("APIKeysSettings", () => {
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

  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mockList.mockResolvedValue({ api_keys: sampleKeys, total: 2 });
    mockConfirm.mockResolvedValue(true);
  });

  afterEach(() => {
    cleanup();
  });

  describe("loading state", () => {
    it("should show loading state initially", () => {
      mockList.mockReturnValue(new Promise(() => {}));
      render(<APIKeysSettings t={mockT} />);
      expect(
        screen.getByText("settings.apiKeys.loading")
      ).toBeInTheDocument();
    });
  });

  describe("empty state", () => {
    it("should show empty state when no keys exist", async () => {
      mockList.mockResolvedValue({ api_keys: [], total: 0 });
      await renderAndWaitForLoad();
      expect(
        screen.getByText("settings.apiKeys.noKeys")
      ).toBeInTheDocument();
    });

    it("should show empty state when api_keys is null/undefined", async () => {
      mockList.mockResolvedValue({ api_keys: null, total: 0 });
      await renderAndWaitForLoad();
      expect(
        screen.getByText("settings.apiKeys.noKeys")
      ).toBeInTheDocument();
    });
  });

  describe("error state", () => {
    it("should show error state when API call fails", async () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      mockList.mockRejectedValue(new Error("Network error"));
      await renderAndWaitForLoad();
      expect(
        screen.getByText("settings.apiKeys.loadError")
      ).toBeInTheDocument();
      consoleSpy.mockRestore();
    });

    it("should dismiss error when dismiss button is clicked", async () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      mockList.mockRejectedValue(new Error("Network error"));
      await renderAndWaitForLoad();
      expect(
        screen.getByText("settings.apiKeys.loadError")
      ).toBeInTheDocument();
      await act(async () => {
        fireEvent.click(screen.getByText("settings.apiKeys.dismiss"));
      });
      expect(
        screen.queryByText("settings.apiKeys.loadError")
      ).not.toBeInTheDocument();
      consoleSpy.mockRestore();
    });
  });

  describe("key list rendering", () => {
    it("should render list of API keys after loading", async () => {
      await renderAndWaitForLoad();
      expect(screen.getByTestId("api-key-card-1")).toBeInTheDocument();
      expect(screen.getByTestId("api-key-card-2")).toBeInTheDocument();
    });

    it("should display correct key names", async () => {
      await renderAndWaitForLoad();
      expect(screen.getByText("CI/CD Key")).toBeInTheDocument();
      expect(screen.getByText("Monitoring Key")).toBeInTheDocument();
    });
  });

  describe("header rendering", () => {
    it("should display title and description", () => {
      render(<APIKeysSettings t={mockT} />);
      expect(
        screen.getByText("settings.apiKeys.title")
      ).toBeInTheDocument();
      expect(
        screen.getByText("settings.apiKeys.description")
      ).toBeInTheDocument();
    });

    it("should display create button", () => {
      render(<APIKeysSettings t={mockT} />);
      expect(
        screen.getByText("settings.apiKeys.createKey")
      ).toBeInTheDocument();
    });
  });

  describe("translation function", () => {
    it("should call t with correct keys for title and description", () => {
      render(<APIKeysSettings t={mockT} />);
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.title");
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.description");
    });

    it("should call t with correct key for create button", () => {
      render(<APIKeysSettings t={mockT} />);
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.createKey");
    });

    it("should call t with correct key for loading state", () => {
      mockList.mockReturnValue(new Promise(() => {}));
      render(<APIKeysSettings t={mockT} />);
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.loading");
    });

    it("should call t with correct key for empty state", async () => {
      mockList.mockResolvedValue({ api_keys: [], total: 0 });
      await renderAndWaitForLoad();
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.noKeys");
    });

    it("should call t with correct key for error state", async () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      mockList.mockRejectedValue(new Error("fail"));
      await renderAndWaitForLoad();
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.loadError");
      consoleSpy.mockRestore();
    });
  });
});
