import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { APIKeyCard } from "../APIKeyCard";
import type { APIKeyData } from "@/lib/api/apikey";

// Create a simple translation function that returns the key
// but also handles parameterized translations
const mockT = vi.fn(
  (key: string, params?: Record<string, string | number>) => {
    if (params) {
      let result = key;
      for (const [k, v] of Object.entries(params)) {
        result = `${result} [${k}=${v}]`;
      }
      return result;
    }
    return key;
  }
);

describe("APIKeyCard", () => {
  const baseApiKey: APIKeyData = {
    id: 1,
    organization_id: 10,
    name: "Production Key",
    key_prefix: "am_prod",
    scopes: ["pods:read", "tickets:write"],
    is_enabled: true,
    created_by: 1,
    created_at: "2024-01-15T10:00:00Z",
    updated_at: "2024-06-01T12:00:00Z",
  };

  let mockOnEdit: ReturnType<typeof vi.fn<(apiKey: APIKeyData) => void>>;
  let mockOnRevoke: ReturnType<typeof vi.fn<(id: number) => void>>;

  beforeEach(() => {
    vi.clearAllMocks();
    mockOnEdit = vi.fn<(apiKey: APIKeyData) => void>();
    mockOnRevoke = vi.fn<(id: number) => void>();
  });

  function renderCard(apiKey: APIKeyData = baseApiKey) {
    return render(
      <APIKeyCard
        apiKey={apiKey}
        onEdit={mockOnEdit}
        onRevoke={mockOnRevoke}
        t={mockT}
      />
    );
  }

  describe("rendering", () => {
    it("should render the key name", () => {
      renderCard();
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    it("should render the key prefix with ellipsis", () => {
      renderCard();
      expect(screen.getByText("am_prod...")).toBeInTheDocument();
    });

    it("should render scopes", () => {
      renderCard();
      expect(screen.getByText("pods:read")).toBeInTheDocument();
      expect(screen.getByText("tickets:write")).toBeInTheDocument();
    });

    it("should render multiple scopes from a different key", () => {
      renderCard({
        ...baseApiKey,
        scopes: ["channels:read", "channels:write", "pods:write"],
      });
      expect(screen.getByText("channels:read")).toBeInTheDocument();
      expect(screen.getByText("channels:write")).toBeInTheDocument();
      expect(screen.getByText("pods:write")).toBeInTheDocument();
    });
  });

  describe("status display", () => {
    it('should show "Enabled" status badge for an enabled key without expiration', () => {
      renderCard();
      expect(
        screen.getByText("settings.apiKeys.enabled")
      ).toBeInTheDocument();
    });

    it('should show "Disabled" status badge for a disabled key', () => {
      renderCard({
        ...baseApiKey,
        is_enabled: false,
      });
      expect(
        screen.getByText("settings.apiKeys.disabled")
      ).toBeInTheDocument();
    });

    it('should show "Expired" status badge for an enabled but expired key', () => {
      renderCard({
        ...baseApiKey,
        is_enabled: true,
        expires_at: "2020-01-01T00:00:00Z", // past date
      });
      expect(
        screen.getByText("settings.apiKeys.expired")
      ).toBeInTheDocument();
    });

    it('should show "Enabled" for an enabled key that has not expired yet', () => {
      renderCard({
        ...baseApiKey,
        is_enabled: true,
        expires_at: "2099-12-31T23:59:59Z", // future date
      });
      expect(
        screen.getByText("settings.apiKeys.enabled")
      ).toBeInTheDocument();
    });
  });

  describe("expiration info", () => {
    it("should show 'never expires' when expires_at is not set", () => {
      renderCard();
      expect(
        screen.getByText("settings.apiKeys.neverExpires")
      ).toBeInTheDocument();
    });

    it("should show expiration date when expires_at is set", () => {
      renderCard({
        ...baseApiKey,
        expires_at: "2025-06-30T00:00:00Z",
      });
      // The translation key is called with a date parameter
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.expiresAt",
        expect.objectContaining({
          date: expect.any(String),
        })
      );
    });
  });

  describe("last used display", () => {
    it('should show "Never used" when last_used_at is null/undefined', () => {
      renderCard();
      // formatRelativeTime(undefined, t) calls t("settings.apiKeys.neverUsed")
      // Then the result is passed to t("settings.apiKeys.lastUsed", { time: ... })
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.neverUsed");
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.lastUsed",
        expect.objectContaining({
          time: "settings.apiKeys.neverUsed",
        })
      );
    });

    it("should show relative time for recent usage (minutes ago)", () => {
      // Set to 30 minutes ago
      const thirtyMinAgo = new Date(Date.now() - 30 * 60 * 1000).toISOString();
      renderCard({
        ...baseApiKey,
        last_used_at: thirtyMinAgo,
      });
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.minutesAgo",
        expect.objectContaining({
          count: 30,
        })
      );
    });

    it("should show 'just now' for very recent usage", () => {
      const justNow = new Date(Date.now() - 5 * 1000).toISOString(); // 5 seconds ago
      renderCard({
        ...baseApiKey,
        last_used_at: justNow,
      });
      expect(mockT).toHaveBeenCalledWith("settings.apiKeys.justNow");
    });

    it("should show hours ago for recent usage", () => {
      const twoHoursAgo = new Date(
        Date.now() - 2 * 60 * 60 * 1000
      ).toISOString();
      renderCard({
        ...baseApiKey,
        last_used_at: twoHoursAgo,
      });
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.hoursAgo",
        expect.objectContaining({
          count: 2,
        })
      );
    });

    it("should show days ago for old usage", () => {
      // 3 days ago
      const threeDaysAgo = new Date(
        Date.now() - 3 * 24 * 60 * 60 * 1000
      ).toISOString();
      renderCard({
        ...baseApiKey,
        last_used_at: threeDaysAgo,
      });
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.daysAgo",
        expect.objectContaining({
          count: 3,
        })
      );
    });

    it("should pass formatted time to lastUsed translation", () => {
      const twoHoursAgo = new Date(
        Date.now() - 2 * 60 * 60 * 1000
      ).toISOString();
      renderCard({
        ...baseApiKey,
        last_used_at: twoHoursAgo,
      });
      // The result of t("settings.apiKeys.hoursAgo", { count: 2 }) is passed as time param
      expect(mockT).toHaveBeenCalledWith(
        "settings.apiKeys.lastUsed",
        expect.objectContaining({
          time: expect.stringContaining("settings.apiKeys.hoursAgo"),
        })
      );
    });
  });

  describe("button interactions", () => {
    it("should call onEdit with the apiKey when edit button is clicked", () => {
      renderCard();
      const editButton = screen.getByLabelText("settings.apiKeys.editDialog.title");
      fireEvent.click(editButton);

      expect(mockOnEdit).toHaveBeenCalledTimes(1);
      expect(mockOnEdit).toHaveBeenCalledWith(baseApiKey);
    });

    it("should call onRevoke with the key ID when revoke button is clicked on an enabled key", () => {
      renderCard();
      const revokeButton = screen.getByLabelText("settings.apiKeys.revokeDialog.title");
      fireEvent.click(revokeButton);

      expect(mockOnRevoke).toHaveBeenCalledTimes(1);
      expect(mockOnRevoke).toHaveBeenCalledWith(1);
    });

    it("should not show revoke button for a disabled key", () => {
      renderCard({
        ...baseApiKey,
        is_enabled: false,
      });
      expect(
        screen.queryByLabelText("settings.apiKeys.revokeDialog.title")
      ).not.toBeInTheDocument();
    });

    it("should show both edit and revoke buttons for an enabled key", () => {
      renderCard();
      expect(
        screen.getByLabelText("settings.apiKeys.editDialog.title")
      ).toBeInTheDocument();
      expect(
        screen.getByLabelText("settings.apiKeys.revokeDialog.title")
      ).toBeInTheDocument();
    });

    it("should show only edit button for a disabled key", () => {
      renderCard({
        ...baseApiKey,
        is_enabled: false,
      });
      expect(
        screen.getByLabelText("settings.apiKeys.editDialog.title")
      ).toBeInTheDocument();
      expect(
        screen.queryByLabelText("settings.apiKeys.revokeDialog.title")
      ).not.toBeInTheDocument();
    });
  });

  describe("description field", () => {
    it("should handle key with description", () => {
      renderCard({
        ...baseApiKey,
        description: "Used for production deployment",
      });
      // Description is not displayed in the card, but ensure it doesn't break rendering
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    it("should handle key without description", () => {
      renderCard({
        ...baseApiKey,
        description: undefined,
      });
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });
  });

  describe("edge cases", () => {
    it("should handle empty scopes array", () => {
      renderCard({
        ...baseApiKey,
        scopes: [],
      });
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    it("should handle single scope", () => {
      renderCard({
        ...baseApiKey,
        scopes: ["pods:read"],
      });
      expect(screen.getByText("pods:read")).toBeInTheDocument();
    });

    it("should render correctly with many scopes", () => {
      renderCard({
        ...baseApiKey,
        scopes: [
          "pods:read",
          "pods:write",
          "tickets:read",
          "tickets:write",
          "channels:read",
          "channels:write",
        ],
      });
      expect(screen.getByText("pods:read")).toBeInTheDocument();
      expect(screen.getByText("channels:write")).toBeInTheDocument();
    });

    it("should show revoke button for enabled expired key", () => {
      // is_enabled is true but expired - revoke button should still show
      // because the condition is apiKey.is_enabled, not isActive
      renderCard({
        ...baseApiKey,
        is_enabled: true,
        expires_at: "2020-01-01T00:00:00Z",
      });
      expect(
        screen.getByLabelText("settings.apiKeys.revokeDialog.title")
      ).toBeInTheDocument();
    });
  });
});
