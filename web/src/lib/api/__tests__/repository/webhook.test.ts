import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock must be at module level for Vitest hoisting
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => ({
      token: "test-token",
      currentOrg: { slug: "test-org" },
    }),
  },
}));

import { repositoryApi, WebhookStatus, WebhookResult, WebhookSecretResponse } from "../../repository";
import { mockFetch, EXPECTED_API_URL, basePath, setupRepositoryTests } from "./testSetup";

describe("repositoryApi - Webhook Operations", () => {
  beforeEach(() => {
    setupRepositoryTests();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("registerWebhook", () => {
    it("should register a webhook successfully", async () => {
      const mockResult: WebhookResult = {
        repo_id: 1,
        registered: true,
        webhook_id: "wh_123",
        needs_manual_setup: false,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ result: mockResult })),
      });

      const result = await repositoryApi.registerWebhook(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1/webhook`,
        expect.objectContaining({ method: "POST" })
      );
      expect(result.result.registered).toBe(true);
      expect(result.result.webhook_id).toBe("wh_123");
    });

    it("should return needs_manual_setup when auto-registration fails", async () => {
      const mockResult: WebhookResult = {
        repo_id: 1,
        registered: false,
        needs_manual_setup: true,
        manual_webhook_url: "https://example.com/webhooks/org/gitlab/1",
        manual_webhook_secret: "secret123",
        error: "OAuth token not available",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ result: mockResult })),
      });

      const result = await repositoryApi.registerWebhook(1);

      expect(result.result.registered).toBe(false);
      expect(result.result.needs_manual_setup).toBe(true);
      expect(result.result.manual_webhook_url).toBeDefined();
      expect(result.result.manual_webhook_secret).toBeDefined();
    });
  });

  describe("deleteWebhook", () => {
    it("should delete a webhook", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ message: "Webhook deleted" })),
      });

      const result = await repositoryApi.deleteWebhook(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1/webhook`,
        expect.objectContaining({ method: "DELETE" })
      );
      expect(result.message).toBe("Webhook deleted");
    });
  });

  describe("getWebhookStatus", () => {
    it("should get webhook status - registered", async () => {
      const mockStatus: WebhookStatus = {
        registered: true,
        webhook_id: "wh_123",
        webhook_url: "https://example.com/webhooks/org/gitlab/1",
        events: ["merge_request", "pipeline"],
        is_active: true,
        needs_manual_setup: false,
        registered_at: "2025-01-01T00:00:00Z",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ webhook_status: mockStatus })),
      });

      const result = await repositoryApi.getWebhookStatus(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1/webhook/status`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.webhook_status.registered).toBe(true);
      expect(result.webhook_status.is_active).toBe(true);
      expect(result.webhook_status.events).toContain("merge_request");
    });

    it("should get webhook status - needs manual setup", async () => {
      const mockStatus: WebhookStatus = {
        registered: true,
        webhook_url: "https://example.com/webhooks/org/gitlab/1",
        events: ["merge_request", "pipeline"],
        is_active: false,
        needs_manual_setup: true,
        last_error: "OAuth token not available",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ webhook_status: mockStatus })),
      });

      const result = await repositoryApi.getWebhookStatus(1);

      expect(result.webhook_status.needs_manual_setup).toBe(true);
      expect(result.webhook_status.is_active).toBe(false);
      expect(result.webhook_status.last_error).toBeDefined();
    });

    it("should get webhook status - not registered", async () => {
      const mockStatus: WebhookStatus = {
        registered: false,
        is_active: false,
        needs_manual_setup: false,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ webhook_status: mockStatus })),
      });

      const result = await repositoryApi.getWebhookStatus(1);

      expect(result.webhook_status.registered).toBe(false);
    });
  });

  describe("getWebhookSecret", () => {
    it("should get webhook secret for manual setup", async () => {
      const mockSecretResponse: WebhookSecretResponse = {
        webhook_url: "https://example.com/webhooks/org/gitlab/1",
        webhook_secret: "super_secret_value",
        events: ["merge_request", "pipeline"],
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockSecretResponse)),
      });

      const result = await repositoryApi.getWebhookSecret(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1/webhook/secret`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.webhook_url).toBeDefined();
      expect(result.webhook_secret).toBeDefined();
      expect(result.events).toContain("merge_request");
    });
  });

  describe("markWebhookConfigured", () => {
    it("should mark webhook as manually configured", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ message: "Webhook marked as configured" })),
      });

      const result = await repositoryApi.markWebhookConfigured(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}${basePath}/1/webhook/configured`,
        expect.objectContaining({ method: "POST" })
      );
      expect(result.message).toBe("Webhook marked as configured");
    });
  });

});
