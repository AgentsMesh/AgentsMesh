import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { getApiBaseUrl } from "@/lib/env";

const EXPECTED_API_URL = getApiBaseUrl();

import { billingApi, publicBillingApi } from "../billing";
import type {
  SeatUsage,
  CheckoutResponse,
  Invoice,
  DeploymentInfo,
  PublicPricingResponse,
} from "../billing";

// Mock useAuthStore
const mockGetState = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => mockGetState(),
  },
}));

// Mock global fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("billingApi - operations", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetState.mockReturnValue({
      token: "test-token",
      currentOrg: { slug: "test-org" },
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("createCheckout", () => {
    it("should create a checkout session", async () => {
      const mockResponse: CheckoutResponse = {
        order_no: "ORD-123",
        session_id: "sess_123",
        session_url: "https://checkout.stripe.com/session",
        expires_at: "2026-01-20T12:00:00Z",
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockResponse)),
      });

      const result = await billingApi.createCheckout({
        order_type: "subscription",
        plan_name: "pro",
        billing_cycle: "monthly",
        success_url: "https://example.com/success",
        cancel_url: "https://example.com/cancel",
      });

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/checkout`,
        expect.objectContaining({
          method: "POST",
          body: expect.any(String),
        })
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe("getSeatUsage", () => {
    it("should fetch seat usage", async () => {
      const mockUsage: SeatUsage = {
        total_seats: 5,
        used_seats: 3,
        available_seats: 2,
        max_seats: 50,
        can_add_seats: true,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockUsage)),
      });

      const result = await billingApi.getSeatUsage();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/seats`,
        expect.anything()
      );
      expect(result).toEqual(mockUsage);
    });
  });

  describe("checkQuota", () => {
    it("should check quota availability", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ available: true })),
      });

      const result = await billingApi.checkQuota("users", 1);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("resource=users"),
        expect.anything()
      );
      expect(result.available).toBe(true);
    });

    it("should check quota without amount", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ available: true })),
      });

      await billingApi.checkQuota("runners");

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/quota/check?resource=runners`,
        expect.anything()
      );
    });
  });

  describe("listInvoices", () => {
    it("should fetch invoices with default pagination", async () => {
      const mockInvoices: Invoice[] = [
        {
          id: 1,
          organization_id: 1,
          invoice_no: "INV-001",
          amount: 9.9,
          tax_amount: 0,
          total_amount: 9.9,
          currency: "USD",
          status: "paid",
          billing_period_start: "2026-01-01T00:00:00Z",
          billing_period_end: "2026-02-01T00:00:00Z",
          created_at: "2026-01-01T00:00:00Z",
        },
      ];

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ invoices: mockInvoices })),
      });

      const result = await billingApi.listInvoices();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/invoices?limit=20&offset=0`,
        expect.anything()
      );
      expect(result.invoices).toEqual(mockInvoices);
    });

    it("should fetch invoices with custom pagination", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ invoices: [] })),
      });

      await billingApi.listInvoices(10, 20);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/invoices?limit=10&offset=20`,
        expect.anything()
      );
    });
  });

  describe("subscription management", () => {
    it("should request subscription cancellation", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              message: "Subscription will be cancelled at period end",
              current_period_end: "2026-02-01T00:00:00Z",
            })
          ),
      });

      await billingApi.requestCancelSubscription(false);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/subscription/cancel`,
        expect.objectContaining({ method: "POST" })
      );
    });

    it("should reactivate subscription", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ message: "Subscription reactivated" })),
      });

      await billingApi.reactivateSubscription();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/subscription/reactivate`,
        expect.objectContaining({ method: "POST" })
      );
    });

    it("should change billing cycle", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              message: "Billing cycle will change on next renewal",
              current_cycle: "monthly",
              next_cycle: "yearly",
              effective_date: "2026-02-01T00:00:00Z",
            })
          ),
      });

      await billingApi.changeBillingCycle("yearly");

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/subscription/change-cycle`,
        expect.objectContaining({ method: "POST" })
      );
    });

    it("should update auto-renew", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(
            JSON.stringify({ subscription: {}, auto_renew: false })
          ),
      });

      await billingApi.updateAutoRenew(false);

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/subscription/auto-renew`,
        expect.objectContaining({ method: "PUT" })
      );
    });
  });

  describe("getDeploymentInfo", () => {
    it("should fetch deployment info", async () => {
      const mockInfo: DeploymentInfo = {
        deployment_type: "global",
        available_providers: ["stripe"],
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockInfo)),
      });

      const result = await billingApi.getDeploymentInfo();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/deployment`,
        expect.anything()
      );
      expect(result).toEqual(mockInfo);
    });
  });
});

describe("publicBillingApi", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetState.mockReturnValue({
      token: null,
      currentOrg: null,
    });
  });

  describe("getPricing", () => {
    it("should fetch public pricing info", async () => {
      const mockPricing: PublicPricingResponse = {
        deployment_type: "global",
        currency: "USD",
        plans: [
          {
            name: "based",
            display_name: "Based Plan",
            price_monthly: 9.9,
            price_yearly: 99,
            max_users: 1,
            max_runners: 1,
            max_repositories: 5,
            max_concurrent_pods: 5,
          },
          {
            name: "pro",
            display_name: "Pro Plan",
            price_monthly: 39,
            price_yearly: 390,
            max_users: 5,
            max_runners: 10,
            max_repositories: 10,
            max_concurrent_pods: 10,
          },
        ],
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockPricing)),
      });

      const result = await publicBillingApi.getPricing();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/config/pricing`,
        expect.objectContaining({
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
        })
      );
      expect(result).toEqual(mockPricing);
      expect(result.plans.length).toBe(2);
      expect(result.currency).toBe("USD");
    });

    it("should return CNY pricing for CN deployment", async () => {
      const mockPricing: PublicPricingResponse = {
        deployment_type: "cn",
        currency: "CNY",
        plans: [
          {
            name: "based",
            display_name: "Based Plan",
            price_monthly: 69,
            price_yearly: 690,
            max_users: 1,
            max_runners: 1,
            max_repositories: 5,
            max_concurrent_pods: 5,
          },
        ],
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockPricing)),
      });

      const result = await publicBillingApi.getPricing();

      expect(result.currency).toBe("CNY");
      expect(result.deployment_type).toBe("cn");
    });
  });

  describe("getDeploymentInfo", () => {
    it("should fetch public deployment info", async () => {
      const mockInfo: DeploymentInfo = {
        deployment_type: "global",
        available_providers: ["stripe"],
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify(mockInfo)),
      });

      const result = await publicBillingApi.getDeploymentInfo();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/config/deployment`,
        expect.anything()
      );
      expect(result).toEqual(mockInfo);
    });
  });
});
