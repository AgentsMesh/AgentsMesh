import { vi } from "vitest";
import { RepositoryData, WebhookStatus, WebhookSecretResponse } from "@/lib/api";

/**
 * IMPORTANT: Each test file must include its own vi.mock() calls at the module level.
 * Vitest hoists mocks per-file, so mocks defined here won't affect imports in other files.
 *
 * Add this to the TOP of each test file (before any other imports except vitest):
 * ```
 * vi.mock("next-intl", () => ({
 *   useTranslations: () => (key: string) => key,
 * }));
 *
 * vi.mock("@/lib/api", () => ({
 *   repositoryApi: {
 *     getWebhookStatus: () => mockGetWebhookStatus(),
 *     getWebhookSecret: () => mockGetWebhookSecret(),
 *     registerWebhook: () => mockRegisterWebhook(),
 *     deleteWebhook: () => mockDeleteWebhook(),
 *     markWebhookConfigured: () => mockMarkWebhookConfigured(),
 *   },
 * }));
 * ```
 */

// Mock functions for repositoryApi
export const mockGetWebhookStatus = vi.fn();
export const mockGetWebhookSecret = vi.fn();
export const mockRegisterWebhook = vi.fn();
export const mockDeleteWebhook = vi.fn();
export const mockMarkWebhookConfigured = vi.fn();

// Mock clipboard API
export const mockClipboardWriteText = vi.fn();
Object.assign(navigator, {
  clipboard: {
    writeText: mockClipboardWriteText,
  },
});

// Test fixtures
export const mockRepository: RepositoryData = {
  id: 1,
  organization_id: 100,
  provider_type: "gitlab",
  provider_base_url: "https://gitlab.com",
  clone_url: "https://gitlab.com/org/repo.git",
  external_id: "123",
  name: "test-repo",
  slug: "org/test-repo",
  default_branch: "main",
  visibility: "organization",
  is_active: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const registeredStatus: WebhookStatus = {
  registered: true,
  webhook_id: "wh_123",
  webhook_url: "https://example.com/webhooks/org/gitlab/1",
  events: ["merge_request", "pipeline"],
  is_active: true,
  needs_manual_setup: false,
  registered_at: "2025-01-01T00:00:00Z",
};

export const manualSetupStatus: WebhookStatus = {
  registered: true,
  webhook_url: "https://example.com/webhooks/org/gitlab/1",
  events: ["merge_request", "pipeline"],
  is_active: false,
  needs_manual_setup: true,
  last_error: "OAuth token not available",
};

export const notRegisteredStatus: WebhookStatus = {
  registered: false,
  is_active: false,
  needs_manual_setup: false,
};

export const secretResponse: WebhookSecretResponse = {
  webhook_url: "https://example.com/webhooks/org/gitlab/1",
  webhook_secret: "super_secret_value",
  events: ["merge_request", "pipeline"],
};

export function resetAllMocks() {
  mockGetWebhookStatus.mockReset();
  mockGetWebhookSecret.mockReset();
  mockRegisterWebhook.mockReset();
  mockDeleteWebhook.mockReset();
  mockMarkWebhookConfigured.mockReset();
  mockClipboardWriteText.mockReset();
  mockClipboardWriteText.mockResolvedValue(undefined);
}
