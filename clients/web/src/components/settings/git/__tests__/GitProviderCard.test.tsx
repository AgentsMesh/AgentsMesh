import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { GitProviderCard } from "../GitProviderCard";
import type { RepositoryProviderData } from "@/lib/api/userRepositoryProviderTypes";

const mockT = vi.fn((key: string) => key);

const baseProvider: RepositoryProviderData = {
  id: 1,
  user_id: 10,
  provider_type: "github",
  name: "GitHub",
  base_url: "https://github.com",
  has_client_id: false,
  has_bot_token: false,
  has_identity: true,
  is_default: false,
  is_active: true,
  created_at: "2026-05-06T00:00:00Z",
  updated_at: "2026-05-06T00:00:00Z",
};

describe("GitProviderCard", () => {
  let onEdit: ReturnType<typeof vi.fn>;
  let onDelete: ReturnType<typeof vi.fn>;
  let onTestConnection: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    onEdit = vi.fn();
    onDelete = vi.fn();
    onTestConnection = vi.fn();
  });

  function renderCard(provider: RepositoryProviderData = baseProvider) {
    return render(
      <GitProviderCard
        provider={provider}
        onEdit={onEdit}
        onDelete={onDelete}
        onTestConnection={onTestConnection}
        t={mockT}
      />
    );
  }

  describe("disabled badge visibility", () => {
    it("should NOT show disabled badge when is_active=true", () => {
      renderCard({ ...baseProvider, is_active: true });
      expect(
        screen.queryByText("settings.gitSettings.providers.disabled")
      ).not.toBeInTheDocument();
    });

    it("should show disabled badge when is_active=false", () => {
      renderCard({ ...baseProvider, is_active: false });
      expect(
        screen.getByText("settings.gitSettings.providers.disabled")
      ).toBeInTheDocument();
    });
  });

  describe("regression — wasm-core field-stripping bug", () => {
    it("should NOT show disabled badge when is_active is undefined (defensive)", () => {
      renderCard({ ...baseProvider, is_active: undefined as unknown as boolean });
      expect(
        screen.queryByText("settings.gitSettings.providers.disabled")
      ).not.toBeInTheDocument();
    });
  });

  describe("default badge", () => {
    it("should show default badge when is_default=true", () => {
      renderCard({ ...baseProvider, is_default: true });
      expect(
        screen.getByText("settings.gitSettings.providers.default")
      ).toBeInTheDocument();
    });

    it("should not show default badge when is_default=false", () => {
      renderCard({ ...baseProvider, is_default: false });
      expect(
        screen.queryByText("settings.gitSettings.providers.default")
      ).not.toBeInTheDocument();
    });
  });

  describe("provider info rendering", () => {
    it("renders the provider name and base_url", () => {
      renderCard({ ...baseProvider, name: "My GitLab", base_url: "https://gitlab.x" });
      expect(screen.getByText("My GitLab")).toBeInTheDocument();
      expect(screen.getByText("https://gitlab.x")).toBeInTheDocument();
    });
  });

  describe("button interactions", () => {
    it("calls onEdit when settings button is clicked", () => {
      renderCard();
      fireEvent.click(screen.getByTestId("git-provider-edit-button"));
      expect(onEdit).toHaveBeenCalledTimes(1);
    });

    it("calls onTestConnection when test button is clicked", () => {
      renderCard();
      fireEvent.click(screen.getByTitle("settings.gitSettings.providers.test"));
      expect(onTestConnection).toHaveBeenCalledTimes(1);
    });
  });
});

describe("GitProviderCard — visual styling reflects is_active", () => {
  it("applies dimmed style when disabled", () => {
    const { container } = render(
      <GitProviderCard
        provider={{ ...baseProvider, is_active: false }}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onTestConnection={vi.fn()}
        t={mockT}
      />
    );
    expect(container.querySelector(".opacity-60")).toBeInTheDocument();
  });

  it("does NOT apply dimmed style when active", () => {
    const { container } = render(
      <GitProviderCard
        provider={{ ...baseProvider, is_active: true }}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onTestConnection={vi.fn()}
        t={mockT}
      />
    );
    expect(container.querySelector(".opacity-60")).not.toBeInTheDocument();
  });
});
